import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'
import { accounts, categories, currencies, line, payment, tags, transaction } from '@/test/fixtures'
import { reply, spy } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

// The provider loads the dialog when first asked; having it in already keeps a busy test run from timing out.
import './TransactionFormDialog'
import { CreateTransactionProvider, useTransactionFormDialog } from './CreateTransactionProvider'

const Buttons = ({ editable = true }: { editable?: boolean }) => {
    const { openCreate, openEdit } = useTransactionFormDialog()
    const existing = transaction({
        status: 'confirmed',
        actions: { edit: editable, editDetails: false, duplicate: true, delete: true, confirm: false, skip: false },
    })

    // One payment in two lines, several payments and a transfer open in the form.
    const lines = transaction({
        status: 'confirmed',
        lines: [line({ amount: 30, note: '- milk' }), line({ seq: 1, categoryId: 3, category: categories[2], amount: 12.5 })],
    })


    const split = transaction({
        status: 'confirmed',
        payments: [payment({ amount: -30 }), payment({ account: accounts[1], accountId: 2, amount: -10 })],
        lines: [line({ amount: 30 }), line({ seq: 1, amount: 10, currencyId: 2 })],
    })

    const transfer = transaction({
        type: 'transfer',
        status: 'confirmed',
        payments: [payment({ amount: -30 }), payment({ account: accounts[1], accountId: 2, amount: 24 })],
        lines: [],
    })

    // A payment on a debt: only its details change.
    const debtPayment = transaction({
        type: 'debt_payment',
        status: 'confirmed',
        description: 'Back to Alex',
        payments: [payment({ amount: -40 })],
        lines: [],
        restrictions: [{ reason: 'debt', message: 'debt' }],
        actions: { edit: false, editDetails: true, duplicate: false, delete: true, confirm: false, skip: false },
    })

    // A transfer between spaces that is in review: nothing can change.
    const inReview = transaction({
        type: 'transfer_in',
        status: 'confirmed',
        payments: [payment({ amount: 30 })],
        lines: [],
        restrictions: [{ reason: 'transfer_review', message: 'in review' }],
        actions: { edit: false, editDetails: false, duplicate: false, delete: false, confirm: false, skip: false },
    })

    // The other side of a transfer to a linked space: the form cannot show it.
    const spaceSide = transaction({
        type: 'transfer_out',
        status: 'confirmed',
        payments: [payment({ amount: -30 })],
        lines: [],
    })

    return (
        <div className="flex gap-2">
            <Button onClick={() => openCreate()}>New</Button>
            <Button onClick={() => openCreate({ type: 'income', account_id: 1, category_id: 2, amount: 10 })}>New income</Button>
            <Button variant="outline" onClick={() => openEdit(existing)}>Edit</Button>
            <Button variant="outline" onClick={() => openEdit(lines)}>Edit lines</Button>
            <Button variant="outline" onClick={() => openEdit(split)}>Edit split</Button>
            <Button variant="outline" onClick={() => openEdit(transfer)}>Edit transfer</Button>
            <Button variant="outline" onClick={() => openEdit(spaceSide)}>Edit space transfer</Button>
            <Button variant="outline" onClick={() => openEdit(debtPayment)}>Edit debt payment</Button>
            <Button variant="outline" onClick={() => openEdit(inReview)}>Edit transfer in review</Button>
        </div>
    )
}

const meta = {
    title: 'features/transactions/form/CreateTransactionProvider',
    component: Buttons,
    parameters: { a11y: overlayA11y },
    render: (args) => (
        <CreateTransactionProvider>
            <Buttons {...args} />
        </CreateTransactionProvider>
    ),
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(
            reply('get', '/accounts', accounts),
            reply('get', '/categories', categories),
            reply('get', '/currencies', currencies),
            reply('get', '/tags', tags),
            reply('get', '/links', []),
        )
    },
} satisfies Meta<typeof Buttons>

export default meta
type Story = StoryObj<typeof meta>

const open = (canvasElement: HTMLElement, name: string) =>
    userEvent.click(within(canvasElement).getByRole('button', { name }))

export const CreateOpened: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'New')
        // The form is loaded when first needed.
        await expect(await screen.findByRole('dialog', {}, { timeout: 5000 })).toBeVisible()
    },
}

export const CreateWithDefaults: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'New income')
        await expect(await screen.findByRole('spinbutton', { name: /amount/i }, { timeout: 5000 })).toHaveValue(10)
    },
}

export const EditOpened: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit')
        await expect(await screen.findByDisplayValue('Weekly shop', {}, { timeout: 5000 })).toBeVisible()
    },
}

// One payment in several lines opens in the form, with the amount and note of each line.
export const EditOfLinesOpensTheForm: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit lines')
        const group = await screen.findByRole('group', { name: /lines|строки/i }, { timeout: 5000 })
        await expect(within(group).getAllByRole('spinbutton')).toHaveLength(2)
        await expect(within(group).getAllByRole('spinbutton')[1]).toHaveValue(12.5)
    },
}

// Several payments open in the form, one row each, with the line of each currency.
export const EditOfSeveralPaymentsOpensTheForm: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit split')
        const dialog = await screen.findByRole('dialog', {}, { timeout: 5000 })
        const amounts = await within(dialog).findAllByRole('spinbutton')
        await expect(amounts.slice(0, 2).map((a) => (a as HTMLInputElement).value)).toEqual(['30', '10'])
    },
}

// A transfer between two accounts opens in the form, with the amount sent and the amount received.
export const EditOfATransferOpensTheForm: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit transfer')
        const dialog = await screen.findByRole('dialog', {}, { timeout: 5000 })
        const amounts = await within(dialog).findAllByRole('spinbutton')
        await expect(amounts.slice(0, 2).map((a) => (a as HTMLInputElement).value)).toEqual(['30', '24'])
    },
}

// The side of a transfer to another space cannot be shown whole: it opens with only its details to change.
export const EditOfASpaceTransferOpensRestricted: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit space transfer')
        const dialog = await screen.findByRole('dialog', {}, { timeout: 5000 })
        await expect(await within(dialog).findByRole('note')).toBeVisible()
        await expect(within(dialog).queryByRole('spinbutton')).toBeNull()
    },
}

// A payment on a debt opens with its title and tags to change, and the reason.
export const EditOfADebtPaymentOpensRestricted: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit debt payment')
        const dialog = await screen.findByRole('dialog', {}, { timeout: 5000 })
        await expect(await within(dialog).findByDisplayValue('Back to Alex')).toBeEnabled()
        await expect(within(dialog).getByRole('note')).toHaveTextContent(/debt/i)
    },
}

// A transfer in review cannot be changed at all: nothing opens, the reason is a message.
export const EditOfATransferInReviewIsRefused: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit transfer in review')
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

// A transaction the server does not let the user edit is not opened.
export const EditRefused: Story = {
    args: { editable: false },
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'Edit')
        await new Promise((resolve) => setTimeout(resolve, 300))
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

// What the page sends is kept by the spy, so the story can look at it.
const created = spy('post', '/transactions', transaction())

export const CreateSent: Story = {
    beforeEach: ({ msw }) => {
        created.bodies.length = 0
        msw.use(created.handler)
    },
    play: async ({ canvasElement }) => {
        await open(canvasElement, 'New income')
        const dialog = await screen.findByRole('dialog', {}, { timeout: 5000 })
        await within(dialog).findByRole('spinbutton', { name: /amount/i })
        await userEvent.click(within(dialog).getByRole('button', { name: /^create$/i }))
        await waitFor(() => expect(created.bodies).toHaveLength(1))
        await expect(created.bodies[0]).toMatchObject({
            type: 'income',
            payments: [{ account_id: 1, amount: 10 }],
            lines: [{ category_id: 2, amount: 10 }],
        })
    },
}
