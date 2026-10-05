// Pages to capture. Keep in sync with resources/ts/app/router.tsx.
//
// scope:
//   public - no session (the sign-in and invitation pages)
//   space  - data of the current space: captured for every user in each space they belong to
//   user   - the signed-in user's own pages: captured once per user
//   admin  - server administration: captured once per user with the server role "admin"
//
// A ":name" segment is filled in from the manifest (see params in run.ts); a page
// whose parameter has no value is skipped.

export type Scope = 'public' | 'space' | 'user' | 'admin'

export interface Route {
    path: string
    scope: Scope
}

const space = (...paths: string[]): Route[] => paths.map((path) => ({ path, scope: 'space' }))
const user = (...paths: string[]): Route[] => paths.map((path) => ({ path, scope: 'user' }))
const admin = (...paths: string[]): Route[] => paths.map((path) => ({ path, scope: 'admin' }))

export const routes: Route[] = [
    { path: '/login', scope: 'public' },
    { path: '/invite/:inviteToken', scope: 'public' },

    ...space(
        '/',
        '/transactions',
        '/accounts',
        '/categories',
        '/currencies',
        '/budgets',
        '/tags',
        '/debts',
        '/recurring',
        '/automation',
        '/automation/:automationId/logs',
        '/reports',
        '/settings/space',
        '/settings/import',
        '/settings/backups',
    ),

    ...user('/settings/user', '/settings/security', '/settings/api'),

    ...admin(
        '/admin/system',
        '/admin/monitoring',
        '/admin/sso',
        '/admin/security',
        '/admin/backups',
        '/admin/users',
        '/admin/spaces',
    ),
]
