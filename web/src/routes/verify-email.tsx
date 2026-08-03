import { createFileRoute } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { AuthShell } from "@/components/auth-shell"
import { FormTooltipInput } from "@/components/form-tooltip-input"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { toast } from "@/components/ui/toast"
import {
	AuthError,
	confirmVerification,
	requestVerification,
} from "@/lib/auth-api"

export const Route = createFileRoute("/verify-email")({
	validateSearch: (search: Record<string, unknown>) => ({
		token: typeof search.token === "string" ? search.token : "",
	}),
	component: VerifyEmailPage,
})

function VerifyEmailPage() {
	const { token } = Route.useSearch()
	const navigate = Route.useNavigate()
	const form = useForm<{ email: string }>({ mode: "onSubmit" })
	const [verificationState, setVerificationState] = useState<
		"verifying" | "failed"
	>(token ? "verifying" : "failed")
	const [pending, setPending] = useState(false)
	const [sent, setSent] = useState(false)

	useEffect(() => {
		if (!token) return
		const minimumDwell = new Promise((resolve) => setTimeout(resolve, 1100))
		void Promise.all([confirmVerification(token), minimumDwell])
			.then(() => navigate({ to: "/", replace: true }))
			.catch((cause) => {
				toast.add({
					title: "Verification failed",
					description:
						cause instanceof AuthError
							? cause.message
							: "Please request a new link.",
					type: "error",
				})
				setVerificationState("failed")
			})
	}, [navigate, token])

	const resend = form.handleSubmit(async ({ email }) => {
		setPending(true)
		try {
			await requestVerification(email)
			setSent(true)
		} catch (cause) {
			toast.add({
				title: "Request failed",
				description:
					cause instanceof AuthError ? cause.message : "Please try again.",
				type: "error",
			})
		} finally {
			setPending(false)
		}
	})

	const isVerifying = token && verificationState === "verifying"

	return (
		<AuthShell>
			<div className="flex flex-col gap-6">
				<div className="flex flex-col items-center gap-1 text-center">
					<h1 className="font-bold text-2xl">
						{isVerifying ? "Verifying your email" : "Resend verification"}
					</h1>
					<p className="text-balance text-muted-foreground text-sm">
						{isVerifying
							? "Hold tight while we confirm your verification link."
							: "Request a fresh 24-hour verification link."}
					</p>
				</div>
				<FieldGroup>
					{isVerifying && (
						<p className="min-h-10 text-center text-muted-foreground text-sm">
							Checking your verification link…
						</p>
					)}
					{!isVerifying &&
						(sent ? (
							<p className="rounded-md bg-primary/10 p-4 text-sm">
								If the account needs verification, a new link is on its way.
							</p>
						) : (
							<form onSubmit={resend} noValidate>
								<FieldGroup>
									<Field>
										<FieldLabel htmlFor="email">Email</FieldLabel>
										<FormTooltipInput
											id="email"
											type="email"
											autoComplete="email"
											error={form.formState.errors.email?.message}
											{...form.register("email", {
												required: "Email is required",
												pattern: {
													value: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
													message: "Enter a valid email address",
												},
											})}
										/>
									</Field>
									<Field>
										<Button type="submit" disabled={pending}>
											{pending ? "Working…" : "Resend verification"}
										</Button>
									</Field>
								</FieldGroup>
							</form>
						))}
				</FieldGroup>
			</div>
		</AuthShell>
	)
}
