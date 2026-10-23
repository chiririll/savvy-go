import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { hang, reply } from '@/test/api-mocks'

import TagsPage from './index'

const tag = (id: number, name: string, transactionsCount: number) => ({ id, name, transactionsCount })

const meta = {
    title: 'pages/Tags',
    component: TagsPage,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/tags', [tag(1, 'holiday', 4), tag(2, '#VACATION', 0)])),
} satisfies Meta<typeof TagsPage>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('#holiday')).toBeVisible()
        // A built-in name shows in the language of the interface.
        await expect(canvas.getByText('#Vacation')).toBeVisible()
        await expect(canvas.getByRole('button', { name: /new tag/i })).toBeVisible()
    },
}
export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/tags')) }
export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/tags', [])),
    play: async ({ canvasElement }) => {
        await expect((await within(canvasElement).findAllByRole('button', { name: /new tag/i }))[0]).toBeVisible()
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        await waitFor(async () => expect(await within(canvasElement).findByText('#holiday')).toBeVisible())
        // The role comes from the space list, which can land after the rows.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new tag/i })).toBeNull())
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new tag/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
