import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { currencies, debt } from '@/test/fixtures'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

import { DebtRow } from './DebtRow'

const [, eur, jpy] = currencies

const meta = {
    title: 'features/debts/DebtRow',
    component: DebtRow,
    parameters: { a11y: overlayA11y },
    beforeEach: () => useCurrenciesStore.getState().setAll(currencies),
    args: {
        debt: debt(),
        onEdit: fn(),
        onDelete: fn(),
        onPayment: fn(),
        onCollect: fn(),
        onReopen: fn(),
    },
} satisfies Meta<typeof DebtRow>

export default meta
type Story = StoryObj<typeof meta>

const openMenu = (canvasElement: HTMLElement) =>
    userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))

export const IOwe: Story = {}
export const OwedToMe: Story = { args: { debt: debt({ debtType: 'owed_to_me', name: 'Loan to Sam', counterparty: 'Sam' }) } }
export const PartlyPaid: Story = { args: { debt: debt({ currentBalance: 125, paymentProgress: 75 }) } }
export const PaidOff: Story = { args: { debt: debt({ currentBalance: 0, paymentProgress: 100, isPaidOff: true }) } }

// A debt past its due date is called out, unless it has been paid.
export const Overdue: Story = { args: { debt: debt({ dueDate: '2020-01-31' }) } }
export const OverduePaidOff: Story = {
    args: { debt: debt({ dueDate: '2020-01-31', currentBalance: 0, isPaidOff: true, paymentProgress: 100 }) },
}
export const NoDueDate: Story = { args: { debt: debt({ dueDate: null }) } }
export const NoCounterparty: Story = { args: { debt: debt({ counterparty: null }) } }
export const NoDetails: Story = { args: { debt: debt({ counterparty: null, dueDate: null }) } }
export const NoTarget: Story = { args: { debt: debt({ targetAmount: 0, paymentProgress: 0 }) } }

// Owing more than the original amount shows how much it went over.
export const OverTheTarget: Story = { args: { debt: debt({ currentBalance: 540, paymentProgress: 0 }) } }
export const InEuro: Story = { args: { debt: debt({ currency: eur, currentBalance: 99.9, targetAmount: 120 }) } }
export const InYen: Story = { args: { debt: debt({ currency: jpy, currentBalance: 45000, targetAmount: 60000 }) } }
export const LongTitle: Story = { args: { debt: debt({ name: 'A debt with a name much too long for the single line it sits on' }) } }

export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const MakePayment: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /payment/i }))
        await expect(args.onPayment).toHaveBeenCalledWith(args.debt)
        await expect(args.onCollect).not.toHaveBeenCalled()
    },
}

export const CollectPayment: Story = {
    args: { debt: debt({ debtType: 'owed_to_me' }) },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /collect/i }))
        await expect(args.onCollect).toHaveBeenCalledWith(args.debt)
        await expect(args.onPayment).not.toHaveBeenCalled()
    },
}

export const PaidOffCanBeReopened: Story = {
    args: { debt: debt({ currentBalance: 0, isPaidOff: true }) },
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await expect(screen.queryByRole('menuitem', { name: /payment|collect/i })).toBeNull()
        await userEvent.click(await screen.findByRole('menuitem', { name: /reopen/i }))
        await expect(args.onReopen).toHaveBeenCalledWith(args.debt.id)
    },
}

export const OpenDebtCannotBeReopened: Story = {
    play: async ({ canvasElement }) => {
        await openMenu(canvasElement)
        await screen.findByRole('menuitem', { name: /edit/i })
        await expect(screen.queryByRole('menuitem', { name: /reopen/i })).toBeNull()
    },
}

export const Edited: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Loan from Alex'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.debt)
    },
}

export const Deleted: Story = {
    play: async ({ canvasElement, args }) => {
        await openMenu(canvasElement)
        await userEvent.click(await screen.findByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(args.debt.id)
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
