// Who is signed in, and what they may do in the current space, for a story. A story sets it with
// `parameters: { app: { role: 'viewer' } }`; the default is an administrator of an editable space.
import { http, HttpResponse } from 'msw'

import { reply } from '@/test/api-mocks'
import { useAuthStore } from '@/stores/auth'
import { useCurrenciesStore } from '@/stores/currencies'
import { useSpaceStore } from '@/stores/space'
import type { SpaceRole, UserRole } from '@/types'

export type AppParameters = {
    /** The role in the current space; a viewer cannot change its data. */
    role?: SpaceRole
    /** The server role. */
    serverRole?: UserRole
    /** Nobody is signed in. */
    signedOut?: boolean
}

export const space = (role: SpaceRole = 'admin') => ({
    id: 1,
    uuid: '00000000-0000-4000-8000-000000000001',
    name: 'Home',
    emoji: '🏠',
    color: '#2563eb',
    role,
})

/** Sets the stores and returns the handlers; the returned function undoes it. */
export function setUpApp(parameters: AppParameters | undefined, use: (...handlers: never[]) => void) {
    const { role = 'admin', serverRole = 'admin', signedOut = false } = parameters ?? {}

    useSpaceStore.setState({ currentId: 1 })
    useCurrenciesStore.setState({ byCode: {} })
    useAuthStore.setState({
        user: signedOut
            ? null
            : { id: 1, name: 'Kirill', email: 'kirill@example.com', role: serverRole, emailVerified: true },
        isAuthenticated: !signedOut,
        isLoading: false,
    })

    // Any API call a story did not mock fails loudly instead of reaching a real server.
    // The first handler listed wins, so the catch-all goes last.
    use(
        reply('get', '/spaces', [space(role)]) as never,
        http.all('/api/*', ({ request }) =>
            HttpResponse.json({ message: `Unmocked ${request.method} ${new URL(request.url).pathname}` }, { status: 599 }),
        ) as never,
    )

    return () => {
        localStorage.clear()
        sessionStorage.clear()
        useSpaceStore.setState({ currentId: null })
        useAuthStore.getState().clear()
    }
}
