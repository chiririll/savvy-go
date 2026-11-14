export type QueryParamValue =
    | string
    | number
    | boolean
    | Array<string | number>
    | undefined
    | null

export function toQueryString(params?: Record<string, QueryParamValue>): string {
    if (!params) {
        return ''
    }

    const search = new URLSearchParams()

    for (const [key, value] of Object.entries(params)) {
        if (value === undefined || value === null || value === false || value === '') {
            continue
        }

        if (Array.isArray(value)) {
            value.forEach((item) => search.append(`${key}[]`, String(item)))
            continue
        }

        search.set(key, value === true ? 'true' : String(value))
    }

    const query = search.toString()
    return query ? `?${query}` : ''
}
