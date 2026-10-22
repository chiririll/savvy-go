import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { SidebarProvider } from '@/components/ui/sidebar'
import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'
import { space } from '@/test/app-state'
import { useSpaceStore } from '@/stores/space'

import { SpaceSwitcher } from './SpaceSwitcher'

const mine = [
    space(),
    { ...space('editor'), id: 2, name: 'Work', emoji: '💼', color: '#16a34a' },
    { ...space('viewer'), id: 3, name: 'Parents', emoji: null, color: null },
]

const meta = {
    title: 'features/spaces/SpaceSwitcher',
    component: SpaceSwitcher,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/spaces', mine)),
    decorators: [
        (Story) => (
            <SidebarProvider>
                <div className="w-64">
                    <Story />
                </div>
            </SidebarProvider>
        ),
    ],
} satisfies Meta<typeof SpaceSwitcher>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    await userEvent.click(await within(canvasElement).findByRole('button', { name: /home/i }))
    return within(await screen.findByRole('menu'))
}

export const Current: Story = {
    play: async ({ canvasElement }) => {
        const trigger = await within(canvasElement).findByRole('button', { name: /home/i })
        await expect(trigger).toHaveTextContent('Admin')
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
export const InDarkTheme: Story = { globals: { theme: 'dark' } }

export const Opened: Story = {
    play: async ({ canvasElement }) => {
        const menu = await open(canvasElement)
        await expect(menu.getAllByRole('menuitem')).toHaveLength(5)
        await expect(menu.getByRole('menuitem', { name: /parents/i })).toHaveTextContent('Viewer')
        await expect(menu.getByRole('menuitem', { name: /create a space/i })).toBeVisible()
    },
}

export const Switched: Story = {
    play: async ({ canvasElement }) => {
        const menu = await open(canvasElement)
        await userEvent.click(menu.getByRole('menuitem', { name: /work/i }))
        await waitFor(() => expect(useSpaceStore.getState().currentId).toBe(2))
    },
}

// A guest of the server cannot make spaces of their own.
export const AsGuest: Story = {
    parameters: { app: { serverRole: 'guest' } },
    play: async ({ canvasElement }) => {
        const menu = await open(canvasElement)
        await expect(menu.queryByRole('menuitem', { name: /create a space/i })).toBeNull()
        await expect(menu.getByRole('menuitem', { name: /space settings/i })).toBeVisible()
    },
}

export const CreateOpensTheModal: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/presets', { categories: [], tags: [] }), reply('get', '/currencies/catalog', [])),
    play: async ({ canvasElement }) => {
        const menu = await open(canvasElement)
        await userEvent.click(menu.getByRole('menuitem', { name: /create a space/i }))
        await expect(await screen.findByRole('dialog', { name: /new space/i })).toBeVisible()
    },
}

export const LongName: Story = {
    beforeEach: ({ msw }) =>
        msw.use(reply('get', '/spaces', [{ ...space(), name: 'Home with a very long name that keeps on going' }])),
}
