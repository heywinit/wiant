import { createFileRoute } from "@tanstack/react-router"
import { proxyAPIRequest } from "@/lib/server/api-proxy"

export const Route = createFileRoute("/api/v1/auth/login")({
	server: { handlers: { POST: ({ request }) => proxyAPIRequest(request) } },
})
