// MSW handlers for stories. The API client sits at /api, and data of a space under /spaces/{id}, so a
// path is matched by its tail: `reply('get', '/currencies', data)` answers /api/currencies and
// /api/spaces/1/currencies.
import { delay, http, HttpResponse } from 'msw'

type Method = 'get' | 'post' | 'put' | 'patch' | 'delete'

const escapeRegExp = (text: string) => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const route = (method: Method, path: string, respond: () => Response | Promise<Response>) =>
    http[method](new RegExp(`/api(/spaces/\\d+)?${escapeRegExp(path)}(\\?.*)?$`), respond)

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
