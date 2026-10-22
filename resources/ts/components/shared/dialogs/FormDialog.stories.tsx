import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { FormHarness } from '@/test/form-harness'

import { FormActiveField } from '../form/FormActiveField'
import { FormEstimatedField } from '../form/FormEstimatedField'
import { FormInputField } from '../form/FormInputField'
import { FormDialog } from './FormDialog'

type Values = { name: string; is_active: boolean; is_estimated: boolean }

const Body = () => (
    <FormHarness<Values> defaultValues={{ name: '', is_active: true, is_estimated: false }}>
        {(control) => (
            <>
                <FormInputField<Values> control={control} name="name" label="Name" />
                <FormActiveField<Values> control={control} help="Inactive accounts are hidden from pickers." />
                <FormEstimatedField<Values> control={control} help="The amount is a guess, to be corrected later." />
            </>
        )}
    </FormHarness>
)

const meta = {
    title: 'shared/dialogs/FormDialog',
    component: FormDialog,
    args: {
        open: true,
        onOpenChange: fn(),
        title: 'New account',
        description: 'Where your money lives.',
        formId: 'account-form',
        children: <Body />,
    },
} satisfies Meta<typeof FormDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {}

export const Edit: Story = {
    args: { isEdit: true },
    play: async () => {
        await expect(await screen.findByRole('button', { name: /^save$/i })).toBeVisible()
    },
}

export const Submitting: Story = {
    args: { isSubmitting: true },
    play: async () => {
        await expect(await screen.findByRole('button', { name: /saving/i })).toBeDisabled()
    },
}

export const WithHeaderExtra: Story = { args: { headerExtra: <p className="text-sm">Created by Kirill</p> } }

export const WithoutDescription: Story = { args: { description: undefined } }

export const Closed: Story = {
    args: { open: false },
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

// A viewer may look but not save: the submit button is gone.
export const ReadOnly: Story = {
    parameters: { app: { role: 'viewer' } },
    play: async () => {
        await screen.findByRole('dialog')
        // The role arrives with the space, a moment after the first paint.
        await waitFor(() => expect(screen.queryByRole('button', { name: /create|save/i })).toBeNull())
    },
}

export const CancelClosesIt: Story = {
    play: async ({ args }) => {
        await userEvent.click(await screen.findByRole('button', { name: /cancel/i }))
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}

// The check boxes sit in the footer, beside the buttons.
export const FooterCheckboxes: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(await within(dialog).findByRole('checkbox', { name: /active/i })).toBeChecked()
        await expect(await within(dialog).findByRole('checkbox', { name: /approximate/i })).not.toBeChecked()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }
