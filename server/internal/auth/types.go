package auth

import "time"

type User struct {
	ID              int64      `json:"id"`
	Email           string     `json:"email"`
	EmailVerifiedAt *time.Time `json:"emailVerifiedAt"`
}

type SessionResponse struct {
	User      *User    `json:"user"`
	Providers []string `json:"providers"`
}

type oauthAttempt struct {
	Provider     string
	CodeVerifier string
	Nonce        string
	Intent       string
	UserID       *int64
	ReturnTo     string
}

type providerIdentity struct {
	Subject, Email string
	EmailVerified  bool
}
