import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, within } from 'storybook/test'

import { result } from '../fixtures'
import { ResultStep } from './ResultStep'

const meta = {
    title: 'features/import/ResultStep',
    component: ResultStep,
    parameters: { layout: 'padded' },
    args: { result: result() },
} satisfies Meta<typeof ResultStep>

export default meta
type Story = StoryObj<typeof meta>

export const Imported: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/successfully imported 117 transactions/i)).toBeVisible()
        await expect(canvas.getByText(/row 6: bad date/i)).toBeVisible()
        await expect(canvas.getByRole('button', { name: /view transactions/i })).toBeVisible()
    },
}

export const NothingImported: Story = {
    args: {
        result: result({ created: 0, skippedDuplicates: 5, errors: [], createdCurrencies: [], createdTags: [], createdCategories: [] }),
    },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/no transactions imported/i)).toBeVisible()
        await expect(canvas.queryByText(/new entities created/i)).toBeNull()
    },
}

export const OneImported: Story = {
    args: { result: result({ created: 1, errors: [] }) },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText(/imported 1 transaction$/i)).toBeVisible()
    },
}

// Only new currencies came out of the file: they are listed, not an empty box.
export const CurrenciesCreated: Story = {
    args: { result: result({ createdCurrencies: ['JPY', 'KWD'], createdTags: [], createdCategories: [] }) },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/new entities created/i)).toBeVisible()
        await expect(canvas.getByText('JPY')).toBeVisible()
        await expect(canvas.getByText('KWD')).toBeVisible()
    },
}

export const ManyErrors: Story = {
    args: {
        result: result({
            errors: Array.from({ length: 40 }, (_, i) => ({ row: i + 2, message: 'Amount is not a number' })),
        }),
    },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('Errors (40)')).toBeVisible()
    },
}
