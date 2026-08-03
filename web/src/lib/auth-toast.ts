import { toast } from "@/components/ui/toast"

const authToastMessages: Record<
	string,
	{ title: string; description: string; type: "error" | "info" | "success" }
> = {
	account_exists: {
		title: "Account already exists",
		description:
			"That email already has an account. Sign in first, then connect this provider from your home page.",
		type: "error",
	},
	identity_owned: {
		title: "Provider already connected",
		description: "That provider is already connected to another account.",
		type: "error",
	},
	provider_denied: {
		title: "Sign-in cancelled",
		description: "Provider sign-in was cancelled.",
		type: "info",
	},
	state_invalid: {
		title: "Sign-in expired",
		description: "The sign-in attempt expired. Please try again.",
		type: "error",
	},
	provider_error: {
		title: "Provider sign-in failed",
		description: "The provider could not complete sign-in.",
		type: "error",
	},
	session_failed: {
		title: "Session not created",
		description: "The session could not be created. Please try again.",
		type: "error",
	},
	session_load_failed: {
		title: "Session not loaded",
		description: "The session could not be loaded. Please try again.",
		type: "error",
	},
}

export function showAuthToast(key?: string) {
	if (!key) return
	const message = authToastMessages[key] ?? authToastMessages.provider_error
	toast.add({
		id: `auth-${key}`,
		title: message.title,
		description: message.description,
		type: message.type,
	})
}
