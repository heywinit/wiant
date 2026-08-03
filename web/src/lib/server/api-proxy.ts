const apiBase = (
	process.env.API_PROXY_URL ??
	process.env.VITE_API_URL ??
	"http://localhost:3001/api/v1"
).replace(/\/$/, "")

export async function proxyAPIRequest(request: Request) {
	const incoming = new URL(request.url)
	const base = new URL(apiBase)

	const prefix = base.pathname.replace(/\/$/, "")
	const suffix = incoming.pathname.startsWith(prefix)
		? incoming.pathname.slice(prefix.length)
		: incoming.pathname

	const target = new URL(base)
	target.pathname = `${prefix}${suffix}`
	target.search = incoming.search

	const requestHeaders = new Headers(request.headers)
	requestHeaders.delete("host")
	requestHeaders.delete("content-length")

	const init: RequestInit = {
		method: request.method,
		headers: requestHeaders,
		redirect: "manual",
	}

	if (request.method !== "GET" && request.method !== "HEAD") {
		init.body = await request.arrayBuffer()
	}

	const response = await fetch(target, init)
	const responseHeaders = new Headers(response.headers)
	const setCookies = getSetCookies(response.headers)
	responseHeaders.delete("set-cookie")

	for (const cookie of setCookies) {
		responseHeaders.append("set-cookie", cookie)
	}

	return new Response(response.body, {
		status: response.status,
		statusText: response.statusText,
		headers: responseHeaders,
	})
}

function getSetCookies(headers: Headers) {
	const value = (
		headers as Headers & { getSetCookie?: () => string[] }
	).getSetCookie?.()
	return value ?? []
}
