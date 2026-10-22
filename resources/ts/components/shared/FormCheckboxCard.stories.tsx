import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { FormHarness } from '@/test/form-harness'

import { FormCheckboxCard } from './FormCheckboxCard'

type Values = { active: boolean }

const meta = {
    title: 'shared/FormCheckboxCard',
    component: FormCheckboxCard<Values>,
    args: { control: undefined as never, name: 'active', label: 'Active' },
} satisfies Meta<typeof FormCheckboxCard<Values>>

export default meta
type Story = StoryObj<typeof meta>

const harness =
    (props: Record<string, unknown>, active = false): Story['render'] =>
    () => (
        <FormHarness<Values> defaultValues={{ active }}>
            {(control) => <FormCheckboxCard<Values> control={control} name="active" label="Active" {...props} />}
        </FormHarness>
    )

export const Unchecked: Story = { render: harness({}) }
export const Checked: Story = { render: harness({}, true) }
export const WithDescription: Story = { render: harness({ description: 'Inactive accounts are hidden.' }) }
export const Disabled: Story = { render: harness({ disabled: true }) }
export const ForcedOn: Story = { render: harness({ forceChecked: true, disabled: true }) }

export const Toggled: Story = {
    render: harness({}),
    play: async ({ canvasElement }) => {
        const box = within(canvasElement).getByRole('checkbox', { name: 'Active' })
        await userEvent.click(box)
        await expect(box).toBeChecked()
    },
}
