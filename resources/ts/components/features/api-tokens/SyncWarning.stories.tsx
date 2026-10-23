import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { reply } from '@/test/api-mocks'

import { SyncWarning } from './SyncWarning'

const spaces = [{ id: 1, name: 'Home' }, { id: 2, name: 'Trip' }]
const group = { id: 1, categories: true, tags: true, members: spaces.map((s) => ({ ...s, role: 'admin' })), hidden: 0 }

const meta = {
    tags: ['desktop-only'],
    title: 'features/api-tokens/SyncWarning',
    component: SyncWarning,
    args: { spaces, choice: { all: true, ids: [] } },
    beforeEach: ({ msw }) => msw.use(reply('get', '/sync-group', group)),
} satisfies Meta<typeof SyncWarning>

export default meta
type Story = StoryObj<typeof meta>

export const Mirrored: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findAllByText(/trip/i)).not.toHaveLength(0)
    },
}
export const OnlyCategories: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/sync-group', { ...group, tags: false })),
}
export const OnlyTags: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/sync-group', { ...group, categories: false })),
}

// The token reaches Home but not Trip, where its changes would be refused.
export const OtherSpaceUnreachable: Story = {
    args: { choice: { all: false, ids: [1] } },
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByText(/trip/i, { selector: '.text-destructive' })
    },
}

export const HiddenSpaces: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/sync-group', { ...group, hidden: 2 })),
}

export const NoGroup: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/sync-group', null)),
    play: async ({ canvasElement }) => {
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(canvasElement.textContent).toBe('')
    },
}

export const NothingChosen: Story = {
    args: { choice: { all: false, ids: [] } },
    play: async ({ canvasElement }) => {
        await expect(canvasElement.textContent).toBe('')
    },
}
