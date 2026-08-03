import { Loading03Icon } from "@hugeicons/core-free-icons"
import { HugeiconsIcon } from "@hugeicons/react"
import { createFileRoute, Link } from "@tanstack/react-router"
import { useState } from "react"
import { useForm } from "react-hook-form"
import { AuthShell } from "@/components/auth-shell"
import { FormTooltipInput } from "@/components/form-tooltip-input"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { toast } from "@/components/ui/toast"
import { AuthError, forgotPassword } from "@/lib/auth-api"

export const Route = createFileRoute("/forgot-password")({
	component: ForgotPasswordPage,
})

function ForgotPasswordPage() {
	const form = useForm<{ email: string }>({ mode: "onSubmit" })
	const [pending, setPending] = useState(false)
	const [sent, setSent] = useState(false)

	const submit = form.handleSubmit(async ({ email }) => {
		setPending(true)
		try {
			await forgotPassword(email)
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

	return (
		<AuthShell>
			<div className="flex flex-col gap-6">
				<div className="flex flex-col items-center gap-1 text-center">
					<h1 className="font-bold text-2xl">Reset your password</h1>
					<p className="text-balance text-muted-foreground text-sm">
						We’ll send a one-hour reset link if the address has an account.
					</p>
				</div>
				{sent ? (
					<FieldGroup>
						<p className="rounded-md bg-primary/10 p-4 text-sm">
							Check your inbox for the next step.
						</p>
						<Link
							className="text-center text-sm underline underline-offset-4"
							to="/login"
						>
							Back to sign in
						</Link>
					</FieldGroup>
				) : (
					<form onSubmit={submit} noValidate>
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
								<Button
									type="submit"
									disabled={pending}
									aria-label={pending ? "Send reset link pending" : undefined}
								>
									{pending ? (
										<HugeiconsIcon
											icon={Loading03Icon}
											strokeWidth={2}
											className="animate-spin"
											aria-hidden="true"
										/>
									) : (
										"Send reset link"
									)}
								</Button>
							</Field>
						</FieldGroup>
					</form>
				)}
			</div>
		</AuthShell>
	)
}
