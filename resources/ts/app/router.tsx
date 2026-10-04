import { lazy, Suspense } from 'react'
import { createBrowserRouter, Navigate, useParams } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { AuthProvider } from '@/components/providers/AuthProvider'
import { useUser } from '@/stores/auth'
import { ErrorBoundary } from '@/components/shared/ErrorBoundary'

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
        path: '/login',
        element: withSuspense(LoginPage),
    },
    {
        path: '/setup',
        element: withSuspense(SetupPage),
    },
    {
        path: '/setup-2fa',
        element: withSuspense(Setup2FAPage),
    },
    {
        path: '/auth/sso/callback',
        element: withSuspense(SsoCallbackPage),
    },
    {
        path: '/set-password/:token',
        element: withSuspense(SetPasswordPage),
    },
    {
        path: '/invite/:token',
        element: withSuspense(InvitePage),
    },

    // Protected routes
    {
        element: <AuthProvider />,
        children: [
            {
                path: '/',
                element: <AppLayout />,
                children: [
                    { index: true, element: withSuspense(DashboardPage) },
                    { path: 'transactions', element: withSuspense(TransactionsPage) },
                    { path: 'accounts', element: withSuspense(AccountsPage) },
                    { path: 'categories', element: withSuspense(CategoriesPage) },
                    { path: 'currencies', element: withSuspense(CurrenciesPage) },
                    { path: 'budgets', element: withSuspense(BudgetsPage) },
                    { path: 'tags', element: withSuspense(TagsPage) },
                    { path: 'debts', element: withSuspense(DebtsPage) },
                    { path: 'recurring', element: withSuspense(RecurringPage) },
                    { path: 'automation', element: withSuspense(AutomationPage) },
                    { path: 'automation/create', element: <AutomationCreateRedirect /> },
                    { path: 'automation/:id/edit', element: <AutomationEditRedirect /> },
                    { path: 'automation/:id/logs', element: withSuspense(AutomationLogsPage) },
                    { path: 'reports', element: withSuspense(ReportsPage) },
                    { path: 'settings/space', element: withSuspense(SpaceSettingsPage) },
                    { path: 'settings/user', element: withSuspense(UserSettingsPage) },
                    { path: 'settings/security', element: withSuspense(SecuritySettingsPage) },
                    { path: 'settings/api', element: withSuspense(ApiSettingsPage) },
                    { path: 'settings/import', element: withSuspense(ImportSettingsPage) },
                    { path: 'settings/backups', element: withSuspense(SpaceBackupsPage) },
                    { path: 'admin/system', element: admin(AdminSystemPage) },
                    { path: 'admin/monitoring', element: admin(AdminMonitoringPage) },
                    { path: 'admin/sso', element: admin(AdminSsoPage) },
                    { path: 'admin/security', element: admin(AdminSecurityPage) },
                    { path: 'admin/providers/create', element: admin(ProviderCreatePage) },
                    { path: 'admin/providers/:id/edit', element: admin(ProviderEditPage) },
                    { path: 'admin/backups', element: admin(SystemBackupsPage) },
                    { path: 'admin/users', element: admin(AdminUsersPage) },
                    { path: 'admin/spaces', element: admin(AdminSpacesPage) },
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
