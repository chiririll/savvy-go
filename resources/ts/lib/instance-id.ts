/**
 * The id of the server install that served this page, from the SPA shell. It stays the same
 * across restarts and differs on a fresh install, so what the browser kept for another
 * install can be told apart. Empty when the shell has none (static builds, tests).
 */
export function getInstanceId(): string {
    if (typeof document === 'undefined') {
        return ''
    }

    return document.querySelector('meta[name="instance-id"]')?.getAttribute('content')?.trim() ?? ''
}
