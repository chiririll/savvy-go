import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { PeriodPicker } from './PeriodPicker'
import type { PeriodPreset } from './period'

const meta = {
    title: 'features/dashboard/PeriodPicker',
    component: PeriodPicker,
    parameters: { a11y: overlayA11y },
    args: {
        period: 'this_month',
        customStartDate: '2026-01-01',
        customEndDate: '2026-01-31',
        onPeriodChange: () => {},
        onCustomStartChange: () => {},
        onCustomEndChange: () => {},
    },
    render: (args) => {
        const [period, setPeriod] = useState<PeriodPreset>(args.period)
        return <PeriodPicker {...args} period={period} onPeriodChange={setPeriod} />
    },
} satisfies Meta<typeof PeriodPicker>

export default meta
type Story = StoryObj<typeof meta>

export const ThisMonth: Story = {}
export const Last30Days: Story = { args: { period: 'last_30_days' } }
export const Custom: Story = {
    args: { period: 'custom' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getAllByDisplayValue(/2026-01/)).toHaveLength(2)
    },
}

export const DatesOnlyForCustom: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByDisplayValue('2026-01-01')).toBeNull()
    },
}

export const ChoosingCustomShowsTheDates: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox'))
        await userEvent.click(await screen.findByRole('option', { name: /custom|period|range/i }))
        await expect(await within(canvasElement).findAllByDisplayValue(/2026-01/)).toHaveLength(2)
    },
}

export const AllPresetsOffered: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox'))
        await expect(await screen.findAllByRole('option')).toHaveLength(7)
    },
}
