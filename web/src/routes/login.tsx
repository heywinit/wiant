import { createFileRoute } from "@tanstack/react-router"
import type { MouseEvent } from "react"
import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { AuthShell } from "@/components/auth-shell"
import { LoginForm } from "@/components/login-form"
import { toast } from "@/components/ui/toast"
import { AuthError, login, oauthURL } from "@/lib/auth-api"
import { showAuthToast } from "@/lib/auth-toast"

export const Route = createFileRoute("/login")({ component: LoginPage })

function LoginPage() {
	const navigate = Route.useNavigate()
	const form = useForm<{ email: string; password: string }>({
		mode: "onSubmit",
	})

	useEffect(() => {
		const authToast = new URLSearchParams(window.location.search).get(
			"authToast"
		)
		if (!authToast) return
		showAuthToast(authToast)
		window.history.replaceState({}, "", "/login")
	}, [])

	const submit = form.handleSubmit(async ({ email, password }) => {
		try {
			await login(email, password)
			await navigate({ to: "/", replace: true })
		} catch (cause) {
			toast.add({
				title: "Sign in failed",
				description:
					cause instanceof AuthError ? cause.message : "Please try again.",
				type: "error",
			})
		}
	})

	function click(event: MouseEvent<HTMLFormElement>) {
		const target = event.target
		if (!(target instanceof HTMLElement)) return
		const link = target.closest("a")
		if (link?.textContent?.includes("Forgot")) {
			event.preventDefault()
			void navigate({ to: "/forgot-password" })
			return
		}
		if (link?.textContent?.includes("Sign up")) {
			event.preventDefault()
			void navigate({ to: "/register" })
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
			<LoginForm
				onSubmit={submit}
				onClickCapture={click}
				submitPending={form.formState.isSubmitting}
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
				})}
				passwordError={form.formState.errors.password?.message}
			/>
		</AuthShell>
	)
}
