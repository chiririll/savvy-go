import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import type { RecurringFormData } from '@/schemas'
import { overlayA11y } from '@/test/a11y'
import { FormHarness } from '@/test/form-harness'

import { ScheduleSection } from './ScheduleSection'

const base = {
    frequency: 'monthly',
    interval: 1,
    day_of_week: 1,
    day_of_month: 15,
    start_date: '2026-01-01',
    end_date: null,
} as Partial<RecurringFormData>

const meta = {
    tags: ['desktop-only'],
    title: 'features/recurring/ScheduleSection',
    component: ScheduleSection,
    parameters: { a11y: overlayA11y },
    args: { control: undefined as never, frequency: 'monthly' },
    render: (args) => (
        <FormHarness<RecurringFormData> defaultValues={{ ...base, frequency: args.frequency } as RecurringFormData}>
            {(control) => <ScheduleSection control={control} frequency={args.frequency} />}
        </FormHarness>
    ),
} satisfies Meta<typeof ScheduleSection>

export default meta
type Story = StoryObj<typeof meta>

export const Monthly: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('spinbutton', { name: /day of the month|day of month/i })).toBeVisible()
    },
}

export const Weekly: Story = {
    args: { frequency: 'weekly' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('combobox', { name: /day of the week|day of week/i })).toBeVisible()
    },
}

export const Daily: Story = {
    args: { frequency: 'daily' },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByRole('spinbutton', { name: /day of/i })).toBeNull()
        await expect(canvas.queryByRole('combobox', { name: /day of/i })).toBeNull()
    },
}

export const Yearly: Story = { args: { frequency: 'yearly' } }
export const EndDateIsOptional: Story = {
    play: async ({ canvasElement }) => {
        const end = within(canvasElement).getByLabelText(/end/i)
        await expect(end).toHaveValue('')
        await userEvent.type(end, '2027-01-01')
        await expect(end).toHaveValue('2027-01-01')
    },
}
