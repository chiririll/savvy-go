import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { FormHarness } from '@/test/form-harness'

import { FormInputField } from './FormInputField'

type Values = { name: string; note: string | null; limit: number | null }

const meta = {
    tags: ['desktop-only'],
    title: 'shared/form/FormInputField',
    component: FormInputField<Values>,
    args: { control: undefined as never, name: 'name', label: 'Name' },
} satisfies Meta<typeof FormInputField<Values>>

export default meta
type Story = StoryObj<typeof meta>

type Errors = Partial<Record<keyof Values, string>>

const harness =
    (props: Record<string, unknown>, values: Partial<Values> = {}, errors?: Errors): Story['render'] =>
    () => (
        <FormHarness<Values> defaultValues={{ name: '', note: null, limit: null, ...values }} errors={errors}>
            {(control) => <FormInputField<Values> control={control} name="name" label="Name" {...props} />}
        </FormHarness>
    )

export const Empty: Story = { render: harness({}) }
export const Filled: Story = { render: harness({}, { name: 'Groceries' }) }
export const WithDescription: Story = { render: harness({ description: 'Shown in lists.' }) }
export const WithError: Story = {
    render: harness({}, { name: '' }, { name: 'Name is required' }),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Name is required')).toBeVisible()
    },
}
export const Multiline: Story = { render: harness({ multiline: true, name: 'note' }, { note: 'Two\nlines' }) }
export const Disabled: Story = { render: harness({ inputProps: { disabled: true } }, { name: 'Locked' }) }

export const Typed: Story = {
    render: harness({}),
    play: async ({ canvasElement }) => {
        const input = within(canvasElement).getByRole('textbox', { name: 'Name' })
        await userEvent.type(input, 'Rent')
        await expect(input).toHaveValue('Rent')
    },
}

export const NullableStaysNullWhenEmpty: Story = {
    render: harness({ name: 'limit', nullable: true, inputProps: { type: 'number' } }),
    play: async ({ canvasElement }) => {
        const input = within(canvasElement).getByRole('spinbutton', { name: 'Name' })
        await userEvent.type(input, '12')
        await expect(input).toHaveValue(12)
        await userEvent.clear(input)
        await expect(input).toHaveValue(null)
    },
}
