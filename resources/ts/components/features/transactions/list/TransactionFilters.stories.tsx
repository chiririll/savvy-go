import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { useTransactionListFilters } from '@/hooks/use-transaction-list-filters'
import { categories, tags } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'

import { TransactionFilterDetails } from './TransactionFilterDetails'
import { TransactionFiltersPanel } from './TransactionFiltersPanel'
import { TransactionSortSelect, TransactionTypeFilter } from './TransactionTypeAndSort'

// The filters live in the address (nuqs), so a story picks a state by its route.
const Panel = () => {
    const list = useTransactionListFilters()
    return <TransactionFiltersPanel list={list} categories={categories} tags={tags} />
}

const Details = () => {
    const list = useTransactionListFilters()
    return <TransactionFilterDetails list={list} categories={categories} tags={tags} />
}

const meta = {
    title: 'features/transactions/list/TransactionFilters',
    component: Panel,
    parameters: { a11y: overlayA11y },
} satisfies Meta<typeof Panel>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const PendingTab: Story = { parameters: { route: '/?status=pending' } }
export const ExpensesOnly: Story = { parameters: { route: '/?type=expense' } }
export const SortedByAmount: Story = { parameters: { route: '/?sortBy=amount&sortDir=desc' } }
export const WithActiveFilters: Story = {
    parameters: { route: '/?categoryIds=1&tagIds=2&startDate=2026-01-01' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('3')).toBeVisible()
        await expect(within(canvasElement).getByRole('button', { name: /clear/i })).toBeVisible()
    },
}
export const InRussian: Story = { globals: { locale: 'ru' } }

export const NoClearButtonWithoutFilters: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button', { name: /clear/i })).toBeNull()
    },
}

export const OpenedDetails: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /filters/i }))
        await expect(await within(canvasElement).findByRole('button', { name: /Groceries/ })).toBeVisible()
    },
}

export const CategoryToggled: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /filters/i }))
        const chip = await canvas.findByRole('button', { name: /Groceries/ })
        await userEvent.click(chip)
        await waitFor(() => expect(chip).toHaveAttribute('aria-pressed', 'true'))
        await userEvent.click(canvas.getByRole('button', { name: /clear/i }))
        await waitFor(() => expect(chip).toHaveAttribute('aria-pressed', 'false'))
    },
}

// Switching the type to transfers shows every category: a transfer has none of its own.
export const TypeNarrowsTheCategories: Story = {
    parameters: { route: '/?type=income' },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /filters/i }))
        await canvas.findByRole('button', { name: /Salary/ })
        await expect(canvas.queryByRole('button', { name: /Groceries/ })).toBeNull()
    },
}

export const PendingTabSelected: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('tab', { name: /pending|upcoming/i }))
        await waitFor(() => expect(canvas.getByRole('tab', { name: /pending|upcoming/i })).toHaveAttribute('aria-selected', 'true'))
    },
}

export const TypeFilterChosen: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /^income$/i }))
        await waitFor(() => expect(canvas.getByRole('button', { name: /^income$/i })).toHaveAttribute('aria-pressed', 'true'))
    },
}

export const SortChosen: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox'))
        await userEvent.click(await screen.findByRole('option', { name: /oldest/i }))
        await waitFor(() => expect(within(canvasElement).getByRole('combobox')).toHaveAccessibleName(/oldest/i))
    },
}

export const DetailsOnly: Story = { render: () => <Details /> }
export const DetailsWithDates: Story = {
    render: () => <Details />,
    parameters: { route: '/?startDate=2026-01-01&endDate=2026-02-01' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByLabelText(/from/i)).toHaveValue('2026-01-01')
        await expect(within(canvasElement).getByLabelText(/to$/i)).toHaveValue('2026-02-01')
    },
}

export const TypeButtons: Story = {
    render: () => <TransactionTypeFilter value="expense" onChange={() => {}} />,
}
export const TypeButtonsAll: Story = {
    render: () => <TransactionTypeFilter value={null} onChange={() => {}} />,
}
export const SortSelect: Story = {
    render: () => <TransactionSortSelect sortBy="amount" sortDir="asc" onChange={() => {}} />,
}
