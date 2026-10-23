import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, waitFor, within } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { EntityFormDialog } from './EntityFormDialog'

type Entity = { id: number; name: string }
type Values = { name: string }

const toFormValues = (entity: Entity): Partial<Values> => ({ name: entity.name })

// A bare form wired the way the real ones are: it reads defaultValues and reports every change.
const meta = {
    title: 'shared/dialogs/EntityFormDialog',
    component: EntityFormDialog<Entity, Values>,
    args: {
        open: true,
        onOpenChange: fn(),
        onSubmit: fn(),
        formId: 'entity-form',
        title: 'New tag',
        toFormValues,
        children: ({ formKey, formProps }) => (
            <form
                key={formKey}
                id={formProps.formId}
                className="space-y-2"
                onSubmit={(event) => {
                    event.preventDefault()
                    formProps.onSubmit({ name: new FormData(event.currentTarget).get('name') as string })
                }}
            >
                <label className="text-sm" htmlFor="entity-name">
                    Name
                </label>
                <input
                    id="entity-name"
                    name="name"
                    className="w-full rounded-md border px-3 py-2"
                    defaultValue={formProps.defaultValues?.name ?? ''}
                    onChange={(event) => formProps.onValuesChange?.({ name: event.target.value })}
                />
            </form>
        ),
    },
} satisfies Meta<typeof EntityFormDialog<Entity, Values>>

export default meta
type Story = StoryObj<typeof meta>

export const Create: Story = {
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByLabelText('Name')).toHaveValue('')
        await expect(within(dialog).getByRole('button', { name: /create/i })).toBeVisible()
    },
}

// Every form dialog wraps this one, so the closed state is asserted here once.
export const Closed: Story = {
    args: { open: false },
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

export const CreateWithFallback: Story = {
    args: { fallbackValues: { name: 'Groceries' } },
    play: async () => {
        await expect(within(await screen.findByRole('dialog')).getByLabelText('Name')).toHaveValue('Groceries')
    },
}

export const Edit: Story = {
    args: { title: 'Edit tag', entity: { id: 7, name: 'Travel' } },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await expect(within(dialog).getByLabelText('Name')).toHaveValue('Travel')
        await expect(within(dialog).getByRole('button', { name: /^save$/i })).toBeVisible()
    },
}

export const SubmitSendsTheForm: Story = {
    play: async ({ args }) => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.type(within(dialog).getByLabelText('Name'), 'Rent')
        await userEvent.click(within(dialog).getByRole('button', { name: /create/i }))
        await waitFor(() => expect(args.onSubmit).toHaveBeenCalledWith({ name: 'Rent' }))
    },
}

export const Submitting: Story = {
    args: { isSubmitting: true },
    play: async () => {
        await expect(await screen.findByRole('button', { name: /saving/i })).toBeDisabled()
    },
}

// A half-filled create form survives closing and opening the dialog again.
export const DraftKeptWhenReopened: Story = {
    render: function Render(args) {
        const [open, setOpen] = useState(true)
        return (
            <>
                <Button onClick={() => setOpen(true)}>Open</Button>
                <EntityFormDialog<Entity, Values> {...args} open={open} onOpenChange={setOpen} />
            </>
        )
    },
    play: async () => {
        const dialog = await screen.findByRole('dialog')
        await userEvent.type(within(dialog).getByLabelText('Name'), 'Draft')
        await userEvent.click(within(dialog).getByRole('button', { name: /cancel/i }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
        await userEvent.click(screen.getByRole('button', { name: 'Open' }))
        await expect(within(await screen.findByRole('dialog')).getByLabelText('Name')).toHaveValue('Draft')
    },
}
