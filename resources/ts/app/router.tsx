import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate, useParams } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { AuthProvider } from '@/components/providers/AuthProvider'
import { useUser } from '@/stores/auth'
import { ErrorBoundary } from '@/components/shared/ErrorBoundary'
import { childPath, pages } from './pages'

// Auth pages
const LoginPage = lazy(() => import('@/pages/auth/login'))
const SetupPage = lazy(() => import('@/pages/auth/setup'))
const Setup2FAPage = lazy(() => import('@/pages/auth/setup-2fa'))
const SsoCallbackPage = lazy(() => import('@/pages/auth/sso-callback'))
const SetPasswordPage = lazy(() => import('@/pages/auth/set-password'))
const InvitePage = lazy(() => import('@/pages/invite'))

// Protected pages
const DashboardPage = lazy(() => import('@/pages/dashboard'))
const TransactionsPage = lazy(() => import('@/pages/transactions'))
const AccountsPage = lazy(() => import('@/pages/accounts'))
const CategoriesPage = lazy(() => import('@/pages/categories'))
const CurrenciesPage = lazy(() => import('@/pages/currencies'))
const BudgetsPage = lazy(() => import('@/pages/budgets'))
const TagsPage = lazy(() => import('@/pages/tags'))
const DebtsPage = lazy(() => import('@/pages/debts'))
const RecurringPage = lazy(() => import('@/pages/recurring'))
const AutomationPage = lazy(() => import('@/pages/automation'))
const AutomationLogsPage = lazy(() => import('@/pages/automation/[id]/logs'))

function AutomationCreateRedirect() {
    return <Navigate to="/automation?create=1" replace />
}

function AutomationEditRedirect() {
    const { id } = useParams<{ id: string }>()
    return <Navigate to={id ? `/automation?edit=${id}` : '/automation'} replace />
}

const ReportsPage = lazy(() => import('@/pages/reports'))
// Settings: the current space and the user's own.
const SpaceSettingsPage = lazy(() => import('@/pages/settings/space'))
const UserSettingsPage = lazy(() => import('@/pages/settings/user'))
const SecuritySettingsPage = lazy(() => import('@/pages/settings/security'))
const ApiSettingsPage = lazy(() => import('@/pages/settings/api'))
const ImportSettingsPage = lazy(() => import('@/pages/settings/import'))
const SpaceBackupsPage = lazy(() => import('@/pages/settings/backups'))
// Administration: server admins.
const AdminSystemPage = lazy(() => import('@/pages/admin/system'))
const AdminMonitoringPage = lazy(() => import('@/pages/admin/monitoring'))
const AdminSsoPage = lazy(() => import('@/pages/admin/sso'))
const AdminSecurityPage = lazy(() => import('@/pages/admin/security'))
const ProviderCreatePage = lazy(() => import('@/pages/admin/providers/create'))
const ProviderEditPage = lazy(() => import('@/pages/admin/providers/[id]/edit'))
const SystemBackupsPage = lazy(() => import('@/pages/admin/backups'))
const AdminUsersPage = lazy(() => import('@/pages/admin/users'))
const AdminSpacesPage = lazy(() => import('@/pages/admin/spaces'))

/** Server administration: others are sent home (the API refuses them anyway). */
function AdminOnly({ children }: { children: React.ReactNode }) {
    return useUser()?.role === 'admin' ? <>{children}</> : <Navigate to="/" replace />
}

const admin = (Component: React.LazyExoticComponent<() => React.JSX.Element>) => <AdminOnly>{withSuspense(Component)}</AdminOnly>

function ProviderEditRedirect() {
    const { id } = useParams<{ id: string }>()
    return <Navigate to={`/admin/providers/${id}/edit`} replace />
}
const NotFoundPage = lazy(() => import('@/pages/not-found'))

const withSuspense = (Component: React.LazyExoticComponent<() => React.JSX.Element>) => (
    <ErrorBoundary>
        <Suspense fallback={
            <div className="flex items-center justify-center min-h-[200px]">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
            </div>
        }>
            <Component />
        </Suspense>
    </ErrorBoundary>
)

export const router = createBrowserRouter([
    // Public routes
    {
        path: pages.login.path,
        element: withSuspense(LoginPage),
    },
    {
        path: pages.setup.path,
        element: withSuspense(SetupPage),
    },
    {
        path: pages.setup2fa.path,
        element: withSuspense(Setup2FAPage),
    },
    {
        path: pages.ssoCallback.path,
        element: withSuspense(SsoCallbackPage),
    },
    {
        path: pages.setPassword.path,
        element: withSuspense(SetPasswordPage),
    },
    {
        path: pages.invite.path,
        element: withSuspense(InvitePage),
    },

    // Protected routes
    {
        element: <AuthProvider />,
        children: [
            {
                // The layout sits at the root, where the dashboard is the index page.
                path: pages.dashboard.path,
                element: <AppLayout />,
                children: [
                    { index: true, element: withSuspense(DashboardPage) },
                    { path: childPath(pages.transactions), element: withSuspense(TransactionsPage) },
                    { path: childPath(pages.accounts), element: withSuspense(AccountsPage) },
                    { path: childPath(pages.categories), element: withSuspense(CategoriesPage) },
                    { path: childPath(pages.currencies), element: withSuspense(CurrenciesPage) },
                    { path: childPath(pages.budgets), element: withSuspense(BudgetsPage) },
                    { path: childPath(pages.tags), element: withSuspense(TagsPage) },
                    { path: childPath(pages.debts), element: withSuspense(DebtsPage) },
                    { path: childPath(pages.recurring), element: withSuspense(RecurringPage) },
                    { path: childPath(pages.automation), element: withSuspense(AutomationPage) },
                    { path: 'automation/create', element: <AutomationCreateRedirect /> },
                    { path: 'automation/:id/edit', element: <AutomationEditRedirect /> },
                    { path: childPath(pages.automationLogs), element: withSuspense(AutomationLogsPage) },
                    { path: childPath(pages.reports), element: withSuspense(ReportsPage) },
                    { path: childPath(pages.spaceSettings), element: withSuspense(SpaceSettingsPage) },
                    { path: childPath(pages.userSettings), element: withSuspense(UserSettingsPage) },
                    { path: childPath(pages.securitySettings), element: withSuspense(SecuritySettingsPage) },
                    { path: childPath(pages.apiSettings), element: withSuspense(ApiSettingsPage) },
                    { path: childPath(pages.importSettings), element: withSuspense(ImportSettingsPage) },
                    { path: childPath(pages.spaceBackups), element: withSuspense(SpaceBackupsPage) },
                    { path: childPath(pages.adminSystem), element: admin(AdminSystemPage) },
                    { path: childPath(pages.adminMonitoring), element: admin(AdminMonitoringPage) },
                    { path: childPath(pages.adminSso), element: admin(AdminSsoPage) },
                    { path: childPath(pages.adminSecurity), element: admin(AdminSecurityPage) },
                    { path: childPath(pages.providerCreate), element: admin(ProviderCreatePage) },
                    { path: childPath(pages.providerEdit), element: admin(ProviderEditPage) },
                    { path: childPath(pages.adminBackups), element: admin(SystemBackupsPage) },
                    { path: childPath(pages.adminUsers), element: admin(AdminUsersPage) },
                    { path: childPath(pages.adminSpaces), element: admin(AdminSpacesPage) },
                    // Old addresses.
                    { path: 'users', element: <Navigate to="/admin/users" replace /> },
                    { path: 'settings/system', element: <Navigate to="/admin/system" replace /> },
                    { path: 'settings/monitoring', element: <Navigate to="/admin/monitoring" replace /> },
                    { path: 'settings/providers', element: <Navigate to="/admin/sso" replace /> },
                    { path: 'settings/providers/create', element: <Navigate to="/admin/providers/create" replace /> },
                    { path: 'settings/providers/:id/edit', element: <ProviderEditRedirect /> },
                ],
            },
        ],
    },

    // 404
    { path: '*', element: withSuspense(NotFoundPage) },
])
