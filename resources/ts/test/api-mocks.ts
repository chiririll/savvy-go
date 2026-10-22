// MSW handlers for stories. The API client sits at /api, and data of a space under /spaces/{id}, so a
// path is matched by its tail: `reply('get', '/currencies', data)` answers /api/currencies and
// /api/spaces/1/currencies.
import { delay, http, HttpResponse } from 'msw'

import type { Settings } from '@/types'

import { catalog, currencies, settings } from './fixtures'

type Method = 'get' | 'post' | 'put' | 'patch' | 'delete'

const escapeRegExp = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const route = (method: Method, path: string, respond: (request: Request) => Response | Promise<Response>) =>
    http[method](new RegExp(`/api(/spaces/\\d+)?${escapeRegExp(path)}(\\?.*)?$`), ({ request }) => respond(request))

/** Answers with `data` wrapped the way the API wraps it. */
export const reply = (method: Method, path: string, data: unknown, status = 200) =>
    route(method, path, () => HttpResponse.json({ data }, { status }))

/** Answers with an API error. */
export const fail = (method: Method, path: string, message = 'Something went wrong', status = 500) =>
    route(method, path, () => HttpResponse.json({ message }, { status }))

/** Never answers, which keeps a component in its loading state. */
export const hang = (method: Method, path: string) =>
    route(method, path, async () => {
        await delay('infinite')
        return HttpResponse.json({})
    })

/** The currency list and catalog that any currency picker reads. */
export const currencyApi = () => [
    reply('get', '/currencies', currencies),
    reply('get', '/currencies/catalog', catalog),
]

/** Instance and space settings, with the given values changed. */
export const settingsApi = (overrides: Partial<Settings> = {}) => [reply('get', '/settings', { ...settings, ...overrides })]

/**
 * Like `reply`, and keeps the JSON body of every request it gets, to check what the page sent.
 * `wrapped: false` answers with `data` as it is, like `raw`.
 */
export function spy(method: Method, path: string, data: unknown = {}, status = 200, wrapped = true) {
    const bodies: unknown[] = []
    const handler = route(method, path, async (request) => {
        bodies.push(await request.clone().json().catch(() => null))
        return HttpResponse.json((wrapped ? { data } : data) as never, { status })
    })
    return { handler, bodies }
}

/** Answers after a pause, to make a request arrive after the others. */
export const slowly = (method: Method, path: string, data: unknown, ms: number) =>
    route(method, path, async () => {
        await delay(ms)
        return HttpResponse.json({ data })
    })

/** Answers with `body` as it is, for the endpoints whose JSON is not wrapped in `data`. */
export const raw = (method: Method, path: string, body: unknown, status = 200) =>
    route(method, path, () => HttpResponse.json(body as never, { status }))

/** The accounts list with the totals the header and dashboard read. */
export const accountsWithSummary = (data: unknown[], summary: Record<string, unknown>) =>
    raw('get', '/accounts', { data, summary })
