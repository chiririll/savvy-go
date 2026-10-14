// Calls of the app's API from the page, as the signed-in user: for what a scenario needs to have
// in place (a user, an invitation) without driving the screens that make it.
import type { Page } from 'playwright'

export async function api<T = unknown>(page: Page, method: string, path: string, body?: unknown): Promise<T> {
    const res = await page.evaluate(
        async ({ method, path, body }) => {
            const csrf = document.cookie.split('; ').find((c) => c.startsWith('svy_csrf='))?.split('=')[1]
            const r = await fetch(`/api${path}`, {
                method,
                credentials: 'include',
                headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': decodeURIComponent(csrf ?? '') },
                body: body === undefined ? undefined : JSON.stringify(body),
            })
            const text = await r.text()
            return { ok: r.ok, status: r.status, text }
        },
        { method, path, body },
    )
    if (!res.ok) throw new Error(`${method} ${path}: ${res.status} ${res.text}`)
    if (!res.text) return undefined as T
    const json = JSON.parse(res.text) as { data?: T }
    return (json.data ?? json) as T
}

/** The id of the space the signed-in user is in (the first). */
export async function spaceId(page: Page): Promise<number> {
    const spaces = await api<{ id: number }[]>(page, 'GET', '/spaces')
    return spaces[0].id
}
