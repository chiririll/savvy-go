import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { AutomationRuleFormDialog } from './AutomationRuleFormDialog'
import { pickerApi, rule } from './fixtures'

const meta = {
    title: 'features/automation/AutomationRuleFormDialog',
    component: AutomationRuleFormDialog,
    parameters: { a11y: overlayA11y },
    beforeEach: ({ msw }) => msw.use(...pickerApi()),
    args: { open: true, onOpenChange: fn(), onSubmit: fn() },
} satisfies Meta<typeof AutomationRuleFormDialog>

export default meta
type Story = StoryObj<typeof meta>

const dialog = async () => within(await screen.findByRole('dialog'))
const save = async () => userEvent.click((await dialog()).getByRole('button', { name: /^(save|create)$/i }))

export const Create: Story = {
    play: async () => {
        const d = await dialog()
        await expect(d.getByRole('spinbutton', { name: /priority/i })).toHaveValue(50)
        await expect(d.getByText(/no actions yet/i)).toBeVisible()
    },
}

export const Edit: Story = {
    args: { rule: rule() },
    play: async () => {
        const d = await dialog()
        await expect(d.getByDisplayValue('Groceries from the shop')).toBeVisible()
        await expect(d.getByDisplayValue('shop')).toBeVisible()
        await expect(d.getAllByRole('combobox', { name: 'Field' })).toHaveLength(2)
    },
}

export const NameRequired: Story = {
    play: async ({ args }) => {
        await save()
        await new Promise((resolve) => setTimeout(resolve, 200))
        await expect(args.onSubmit).not.toHaveBeenCalled()
        await expect((await dialog()).getByRole('textbox', { name: 'Name' })).toHaveAttribute('aria-invalid', 'true')
    },
}

export const SavedUnchanged: Story = {
    args: { rule: rule() },
    play: async ({ args }) => {
        await save()
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            name: 'Groceries from the shop',
            trigger_type: 'on_transaction_create',
            priority: 10,
            conditions: rule().conditions,
            actions: rule().actions,
            is_active: true,
            stop_processing: false,
        })
    },
}

export const TriggerAndStopChanged: Story = {
    args: { rule: rule() },
    play: async ({ args }) => {
        const d = await dialog()
        await userEvent.click(d.getByRole('combobox', { name: /trigger/i }))
        await userEvent.click(await screen.findByRole('option', { name: /update/i }))
        await userEvent.click(d.getByRole('switch', { name: /stop processing/i }))
        await save()
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalled())
        await expect((args.onSubmit as ReturnType<typeof fn>).mock.calls[0][0]).toMatchObject({
            trigger_type: 'on_transaction_update',
            stop_processing: true,
        })
    },
}

export const Submitting: Story = { args: { rule: rule(), isSubmitting: true } }
