import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { unmountedTabsOverlayA11y } from '@/test/a11y'
import { category } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'

import CategoriesPage from './index'

const list = [
    category({ id: 1, name: 'Groceries', type: 'expense', isDefault: true }),
    category({ id: 2, name: 'Rent', type: 'expense', isDefault: false }),
    category({ id: 3, name: 'Salary', type: 'income', isDefault: true }),
]

const meta = {
    title: 'pages/Categories',
    component: CategoriesPage,
    parameters: { a11y: unmountedTabsOverlayA11y },
    beforeEach: ({ msw }) => msw.use(reply('get', '/categories', list), reply('get', '/spaces/1/links', [])),
} satisfies Meta<typeof CategoriesPage>

export default meta
type Story = StoryObj<typeof meta>

export const Expenses: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Groceries')).toBeVisible()
        await expect(canvas.getByText('Rent')).toBeVisible()
        await expect(canvas.queryByText('Salary')).toBeNull()
        await expect(canvas.getByRole('button', { name: /new category/i })).toBeVisible()
    },
}
export const InRussian: Story = { tags: ['mobile-only'], globals: { locale: 'ru' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/categories')) }

export const IncomeTab: Story = {
    parameters: { route: '/categories?type=income' },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Salary')).toBeVisible()
        await expect(canvas.queryByText('Rent')).toBeNull()
    },
}

export const SwitchesTheTab: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('tab', { name: /income/i }))
        await expect(await canvas.findByText('Salary')).toBeVisible()
        await waitFor(() => expect(canvas.queryByText('Rent')).toBeNull())
    },
}

export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async ({ canvasElement }) => {
        // The role comes from the space list, which can land after the rows, and the rows are drawn again then.
        await waitFor(() => expect(within(canvasElement).queryByRole('button', { name: /new category/i })).toBeNull())
        await expect(await within(canvasElement).findByText('Groceries')).toBeVisible()
    },
}

export const OpensTheCreateDialog: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('button', { name: /new category/i }))
        await expect(await screen.findByRole('dialog')).toBeVisible()
    },
}
