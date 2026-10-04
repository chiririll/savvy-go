/**
 * First path segments of the API that belong to a space; the backend serves
 * them under /spaces/{space}. Keep in sync with dataRoutes in
 * internal/httpserver/server.go.
 */
const SPACE_ROOTS = new Set([
    'currencies',
    'accounts',
    'accounts-balance-history',
    'accounts-balance-comparison',
    'categories',
    'categories-summary',
    'tags',
    'transactions',
    'transactions-summary',
    'transactions-pending-summary',
    'debts',
    'debts-summary',
    'reports',
    'recurring',
    'recurring-upcoming',
    'budgets',
    'automation-rules',
    's3',
])

/** Prefixes a space data path with /spaces/{id}; other paths are unchanged. */
export function spacePath(url: string, spaceId: number | null): string {
    if (!url.startsWith('/') || url.startsWith('/spaces/')) return url
    const root = url.slice(1).split(/[/?]/, 1)[0]
    if (!SPACE_ROOTS.has(root) || spaceId === null) return url
    return `/spaces/${spaceId}${url}`
}
