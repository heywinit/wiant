package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-chi/chi/v5"
	"github.com/heywinit/wiant/server/config"
	"github.com/heywinit/wiant/server/internal/httpapi"
	"github.com/heywinit/wiant/server/internal/securetoken"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"
)

const sessionCookie = "wiant_session"
const csrfCookie = "wiant_csrf"

type Handler struct {
	cfg    config.Config
	store  *Store
	mailer Mailer
	logger *slog.Logger
	limits *limiter
}

// NewHandler builds the HTTP boundary around the auth store and delivery services.
func NewHandler(cfg config.Config, store *Store, mailer Mailer, logger *slog.Logger) *Handler {
	return &Handler{cfg: cfg, store: store, mailer: mailer, logger: logger, limits: newLimiter()}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/csrf", h.csrf)

	r.With(h.rateLimit("register", 20, time.Hour), h.mutation).Post("/register", h.register)
	r.With(h.rateLimit("login", 10, time.Minute), h.mutation).Post("/login", h.login)
	r.With(h.mutation).Post("/logout", h.logout)
	r.Get("/session", h.session)

	r.With(h.rateLimit("verify-request", 20, time.Hour), h.mutation).Post("/email/verification/request", h.requestVerification)
	r.With(h.mutation).Post("/email/verification/confirm", h.confirmVerification)
	r.With(h.rateLimit("password-forgot", 20, time.Hour), h.mutation).Post("/password/forgot", h.forgotPassword)
	r.With(h.mutation).Post("/password/reset", h.resetPassword)

	r.With(h.rateLimit("oauth-start", 20, 10*time.Minute)).Get("/oauth/{provider}/start", h.oauthStart)
	r.Get("/oauth/{provider}/callback", h.oauthCallback)

	return r
}

func (h *Handler) rateLimit(scope string, maximum int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !h.limits.allow(scope+":"+requestIP(r), maximum, window) {
				httpapi.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests; try again later", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CORS allows credentialed browser requests only from the configured web origin.
func (h *Handler) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Credentialed CORS cannot safely use a wildcard origin. Only the configured
		// browser application may read API responses or send preflighted mutations.
		if origin == h.cfg.WebOrigin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}

		if r.Method == http.MethodOptions {
			if origin != h.cfg.WebOrigin {
				httpapi.WriteError(w, http.StatusForbidden, "origin_not_allowed", "Origin is not allowed", nil)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// mutation enforces the origin, content type, and CSRF proof required by state-changing endpoints.
func (h *Handler) mutation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Origin validation protects login-like endpoints that do not yet have a
		// session, while the signed double-submit token protects every mutation.
		if r.Header.Get("Origin") != h.cfg.WebOrigin {
			httpapi.WriteError(w, http.StatusForbidden, "origin_not_allowed", "Origin is not allowed", nil)
			return
		}

		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			httpapi.WriteError(w, http.StatusUnsupportedMediaType, "json_required", "Content-Type must be application/json", nil)
			return
		}

		cookie, err := r.Cookie(h.cookieName(csrfCookie))
		if err != nil || !h.validCSRF(cookie.Value, r.Header.Get("X-CSRF-Token")) {
			httpapi.WriteError(w, http.StatusForbidden, "csrf_invalid", "Refresh the page and try again", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) csrf(w http.ResponseWriter, r *http.Request) {
	token, _, err := securetoken.New()
	if err != nil {
		h.internal(w, err)
		return
	}

	signature := h.signCSRF(token)

	// JavaScript receives the plain token in the response and echoes it in a
	// header; the matching signed cookie remains HttpOnly.
	h.setCookie(w, h.cookieName(csrfCookie), token+"."+signature, true, 24*time.Hour)
	httpapi.WriteJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) signCSRF(token string) string {
	mac := hmac.New(sha256.New, []byte(h.cfg.CSRFSecret))
	mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (h *Handler) validCSRF(cookie, token string) bool {
	parts := strings.Split(cookie, ".")
	return len(parts) == 2 && hmac.Equal([]byte(parts[1]), []byte(h.signCSRF(token))) && hmac.Equal([]byte(parts[0]), []byte(token))
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email, Password string }
	if !httpapi.DecodeJSON(w, r, &body) {
		return
	}

	email, err := validateEmail(body.Email)
	if err != nil {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Check the submitted fields", map[string]string{"email": err.Error()})
		return
	}

	hash, err := hashPassword(body.Password)
	if err != nil {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Check the submitted fields", map[string]string{"password": err.Error()})
		return
	}

	user, err := h.store.createPasswordUser(r.Context(), email, hash)
	if err != nil {
		var pgerr *pgconn.PgError

		// Keep the response identical when the address is already registered so
		// this endpoint cannot be used to enumerate accounts.
		if errors.As(err, &pgerr) && pgerr.Code == "23505" {
			httpapi.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "verification_required"})
			return
		}

		h.internal(w, err)
		return
	}

	if err = h.sendVerification(r.Context(), user); err != nil {
		h.logger.Error("send verification email", "error", err, "user_id", user.ID)
	}

	httpapi.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "verification_required"})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email, Password string }
	if !httpapi.DecodeJSON(w, r, &body) {
		return
	}

	if !h.limits.allow("login-email:"+normalizeEmail(body.Email), 10, 10*time.Minute) {
		httpapi.WriteError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests; try again later", nil)
		return
	}

	user, hash, err := h.store.userByEmail(r.Context(), body.Email)
	if err != nil || hash == "" || !verifyPassword(hash, body.Password) {
		httpapi.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect", nil)
		return
	}

	if user.EmailVerifiedAt == nil {
		httpapi.WriteError(w, http.StatusForbidden, "email_not_verified", "Verify your email before signing in", nil)
		return
	}

	h.issueSession(w, r, user)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(h.cookieName(sessionCookie)); err == nil {
		_ = h.store.deleteSession(r.Context(), cookie.Value)
	}
	h.clearCookie(w, h.cookieName(sessionCookie), true)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	user, providers, err := h.currentSession(r)
	if errors.Is(err, errNotFound) {
		httpapi.WriteJSON(w, http.StatusOK, SessionResponse{Providers: []string{}})
		return
	}
	if err != nil {
		h.internal(w, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, SessionResponse{User: &user, Providers: providers})
}

func (h *Handler) requestVerification(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email string }
	if !httpapi.DecodeJSON(w, r, &body) {
		return
	}

	user, _, err := h.store.userByEmail(r.Context(), body.Email)
	if err == nil && user.EmailVerifiedAt == nil {
		if err = h.sendVerification(r.Context(), user); err != nil {
			h.logger.Error("send verification email", "error", err, "user_id", user.ID)
		}
	}

	httpapi.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) sendVerification(ctx context.Context, user User) error {
	token, err := h.store.createAuthToken(ctx, user.ID, "verify_email", 24*time.Hour)
	if err != nil {
		return err
	}

	link := h.cfg.WebOrigin + "/verify-email?token=" + url.QueryEscape(token)
	return h.mailer.Send(ctx, user.Email, "Verify your Wiant email", "Verify your email by opening this link:\n\n"+link+"\n\nThis link expires in 24 hours.")
}

func (h *Handler) confirmVerification(w http.ResponseWriter, r *http.Request) {
	var body struct{ Token string }
	if !httpapi.DecodeJSON(w, r, &body) {
		return
	}

	user, err := h.store.consumeAuthToken(r.Context(), body.Token, "verify_email")
	if errors.Is(err, errNotFound) {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "token_invalid", "This verification link is invalid or expired", nil)
		return
	}
	if err != nil {
		h.internal(w, err)
		return
	}

	h.issueSession(w, r, user)
}

func (h *Handler) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var body struct{ Email string }
	if !httpapi.DecodeJSON(w, r, &body) {
		return
	}

	user, _, err := h.store.userByEmail(r.Context(), body.Email)
	if err == nil {
		token, tokenErr := h.store.createAuthToken(r.Context(), user.ID, "reset_password", time.Hour)
		if tokenErr == nil {
			link := h.cfg.WebOrigin + "/reset-password?token=" + url.QueryEscape(token)
			tokenErr = h.mailer.Send(r.Context(), user.Email, "Reset your Wiant password", "Reset your password by opening this link:\n\n"+link+"\n\nThis link expires in one hour.")
		}

		if tokenErr != nil {
			h.logger.Error("send password reset email", "error", tokenErr, "user_id", user.ID)
		}
	}

	httpapi.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	var body struct{ Token, Password string }
	if !httpapi.DecodeJSON(w, r, &body) {
		return
	}

	hash, err := hashPassword(body.Password)
	if err != nil {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "validation_failed", "Check the submitted fields", map[string]string{"password": err.Error()})
		return
	}

	user, err := h.store.resetPassword(r.Context(), body.Token, hash)
	if errors.Is(err, errNotFound) {
		httpapi.WriteError(w, http.StatusUnprocessableEntity, "token_invalid", "This reset link is invalid or expired", nil)
		return
	}
	if err != nil {
		h.internal(w, err)
		return
	}

	h.issueSession(w, r, user)
}

func (h *Handler) oauthStart(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	oauthCfg, err := h.oauthConfig(provider)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	intent := r.URL.Query().Get("intent")
	if intent == "" {
		intent = "login"
	}
	if intent != "login" && intent != "link" {
		httpapi.WriteError(w, http.StatusBadRequest, "intent_invalid", "Unknown OAuth intent", nil)
		return
	}

	returnTo := safeReturnTo(r.URL.Query().Get("returnTo"))
	attempt := oauthAttempt{Provider: provider, Intent: intent, ReturnTo: returnTo}

	// A link attempt is bound to the currently authenticated user in the stored
	// attempt; the callback never chooses an account from provider email alone.
	if intent == "link" {
		user, _, sessionErr := h.currentSession(r)
		if sessionErr != nil {
			http.Redirect(w, r, h.cfg.WebOrigin+"/login?reason=link_requires_login", http.StatusFound)
			return
		}
		attempt.UserID = &user.ID
	}

	state, _, err := securetoken.New()
	if err != nil {
		h.internal(w, err)
		return
	}

	nonce, _, err := securetoken.New()
	if err != nil {
		h.internal(w, err)
		return
	}

	verifier, _, err := securetoken.New()
	if err != nil {
		h.internal(w, err)
		return
	}

	attempt.Nonce = nonce
	attempt.CodeVerifier = verifier

	// Persist the short-lived state before redirecting so callbacks are one-time
	// and do not depend on a browser-readable OAuth cookie.
	if err = h.store.saveOAuthAttempt(r.Context(), state, attempt); err != nil {
		h.internal(w, err)
		return
	}

	options := []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("nonce", nonce)}
	if provider == "github" {
		sum := sha256.Sum256([]byte(verifier))
		options = append(options, oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(sum[:])), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	}

	http.Redirect(w, r, oauthCfg.AuthCodeURL(state, options...), http.StatusFound)
}

func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	// Consuming state atomically rejects replayed, expired, and cross-provider
	// callbacks before any authorization code is exchanged.
	attempt, err := h.store.consumeOAuthAttempt(r.Context(), r.URL.Query().Get("state"))
	if err != nil || attempt.Provider != provider {
		h.oauthRedirect(w, r, "state_invalid", "/")
		return
	}
	if providerError := r.URL.Query().Get("error"); providerError != "" {
		h.oauthRedirect(w, r, "provider_denied", attempt.ReturnTo)
		return
	}

	identity, err := h.fetchIdentity(r.Context(), provider, r.URL.Query().Get("code"), attempt)
	if err != nil {
		h.logger.Warn("OAuth callback failed", "provider", provider, "error", err)
		h.oauthRedirect(w, r, "provider_error", attempt.ReturnTo)
		return
	}

	user, err := h.store.completeOAuth(r.Context(), provider, identity, attempt)
	if errors.Is(err, errAccountExists) {
		h.oauthRedirect(w, r, "account_exists", attempt.ReturnTo)
		return
	}
	if errors.Is(err, errIdentityOwned) {
		h.oauthRedirect(w, r, "identity_owned", attempt.ReturnTo)
		return
	}
	if err != nil {
		h.logger.Error("complete OAuth", "provider", provider, "error", err)
		h.oauthRedirect(w, r, "provider_error", attempt.ReturnTo)
		return
	}

	token, err := h.store.createSession(r.Context(), user.ID)
	if err != nil {
		h.oauthRedirect(w, r, "provider_error", attempt.ReturnTo)
		return
	}

	h.setCookie(w, h.cookieName(sessionCookie), token, true, 30*24*time.Hour)
	h.oauthRedirect(w, r, "success", attempt.ReturnTo)
}

func (h *Handler) oauthConfig(provider string) (*oauth2.Config, error) {
	switch provider {
	case "google":
		if h.cfg.GoogleClientID == "" || h.cfg.GoogleSecret == "" {
			return nil, errors.New("google OAuth is not configured")
		}
		return &oauth2.Config{ClientID: h.cfg.GoogleClientID, ClientSecret: h.cfg.GoogleSecret, RedirectURL: h.cfg.APIPublicURL + "/api/v1/auth/oauth/google/callback", Endpoint: endpoints.Google, Scopes: []string{oidc.ScopeOpenID, "email", "profile"}}, nil
	case "github":
		if h.cfg.GitHubClientID == "" || h.cfg.GitHubSecret == "" {
			return nil, errors.New("github OAuth is not configured")
		}
		return &oauth2.Config{ClientID: h.cfg.GitHubClientID, ClientSecret: h.cfg.GitHubSecret, RedirectURL: h.cfg.APIPublicURL + "/api/v1/auth/oauth/github/callback", Endpoint: endpoints.GitHub, Scopes: []string{"read:user", "user:email"}}, nil
	}
	return nil, errors.New("unknown provider")
}

func (h *Handler) fetchIdentity(ctx context.Context, provider, code string, attempt oauthAttempt) (providerIdentity, error) {
	// Provider tokens are used only long enough to establish a stable subject and
	// verified email. They are intentionally not returned to or stored by Wiant.
	cfg, err := h.oauthConfig(provider)
	if err != nil {
		return providerIdentity{}, err
	}

	options := []oauth2.AuthCodeOption{}
	if provider == "github" {
		options = append(options, oauth2.SetAuthURLParam("code_verifier", attempt.CodeVerifier))
	}

	token, err := cfg.Exchange(ctx, code, options...)
	if err != nil {
		return providerIdentity{}, err
	}

	if provider == "google" {
		raw, ok := token.Extra("id_token").(string)
		if !ok {
			return providerIdentity{}, errors.New("missing Google ID token")
		}

		providerClient, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			return providerIdentity{}, err
		}

		verified, err := providerClient.Verifier(&oidc.Config{ClientID: h.cfg.GoogleClientID}).Verify(ctx, raw)
		if err != nil {
			return providerIdentity{}, err
		}

		var claims struct {
			Subject       string `json:"sub"`
			Email         string `json:"email"`
			Nonce         string `json:"nonce"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err = verified.Claims(&claims); err != nil {
			return providerIdentity{}, err
		}
		if claims.Nonce != attempt.Nonce {
			return providerIdentity{}, errors.New("invalid Google nonce")
		}

		return providerIdentity{Subject: claims.Subject, Email: claims.Email, EmailVerified: claims.EmailVerified}, nil
	}

	client := cfg.Client(ctx, token)
	var profile struct {
		ID int64 `json:"id"`
	}

	if err = getGitHubJSON(client, "https://api.github.com/user", &profile); err != nil {
		return providerIdentity{}, err
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err = getGitHubJSON(client, "https://api.github.com/user/emails", &emails); err != nil {
		return providerIdentity{}, err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return providerIdentity{Subject: fmt.Sprint(profile.ID), Email: email.Email, EmailVerified: true}, nil
		}
	}

	return providerIdentity{}, errors.New("GitHub did not return a primary verified email")
}
func getGitHubJSON(client *http.Client, target string, dst any) error {
	req, _ := http.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("provider returned %s", res.Status)
	}

	return json.NewDecoder(res.Body).Decode(dst)
}

func (h *Handler) currentSession(r *http.Request) (User, []string, error) {
	cookie, err := r.Cookie(h.cookieName(sessionCookie))
	if err != nil {
		return User{}, nil, errNotFound
	}

	return h.store.userBySession(r.Context(), cookie.Value)
}

func (h *Handler) issueSession(w http.ResponseWriter, r *http.Request, user User) {
	token, err := h.store.createSession(r.Context(), user.ID)
	if err != nil {
		h.internal(w, err)
		return
	}

	h.setCookie(w, h.cookieName(sessionCookie), token, true, 30*24*time.Hour)
	_, providers, err := h.store.userBySession(r.Context(), token)
	if err != nil {
		h.internal(w, err)
		return
	}

	httpapi.WriteJSON(w, http.StatusOK, SessionResponse{User: &user, Providers: providers})
}

func (h *Handler) cookieName(base string) string {
	// Production cookies use the __Host- prefix, which requires Secure, Path=/,
	// and no Domain attribute, preventing a sibling host from setting them.
	if h.cfg.CookieSecure {
		return "__Host-" + base
	}
	return base
}

func (h *Handler) setCookie(w http.ResponseWriter, name, value string, httpOnly bool, ttl time.Duration) {
	sameSite := http.SameSiteLaxMode
	if h.cfg.CookieSameSite == "none" {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", MaxAge: int(ttl.Seconds()), Expires: time.Now().Add(ttl), Secure: h.cfg.CookieSecure, HttpOnly: httpOnly, SameSite: sameSite})
}

func (h *Handler) clearCookie(w http.ResponseWriter, name string, httpOnly bool) {
	sameSite := http.SameSiteLaxMode
	if h.cfg.CookieSameSite == "none" {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(w, &http.Cookie{Name: name, Path: "/", MaxAge: -1, Expires: time.Unix(1, 0), Secure: h.cfg.CookieSecure, HttpOnly: httpOnly, SameSite: sameSite})
}

func (h *Handler) oauthRedirect(w http.ResponseWriter, r *http.Request, result, returnTo string) {
	target := h.cfg.WebOrigin + "/auth/callback?result=" + url.QueryEscape(result) + "&returnTo=" + url.QueryEscape(safeReturnTo(returnTo))
	http.Redirect(w, r, target, http.StatusFound)
}

func safeReturnTo(value string) string {
	// Only relative application paths are accepted to prevent an OAuth callback
	// from becoming an open redirect.
	if value == "" || !strings.HasPrefix(value, "/") || strings.HasPrefix(value, "//") {
		return "/"
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.IsAbs() {
		return "/"
	}

	return value
}

func (h *Handler) internal(w http.ResponseWriter, err error) {
	h.logger.Error("auth request failed", "error", err)
	httpapi.WriteError(w, http.StatusInternalServerError, "internal_error", "The request could not be completed", nil)
}
