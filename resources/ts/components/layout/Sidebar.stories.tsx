import type { Meta, StoryObj } from '@storybook/react-vite'
import { Boxes, Hash } from 'lucide-react'
import { expect, screen, userEvent, within } from 'storybook/test'

import { SidebarProvider } from '@/components/ui/sidebar'
import { accounts, currencies } from '@/test/fixtures'
import { accountsWithSummary, currencyApi, reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'
import { useUiStore } from '@/stores/ui'

import { NavGroup } from './NavGroup'
import { AppSidebar } from './Sidebar'
import { SidebarAbout } from './SidebarAbout'

const meta = {
    title: 'layout/Sidebar',
    parameters: { a11y: overlayA11y, layout: 'fullscreen' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        useUiStore.setState({ sidebarOpen: true, settingsOpen: true, adminOpen: true })
        msw.use(
            accountsWithSummary(accounts, { totalBalance: 13150.2, currency: 'USD' }),
            ...currencyApi(),
            reply('get', '/links', []),
            reply('get', '/admin/checks', []),
        )
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const inSidebar = (node: React.ReactNode) => <SidebarProvider>{node}</SidebarProvider>

export const SidebarMenu: Story = {
    render: () => inSidebar(<AppSidebar />),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('link', { name: /transactions/i })).toBeVisible()
    },
}

export const SidebarWithProblems: Story = {
    render: () => inSidebar(<AppSidebar />),
    beforeEach: ({ msw }) => {
        msw.use(reply('get', '/admin/checks', [
            { id: 'backup_missing', severity: 'error', value: null },
            { id: 'mail_off', severity: 'warning', value: null },
            { id: 'app_url_missing', severity: 'info', value: null },
        ]))
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByLabelText(/problems found/i)).toBeVisible()
        await expect(await within(canvasElement).findByText('2')).toBeVisible()
    },
}

export const SidebarWithoutAdministration: Story = {
    render: () => inSidebar(<AppSidebar />),
    parameters: { app: { serverRole: 'user' } },
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(within(canvasElement).queryByRole('link', { name: /checks/i })).toBeNull()
    },
}

export const SidebarInRussian: Story = {
    render: () => inSidebar(<AppSidebar />),
    globals: { locale: 'ru' },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByRole('link', { name: /транзакции/i })).toBeVisible()
    },
}

export const About: Story = {
    render: () => inSidebar(<SidebarAbout />),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(await screen.findByRole('menuitem', { name: /github/i })).toBeVisible()
    },
}

const items = [
    { to: '/spaces', icon: Boxes, labelKey: 'spaces' },
    { to: '/tags', icon: Hash, labelKey: 'tags' },
] as const

export const Group: Story = {
    render: () => inSidebar(
        <NavGroup icon={Boxes} label="Settings" items={[...items]} open onOpenChange={() => {}} isActive={(path) => path === '/tags'} />,
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('link', { name: /tags/i })).toHaveAttribute('data-active', 'true')
    },
}

export const GroupCollapsed: Story = {
    render: () => inSidebar(
        <NavGroup icon={Boxes} label="Settings" items={[...items]} open={false} onOpenChange={() => {}} isActive={() => false} />,
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('link')).toBeNull()
    },
}

export const GroupWithAlert: Story = {
    render: () => inSidebar(
        <NavGroup
            icon={Boxes} label="Settings" items={[...items]} open alert="error" counts={{ '/tags': 3 }}
            onOpenChange={() => {}} isActive={() => false}
        />,
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('3')).toBeVisible()
        await expect(within(canvasElement).getByLabelText(/problems found/i)).toBeVisible()
    },
}

