import { Loading03Icon } from "@hugeicons/core-free-icons"
import { HugeiconsIcon } from "@hugeicons/react"
import { Link } from "@tanstack/react-router"
import { FormTooltipInput } from "@/components/form-tooltip-input"
import { GitHubLogo } from "@/components/icons/github-logo"
import { GoogleLogo } from "@/components/icons/google-logo"
import { Button } from "@/components/ui/button"
import {
	Field,
	FieldDescription,
	FieldGroup,
	FieldLabel,
	FieldSeparator,
} from "@/components/ui/field"
import { cn } from "@/lib/utils"

export function LoginForm({
	className,
	emailError,
	emailInputProps,
	passwordError,
	passwordInputProps,
	title = "Login to your account",
	description = "Enter your email below to login to your account",
	submitLabel = "Login",
	submitPending = false,
	googleLabel = "Continue with Google",
	githubLabel = "Continue with GitHub",
	showForgotPassword = true,
	footerPrompt = "Don’t have an account?",
	footerLinkLabel = "Sign up",
	footerLinkTo = "/register",
	...props
}: React.ComponentProps<"form"> & {
	emailError?: string
	emailInputProps?: React.ComponentProps<typeof FormTooltipInput>
	passwordError?: string
	passwordInputProps?: React.ComponentProps<typeof FormTooltipInput>
	title?: string
	description?: string
	submitLabel?: string
	submitPending?: boolean
	googleLabel?: string
	githubLabel?: string
	showForgotPassword?: boolean
	footerPrompt?: string
	footerLinkLabel?: string
	footerLinkTo?: "/login" | "/register"
}) {
	return (
		<form className={cn("flex flex-col gap-6", className)} noValidate {...props}>
			<FieldGroup>
				<div className="flex flex-col items-center gap-1 text-center">
					<h1 className="font-bold text-2xl">{title}</h1>
					<p className="text-sm text-balance text-muted-foreground">
						{description}
					</p>
				</div>
				<Field>
					<FieldLabel htmlFor="email">Email</FieldLabel>
					<FormTooltipInput
						id="email"
						type="email"
						placeholder="m@example.com"
						error={emailError}
						{...emailInputProps}
					/>
				</Field>
				<Field>
					<div className="flex items-center">
						<FieldLabel htmlFor="password">Password</FieldLabel>
						{showForgotPassword ? (
							<Link
								to="/forgot-password"
								className="ml-auto text-xs underline-offset-4 hover:underline"
							>
								Forgot your password?
							</Link>
            ) : (
                <span></span>
						)}
					</div>
					<FormTooltipInput
						id="password"
						type="password"
						error={passwordError}
						{...passwordInputProps}
					/>
				</Field>
				<Field>
					<Button
						type="submit"
						disabled={submitPending}
						aria-label={submitPending ? `${submitLabel} pending` : undefined}
					>
						{submitPending ? (
							<HugeiconsIcon
								icon={Loading03Icon}
								strokeWidth={2}
								className="animate-spin"
								aria-hidden="true"
							/>
						) : (
							submitLabel
						)}
					</Button>
				</Field>
				<FieldSeparator>Or continue with</FieldSeparator>
				<Field>
					<Button variant="outline" type="button" data-provider="google">
						<GoogleLogo />
						{googleLabel}
					</Button>
					<Button variant="outline" type="button" data-provider="github">
						<GitHubLogo />
						{githubLabel}
					</Button>
					<FieldDescription className="text-center">
						{footerPrompt}{" "}
						<Link to={footerLinkTo} className="underline underline-offset-4">
							{footerLinkLabel}
						</Link>
					</FieldDescription>
				</Field>
			</FieldGroup>
		</form>
	)
}
