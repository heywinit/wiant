import { createFileRoute } from "@tanstack/react-router"
import { useEffect } from "react"
import { getSession } from "@/lib/auth-api"

export const Route = createFileRoute("/auth/callback")({
	validateSearch: (search: Record<string, unknown>) => ({
		result:
			typeof search.result === "string" ? search.result : "provider_error",
		returnTo:
			typeof search.returnTo === "string" &&
			search.returnTo.startsWith("/") &&
			!search.returnTo.startsWith("//")
				? search.returnTo
				: "/",
	}),
	component: OAuthCallbackPage,
})

function OAuthCallbackPage() {
	const { result, returnTo } = Route.useSearch()
	const navigate = Route.useNavigate()

	useEffect(() => {
		const authPage = returnTo === "/register" ? "/register" : "/login"
		const redirectToAuthPage = (authToast: string) => {
			if (authPage === "/register") {
				void navigate({
					to: "/register",
					search: { authToast },
					replace: true,
				})
				return
			}
			void navigate({
				to: "/login",
				search: { authToast },
				replace: true,
			})
		}

		if (result !== "success") {
			redirectToAuthPage(result)
			return
		}

		void getSession()
			.then((session) => {
				if (session.user) {
					const to =
						returnTo === "/login" || returnTo === "/register" ? returnTo : "/"
					void navigate({ to, replace: true })
				} else {
					redirectToAuthPage("session_failed")
				}
			})
			.catch(() => redirectToAuthPage("session_load_failed"))
	}, [navigate, result, returnTo])

	return null
}
