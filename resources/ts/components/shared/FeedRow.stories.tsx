import type { Meta, StoryObj } from '@storybook/react-vite'
import { ShoppingCart } from 'lucide-react'
import { expect, fn, userEvent, within } from 'storybook/test'

import { currencies } from '@/test/fixtures'
import { formatCurrency } from '@/lib/currency'
import { useCurrenciesStore } from '@/stores/currencies'

import { FeedEmpty, FeedList, FeedProgress, FeedRow, FeedRowSkeleton, FeedStatusBadge } from './FeedRow'
import { progressAmounts } from './progress-amounts'
import { RowActions } from './RowActions'

const [usd, eur, jpy, kwd] = currencies

const meta = {
    title: 'shared/FeedRow',
    component: FeedRow,
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: { icon: <ShoppingCart className="size-4" />, title: 'Groceries', iconClassName: 'bg-muted' },
} satisfies Meta<typeof FeedRow>

export default meta
type Story = StoryObj<typeof meta>

export const Minimal: Story = {}

export const Complete: Story = {
    args: {
        subtitle: 'Cash · Today',
        meta: <FeedStatusBadge variant="outline">Recurring</FeedStatusBadge>,
        badge: <FeedStatusBadge>New</FeedStatusBadge>,
        amount: formatCurrency(-42.5, usd),
        amountClassName: 'text-red-600',
        extraAmount: formatCurrency(-39.1, eur),
    },
}

export const Muted: Story = { args: { muted: true, amount: formatCurrency(10, usd), subtitle: 'Inactive' } }

export const LongTitle: Story = {
    args: {
        title: 'An extremely long transaction title that has to be cut with an ellipsis before the amount',
        amount: formatCurrency(1234567.89, usd),
        subtitle: 'And a subtitle that is just as long as the title, to see both of them cut',
    },
}

// Currencies with 0 and 3 decimals show their own precision.
export const ZeroDecimals: Story = { args: { amount: formatCurrency(1500, jpy) } }
export const ThreeDecimals: Story = { args: { amount: formatCurrency(12.345, kwd) } }

export const Opened: Story = {
    args: { onOpen: fn() },
    play: async ({ canvasElement, args }) => {
        const row = within(canvasElement).getByRole('button')
        await userEvent.click(row)
        row.focus()
        await userEvent.keyboard('{Enter}')
        await userEvent.keyboard(' ')
        await expect(args.onOpen).toHaveBeenCalledTimes(3)
    },
}

export const WithActions: Story = {
    args: {
        hasActions: true,
        actions: () => <RowActions onEdit={fn()} onDelete={fn()} deleteTitle="Delete it?" />,
    },
}

export const Progress: Story = {
    args: {
        below: <FeedProgress value={60} />,
        ...progressAmounts({ remaining: 120, total: 300, currency: usd }),
    },
}

export const ProgressExceeded: Story = {
    args: {
        below: <FeedProgress value={140} exceeded />,
        ...progressAmounts({ remaining: -40, total: 300, currency: usd }),
    },
}

export const ProgressDone: Story = {
    args: {
        below: <FeedProgress value={100} />,
        ...progressAmounts({ remaining: 0, total: 300, currency: usd, done: 'Paid off' }),
    },
}

export const ProgressWithoutTotal: Story = { args: progressAmounts({ remaining: 55.5, total: 0, currency: usd }) }

const items = [
    { id: 1, name: 'Coffee', day: 'Today' },
    { id: 2, name: 'Bus', day: 'Today' },
    { id: 3, name: 'Rent', day: 'Yesterday' },
]

export const List: Story = {
    render: () => (
        <FeedList
            items={items}
            getKey={(item) => item.id}
            emptyTitle="Nothing"
            emptyDescription="Nothing here"
            groupBy={(item) => ({ key: item.day, title: item.day })}
        >
            {(item) => <FeedRow icon={<ShoppingCart className="size-4" />} title={item.name} />}
        </FeedList>
    ),
}

export const ListLoading: Story = {
    render: () => <FeedList items={[]} isLoading getKey={() => 0} emptyTitle="" emptyDescription="">{() => null}</FeedList>,
}

export const ListEmpty: Story = {
    render: () => (
        <FeedList items={[]} getKey={() => 0} emptyTitle="No transactions" emptyDescription="Add one." onCreate={fn()} createLabel="Add">
            {() => null}
        </FeedList>
    ),
}

export const ListEmptyReadOnly: Story = {
    render: () => (
        <FeedEmpty title="No transactions" description="Ask an editor to add one." onCreate={fn()} createLabel="Add" isReadOnly />
    ),
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Skeleton: Story = { render: () => <FeedRowSkeleton showHeading /> }
