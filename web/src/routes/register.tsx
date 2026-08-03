import { createFileRoute, Link } from "@tanstack/react-router"
import { type MouseEvent, useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { AuthShell } from "@/components/auth-shell"
import { LoginForm } from "@/components/login-form"
import { toast } from "@/components/ui/toast"
import { AuthError, oauthURL, register } from "@/lib/auth-api"
import { showAuthToast } from "@/lib/auth-toast"

export const Route = createFileRoute("/register")({ component: RegisterPage })
function RegisterPage() {
	const form = useForm<{ email: string; password: string }>({
		mode: "onSubmit",
	})
	const [pending, setPending] = useState(false)
	const [sent, setSent] = useState(false)

	useEffect(() => {
		const authToast = new URLSearchParams(window.location.search).get(
			"authToast"
		)
		if (!authToast) return
		showAuthToast(authToast)
		window.history.replaceState({}, "", "/register")
	}, [])

	const submit = form.handleSubmit(async ({ email, password }) => {
		setPending(true)
		try {
			await register(email, password)
			setSent(true)
		} catch (cause) {
			toast.add({
				title: "Registration failed",
				description:
					cause instanceof AuthError ? cause.message : "Please try again.",
				type: "error",
			})
		} finally {
			setPending(false)
		}
	})

	function click(event: MouseEvent<HTMLFormElement>) {
		const target = event.target
		if (!(target instanceof HTMLElement)) return
		const link = target.closest("a")
		if (link?.textContent?.includes("Forgot")) {
			event.preventDefault()
			return
		}
		if (link?.textContent?.includes("Sign up")) {
			event.preventDefault()
			return
		}
		const button = target.closest("button")
		const provider = button?.dataset.provider
		if (provider === "google" || provider === "github") {
			event.preventDefault()
			window.location.assign(oauthURL(provider, "login"))
		}
	}

	return (
		<AuthShell>
			{sent ? (
				<div className="flex flex-col gap-6">
					<div className="flex flex-col items-center gap-1 text-center">
						<h1 className="font-bold text-2xl">Check your inbox</h1>
						<p className="text-balance text-muted-foreground text-sm">
							Your verification link expires in 24 hours.
						</p>
					</div>
					<p className="rounded-md bg-primary/10 p-4 text-sm">
						Check your inbox for a verification link.
					</p>
					<Link
						className="text-center text-sm underline underline-offset-4"
						to="/login"
					>
						Back to login
					</Link>
				</div>
			) : (
				<>
					<LoginForm
						title="Create your account"
						description="Enter your email below to create your account"
						submitLabel="Create account"
						googleLabel="Continue with Google"
						githubLabel="Continue with GitHub"
						showForgotPassword={false}
						footerPrompt="Already have an account?"
						footerLinkLabel="Login"
						footerLinkTo="/login"
						onSubmit={submit}
						onClickCapture={click}
						emailInputProps={form.register("email", {
							required: "Email is required",
							pattern: {
								value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
								message: "Enter a valid email address",
							},
						})}
						emailError={form.formState.errors.email?.message}
						passwordInputProps={form.register("password", {
							required: "Password is required",
							minLength: {
								value: 8,
								message: "Use at least 8 characters",
							},
							maxLength: {
								value: 128,
								message: "Use 128 characters or fewer",
							},
						})}
						passwordError={form.formState.errors.password?.message}
					/>
					{pending && (
						<p className="mt-4 text-center text-muted-foreground text-sm">
							Working…
						</p>
					)}
				</>
			)}
		</AuthShell>
	)
}
