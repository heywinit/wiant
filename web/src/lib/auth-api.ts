import createClient from "openapi-fetch"
import type { components, paths } from "./api-schema"

export type Session = components["schemas"]["Session"]
export type User = components["schemas"]["User"]

const apiBase = "/api/v1"
const client = createClient<paths>({ baseUrl: apiBase, credentials: "include" })
let csrfToken: string | undefined

export class AuthError extends Error {
	constructor(
		public code: string,
		message: string,
		public fields?: Record<string, string>
	) {
		super(message)
	}
}

async function csrf() {
	if (csrfToken) return csrfToken
	const { data, error } = await client.GET("/auth/csrf")
	if (!data) throw authError(error)
	csrfToken = data.token
	return csrfToken
}

function authError(value: unknown) {
	if (value && typeof value === "object" && "message" in value) {
		const error = value as {
			code?: string
			message: string
			fields?: Record<string, string>
		}
		return new AuthError(
			error.code ?? "request_failed",
			error.message,
			error.fields
		)
	}
	return new AuthError("request_failed", "The request could not be completed")
}

async function csrfHeaders() {
	return { "X-CSRF-Token": await csrf() }
}

export async function getSession(): Promise<Session> {
	const { data, error } = await client.GET("/auth/session")
	if (!data) throw authError(error)
	return data
}

export async function register(email: string, password: string) {
	const { data, error } = await client.POST("/auth/register", {
		body: { email, password },
		headers: await csrfHeaders(),
	})
	if (!data) throw authError(error)
	return data
}

export async function login(email: string, password: string): Promise<Session> {
	const { data, error } = await client.POST("/auth/login", {
		body: { email, password },
		headers: await csrfHeaders(),
	})
	if (!data) throw authError(error)
	return data
}

export async function logout() {
	const { error, response } = await client.POST("/auth/logout", {
		headers: { ...(await csrfHeaders()), "Content-Type": "application/json" },
	})
	if (!response.ok) throw authError(error)
}

export async function requestVerification(email: string) {
	const { data, error } = await client.POST(
		"/auth/email/verification/request",
		{
			body: { email },
			headers: await csrfHeaders(),
		}
	)
	if (!data) throw authError(error)
	return data
}

export async function confirmVerification(token: string): Promise<Session> {
	const { data, error } = await client.POST(
		"/auth/email/verification/confirm",
		{
			body: { token },
			headers: await csrfHeaders(),
		}
	)
	if (!data) throw authError(error)
	return data
}

export async function forgotPassword(email: string) {
	const { data, error } = await client.POST("/auth/password/forgot", {
		body: { email },
		headers: await csrfHeaders(),
	})
	if (!data) throw authError(error)
	return data
}

export async function resetPassword(
	token: string,
	password: string
): Promise<Session> {
	const { data, error } = await client.POST("/auth/password/reset", {
		body: { token, password },
		headers: await csrfHeaders(),
	})
	if (!data) throw authError(error)
	return data
}

export function oauthURL(
	provider: "google" | "github",
	intent: "login" | "link" = "login"
) {
	return `${apiBase}/auth/oauth/${provider}/start?intent=${intent}&returnTo=%2F`
}
