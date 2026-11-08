import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { accounts, categories, currencies, recurring, recurringTemplate } from '@/test/fixtures'
import { reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { RecurringRow } from './RecurringRow'

const meta = {
    title: 'features/recurring/RecurringRow',
    component: RecurringRow,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts), reply('get', '/categories', categories))
    },
    args: { recurring: recurring(), onEdit: fn(), onDelete: fn() },
} satisfies Meta<typeof RecurringRow>

export default meta
type Story = StoryObj<typeof meta>

const make = (template: Parameters<typeof recurringTemplate>[0], overrides: Parameters<typeof recurring>[0] = {}) =>
    recurring({ template: recurringTemplate(template), ...overrides })

export const Expense: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/1,100\.00/)).toBeVisible()
    },
}
export const Income: Story = {
    args: {
        recurring: make({
            type: 'income',
            description: 'Salary',
            payments: [{ account_id: 1, amount: 3200 }],
            lines: [{ category_id: 2, currency_id: 1, amount: 3200 }],
        }),
    },
}
export const Transfer: Story = {
    args: {
        recurring: make({
            type: 'transfer',
            description: 'To savings',
            payments: [
                { account_id: 1, amount: -1000 },
                { account_id: 2, amount: 800 },
            ],
            lines: [],
        }),
    },
}

// The amount received was left open: it is what the rates of each day make of it, so it is shown as a guess.
export const TransferWithTheAmountReceivedOpen: Story = {
    args: {
        recurring: make({
            type: 'transfer',
            description: 'To savings',
            payments: [
                { account_id: 1, amount: -1000 },
                { account_id: 2, amount: null, estimated: true },
            ],
            lines: [],
        }),
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/≈/)).toBeVisible()
    },
}

export const Estimated: Story = {
    args: {
        recurring: make({
            description: 'Electricity',
            payments: [{ account_id: 1, amount: -1100, estimated: true }],
        }),
    },
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/≈/)).toBeVisible()
    },
}
export const Paused: Story = { args: { recurring: recurring({ isActive: false }) } }

// A rule that stopped says why in place of its details, and has no next run.
export const Stopped: Story = {
    args: { recurring: recurring({ isActive: false, error: 'unknown account', nextRunDate: null }) },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Stopped')).toBeVisible()
        await expect(canvas.getByText(/unknown account/i)).toBeVisible()
    },
}

export const SeveralCategories: Story = {
    args: {
        recurring: make({
            description: 'Market',
            payments: [{ account_id: 1, amount: -100 }],
            lines: [
                { category_id: 1, currency_id: 1, amount: 60 },
                { category_id: 3, currency_id: 1, amount: 40 },
            ],
        }),
    },
}

// A run date that has passed is called out.
export const OverdueRun: Story = { args: { recurring: recurring({ nextRunDate: '2020-01-01' }) } }
export const PausedOverdueRunIsNotCalledOut: Story = {
    args: { recurring: recurring({ nextRunDate: '2020-01-01', isActive: false }) },
}
export const NoDescription: Story = { args: { recurring: make({ description: null }) } }
export const YenAmount: Story = {
    args: {
        recurring: make({
            description: 'Rent in Tokyo',
            payments: [{ account_id: 3, amount: -80000 }],
            lines: [{ category_id: 1, currency_id: 3, amount: 80000 }],
        }),
    },
}
export const ThreeDecimals: Story = {
    args: {
        recurring: make({
            description: 'Fee',
            payments: [{ account_id: 4, amount: -12.345 }],
            lines: [{ category_id: 1, currency_id: 4, amount: 12.345 }],
        }),
    },
}
export const LongDescription: Story = {
    args: { recurring: make({ description: 'A description far too long for the single line of a row in a list' }) },
}
export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Rent'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.recurring)
    },
}

export const Deleted: Story = {
    tags: ['desktop-only'],
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}
