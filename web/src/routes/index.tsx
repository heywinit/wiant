import { createFileRoute } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { getSession, logout, oauthURL, type Session } from "@/lib/auth-api"

export const Route = createFileRoute("/")({ component: HomePage })

function HomePage() {
	const navigate = Route.useNavigate()
	const [session, setSession] = useState<Session>()
	const [error, setError] = useState<string>()

	useEffect(() => {
		void getSession()
			.then((value) => {
				if (!value.user) void navigate({ to: "/login", replace: true })
				else setSession(value)
			})
			.catch(() => setError("Could not load your session."))
	}, [navigate])

	if (error)
		return (
			<main className="grid min-h-screen place-items-center p-6">
				<p className="text-destructive">{error}</p>
			</main>
		)

	if (!session?.user)
		return (
			<main className="grid min-h-screen place-items-center p-6">
				<p className="text-muted-foreground text-sm">Loading your account…</p>
			</main>
		)

	const providers = new Set(session.providers)

	return (
		<main className="mx-auto min-h-screen w-full max-w-3xl px-6 py-16">
			<header className="flex items-center justify-between">
				<span className="font-semibold text-lg">wiant</span>
				<Button
					variant="outline"
					onClick={() =>
						void logout().then(() => navigate({ to: "/login", replace: true }))
					}
				>
					Sign out
				</Button>
			</header>
			<section className="mt-16">
				<p className="text-muted-foreground text-sm">Signed in as</p>
				<h1 className="mt-1 font-semibold text-3xl tracking-tight">
					{session.user.email}
				</h1>
				<p className="mt-3 text-muted-foreground text-sm">Email verified</p>
			</section>
			<section className="mt-12 rounded-xl border bg-card p-6">
				<h2 className="font-semibold text-lg">Sign-in methods</h2>
				<p className="mt-1 text-muted-foreground text-sm">
					Connect another provider without creating a duplicate account.
				</p>
				<div className="mt-5 grid gap-3 sm:grid-cols-2">
					{(["google", "github"] as const).map((provider) => (
						<Button
							key={provider}
							variant="outline"
							disabled={providers.has(provider)}
							onClick={() => window.location.assign(oauthURL(provider, "link"))}
						>
							{providers.has(provider)
								? `${provider} connected`
								: `Connect ${provider}`}
						</Button>
					))}
				</div>
			</section>
		</main>
	)
}
