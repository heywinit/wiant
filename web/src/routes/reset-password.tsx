import { createFileRoute } from "@tanstack/react-router"
import { useState } from "react"
import { useForm } from "react-hook-form"
import { AuthShell } from "@/components/auth-shell"
import { FormTooltipInput } from "@/components/form-tooltip-input"
import { Button } from "@/components/ui/button"
import {
	Field,
	FieldError,
	FieldGroup,
	FieldLabel,
} from "@/components/ui/field"
import { toast } from "@/components/ui/toast"
import { AuthError, resetPassword } from "@/lib/auth-api"
export const Route = createFileRoute("/reset-password")({
	validateSearch: (search: Record<string, unknown>) => ({
		token: typeof search.token === "string" ? search.token : "",
	}),
	component: ResetPasswordPage,
})
function ResetPasswordPage() {
	const { token } = Route.useSearch()
	const navigate = Route.useNavigate()
	const form = useForm<{ password: string }>({ mode: "onSubmit" })
	const [pending, setPending] = useState(false)

	const submit = form.handleSubmit(async ({ password }) => {
		setPending(true)
		try {
			await resetPassword(token, password)
			await navigate({ to: "/", replace: true })
		} catch (cause) {
			toast.add({
				title: "Reset failed",
				description:
					cause instanceof AuthError ? cause.message : "Please try again.",
				type: "error",
			})
			setPending(false)
		}
	})

	return (
		<AuthShell>
			<form onSubmit={submit} className="flex flex-col gap-6" noValidate>
				<div className="flex flex-col items-center gap-1 text-center">
					<h1 className="font-bold text-2xl">Choose a new password</h1>
					<p className="text-balance text-muted-foreground text-sm">
						Resetting your password signs out your other sessions.
					</p>
				</div>
				<FieldGroup>
					<FieldError>
						{!token ? "This reset link is missing its token." : undefined}
					</FieldError>
					<Field>
						<FieldLabel htmlFor="password">New password</FieldLabel>
						<FormTooltipInput
							id="password"
							type="password"
							autoComplete="new-password"
							error={form.formState.errors.password?.message}
							{...form.register("password", {
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
						/>
					</Field>
					<Field>
						<Button type="submit" disabled={pending || !token}>
							{pending ? "Working…" : "Reset password"}
						</Button>
					</Field>
				</FieldGroup>
			</form>
		</AuthShell>
	)
}
