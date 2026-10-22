import type { Meta, StoryObj } from '@storybook/react-vite'
import { Route, Routes } from 'react-router-dom'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { SidebarProvider } from '@/components/ui/sidebar'
import { accounts, currencies } from '@/test/fixtures'
import { accountsWithSummary, currencyApi, reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { space } from '@/test/app-state'
import { useCurrenciesStore } from '@/stores/currencies'
import { useUiStore } from '@/stores/ui'
import { CreateTransactionProvider } from '@/components/features/transactions/form/CreateTransactionProvider'

import { AppLayout } from './AppLayout'
import { Header } from './Header'
import { StatusBar } from './StatusBar'
import { CreateTransactionMenu } from './header/CreateTransactionMenu'
import { HeaderBalance } from './header/HeaderBalance'
import { UserMenu } from './header/UserMenu'

const summary = { totalBalance: 13150.2, currency: 'USD' }

const meta = {
    title: 'layout/Layout',
    parameters: { a11y: overlayA11y, layout: 'fullscreen' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        useUiStore.setState({ sidebarOpen: true, settingsOpen: true, adminOpen: true })
        msw.use(
            accountsWithSummary(accounts, summary),
            ...currencyApi(),
            reply('get', '/links', []),
            reply('get', '/admin/checks', []),
        )
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const inSidebar = (node: React.ReactNode) => <SidebarProvider>{node}</SidebarProvider>

// ---- the whole frame

const Frame = () => (
    <Routes>
        <Route element={<AppLayout />}>
            <Route index element={<p>Dashboard page</p>} />
            <Route path="*" element={<p>Another page</p>} />
        </Route>
    </Routes>
)

export const Application: Story = {
    render: () => <Frame />,
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Dashboard page')).toBeVisible()
        await expect(await within(canvasElement).findByText(/13,150\.20/)).toBeVisible()
    },
}

export const ApplicationAsViewer: Story = {
    render: () => <Frame />,
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('status')).toBeVisible())
        await expect(within(canvasElement).queryByRole('button', { name: /new transaction/i })).toBeNull()
    },
}

export const ApplicationInRussian: Story = {
    render: () => <Frame />,
    globals: { locale: 'ru' },
    play: async ({ canvasElement }) => {
        // Russian groups digits with spaces and uses a comma for the decimals.
        await expect(await within(canvasElement).findByText(/13\D150,20/)).toBeVisible()
    },
}
export const ApplicationInDarkTheme: Story = { ...Application, globals: { theme: 'dark' } }

// ---- the status bar above the header

export const NoStatus: Story = {
    render: () => <StatusBar />,
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('status')).toBeNull()
    },
}

export const ReadOnlyStatus: Story = {
    render: () => <StatusBar />,
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('status')).toBeVisible()
    },
}

export const ReadOnlyStatusDismissed: Story = {
    render: () => <StatusBar />,
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /close/i }))
        await waitFor(() => expect(within(canvasElement).queryByRole('status')).toBeNull())
        await expect(localStorage.getItem('savvy-hidden-statuses')).toContain('readOnly')
    },
}

// A build that is not production says so, and cannot be dismissed.
export const DevelopmentStatus: Story = {
    render: () => <StatusBar />,
    beforeEach: () => {
        const meta = document.createElement('meta')
        meta.name = 'app-env'
        meta.content = 'local'
        document.head.appendChild(meta)
        return () => meta.remove()
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('status')).toBeVisible()
        await expect(within(canvasElement).queryByRole('button', { name: /close/i })).toBeNull()
    },
}

export const BothStatuses: Story = {
    ...DevelopmentStatus,
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).getByRole('status').textContent?.length).toBeGreaterThan(30))
    },
}

// ---- the header

export const HeaderBar: Story = {
    render: () => inSidebar(<CreateTransactionProvider><Header /></CreateTransactionProvider>),
}

export const CreateMenu: Story = {
    render: () => <CreateTransactionProvider><CreateTransactionMenu /></CreateTransactionProvider>,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(await screen.findAllByRole('menuitem')).toHaveLength(3)
    },
}

export const CreateMenuWithSpaceTransfer: Story = {
    render: () => <CreateTransactionProvider><CreateTransactionMenu /></CreateTransactionProvider>,
    beforeEach: ({ msw }) => {
        msw.use(
            reply('get', '/spaces', [space('admin'), { ...space('editor'), id: 2, name: 'Trip' }]),
            reply('get', '/links', [{ id: 2, name: 'Trip' }]),
        )
    },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await waitFor(async () => expect(await screen.findAllByRole('menuitem')).toHaveLength(4))
    },
}

export const CreateMenuHiddenForViewers: Story = {
    render: () => <CreateTransactionProvider><CreateTransactionMenu /></CreateTransactionProvider>,
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await waitFor(() => expect(within(canvasElement).queryByRole('button')).toBeNull())
    },
}

export const Balance: Story = {
    render: () => <HeaderBalance />,
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/13,150\.20/)).toBeVisible()
    },
}

export const BalanceAccounts: Story = {
    render: () => <HeaderBalance />,
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /balances/i }))
        await expect(await screen.findByText('Yen wallet')).toBeVisible()
        await expect(screen.getByText(/15,000/)).toBeVisible()
    },
}

export const BalanceWithoutAccounts: Story = {
    render: () => <HeaderBalance />,
    beforeEach: ({ msw }) => msw.use(accountsWithSummary([], summary)),
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /balances/i }))
        await expect(await screen.findByText(/no active accounts/i)).toBeVisible()
    },
}

export const BalanceUnavailable: Story = {
    render: () => <HeaderBalance />,
    beforeEach: ({ msw }) => msw.use(reply('get', '/accounts', [])),
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 400))
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const User: Story = {
    render: () => <UserMenu />,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(await screen.findByText('Kirill')).toBeVisible()
        await expect(screen.getByRole('menuitem', { name: /log ?out|sign out/i })).toBeVisible()
    },
}

export const UserSignedOut: Story = {
    render: () => <UserMenu />,
    parameters: { app: { signedOut: true } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}
