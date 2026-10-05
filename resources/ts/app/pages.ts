// The pages of the app: the one list the router is built from. It imports nothing,
// so tools outside the app (scripts/screenshots) can read it too.
//
// scope says what a page needs:
//   public - no session
//   space  - the current space's data
//   user   - the signed-in user's own settings
//   admin  - a server administrator
//
// Old addresses that only redirect are not pages; they stay in router.tsx.

export type PageScope = 'public' | 'space' | 'user' | 'admin'

export interface Page {
    /** Absolute path; ":name" segments are parameters. */
    path: string
    scope: PageScope
}

export const pages = {
    login: { path: '/login', scope: 'public' },
    setup: { path: '/setup', scope: 'public' },
    setup2fa: { path: '/setup-2fa', scope: 'public' },
    ssoCallback: { path: '/auth/sso/callback', scope: 'public' },
    setPassword: { path: '/set-password/:token', scope: 'public' },
    invite: { path: '/invite/:token', scope: 'public' },

    dashboard: { path: '/', scope: 'space' },
    transactions: { path: '/transactions', scope: 'space' },
    accounts: { path: '/accounts', scope: 'space' },
    categories: { path: '/categories', scope: 'space' },
    currencies: { path: '/currencies', scope: 'space' },
    budgets: { path: '/budgets', scope: 'space' },
    tags: { path: '/tags', scope: 'space' },
    debts: { path: '/debts', scope: 'space' },
    recurring: { path: '/recurring', scope: 'space' },
    automation: { path: '/automation', scope: 'space' },
    automationLogs: { path: '/automation/:id/logs', scope: 'space' },
    reports: { path: '/reports', scope: 'space' },
    spaceSettings: { path: '/settings/space', scope: 'space' },
    importSettings: { path: '/settings/import', scope: 'space' },
    spaceBackups: { path: '/settings/backups', scope: 'space' },

    userSettings: { path: '/settings/user', scope: 'user' },
    securitySettings: { path: '/settings/security', scope: 'user' },
    apiSettings: { path: '/settings/api', scope: 'user' },

    adminSystem: { path: '/admin/system', scope: 'admin' },
    adminMonitoring: { path: '/admin/monitoring', scope: 'admin' },
    adminSso: { path: '/admin/sso', scope: 'admin' },
    adminSecurity: { path: '/admin/security', scope: 'admin' },
    providerCreate: { path: '/admin/providers/create', scope: 'admin' },
    providerEdit: { path: '/admin/providers/:id/edit', scope: 'admin' },
    adminBackups: { path: '/admin/backups', scope: 'admin' },
    adminUsers: { path: '/admin/users', scope: 'admin' },
    adminSpaces: { path: '/admin/spaces', scope: 'admin' },
} as const satisfies Record<string, Page>

/** A page's path relative to the layout route at "/". */
export const childPath = (page: Page) => page.path.replace(/^\//, '')
