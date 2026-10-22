import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'
import { accounts, categories, currencies, tags, transaction } from '@/test/fixtures'
import { reply, spy } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { useCurrenciesStore } from '@/stores/currencies'

// The provider loads the dialog when first asked; having it in already keeps a busy test run from timing out.
import './TransactionFormDialog'
import { CreateTransactionProvider, useTransactionFormDialog } from './CreateTransactionProvider'

const Buttons = ({ editable = true }: { editable?: boolean }) => {
    const { openCreate, openEdit } = useTransactionFormDialog()
    const existing = transaction({ status: 'confirmed', actions: { edit: editable, duplicate: true, delete: true, confirm: false, skip: false } })

    return (
        <div className="flex gap-2">
            <Button onClick={() => openCreate()}>New</Button>
            <Button onClick={() => openCreate({ type: 'income', account_id: 1, category_id: 2, amount: 10 })}>New income</Button>
            <Button variant="outline" onClick={() => openEdit(existing)}>Edit</Button>
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

export const Closed: Story = {}

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
        await expect(created.bodies[0]).toMatchObject({ type: 'income', amount: 10 })
    },
}
