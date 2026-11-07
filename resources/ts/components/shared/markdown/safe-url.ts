const ALLOWED = new Set(['http:', 'https:', 'mailto:', 'tel:'])

/** Keeps web, mail and phone links and relative paths; everything else (`javascript:`, `data:`...) becomes empty. */
export function safeUrl(url: string): string {
    // Browsers ignore control characters and spaces inside the scheme.
    // eslint-disable-next-line no-control-regex
    const clean = url.replace(/[\u0000- ]/g, '')
    const colon = clean.indexOf(':')
    if (colon === -1) return url
    const before = clean.slice(0, colon)
    // A colon after `/`, `?` or `#` belongs to a path, not a scheme.
    if (/[/?#]/.test(before)) return url
    return ALLOWED.has(`${before.toLowerCase()}:`) ? url : ''
}
