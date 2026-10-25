export type LoginLocationState = {
    twoFactorToken?: string
    rememberMe?: boolean
    sessionExpired?: boolean
    from?: string | { pathname?: string; search?: string }
}

const isLocalPath = (path: string) => path.startsWith('/') && !path.startsWith('//')

/** Where to go after signing in: the page that sent us here, never another site. */
export function safeReturnPath(from: LoginLocationState['from']): string {
    if (typeof from === 'string') {
        return isLocalPath(from) ? from : '/'
    }
    const pathname = from?.pathname
    if (typeof pathname === 'string' && isLocalPath(pathname)) {
        return `${pathname}${from?.search ?? ''}`
    }
    return '/'
}
