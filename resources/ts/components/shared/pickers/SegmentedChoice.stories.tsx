import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { ArrowDownLeft, ArrowUpRight, Boxes } from 'lucide-react'
import { expect, fn, userEvent, within } from 'storybook/test'

import { SegmentedChoice } from './SegmentedChoice'

type Kind = 'income' | 'expense' | 'space'

const options = [
    { value: 'income', label: 'Income', icon: ArrowDownLeft, color: 'text-green-600' },
    { value: 'expense', label: 'Expense', icon: ArrowUpRight, color: 'text-red-600' },
] as const

const meta = {
    title: 'shared/pickers/SegmentedChoice',
    component: SegmentedChoice<Kind>,
    args: { value: 'income', options, onChange: fn() },
    render: (args) => {
        const [value, setValue] = useState(args.value)
        return <SegmentedChoice<Kind> {...args} value={value} onChange={(next) => { setValue(next); args.onChange(next) }} />
    },
} satisfies Meta<typeof SegmentedChoice<Kind>>

export default meta
type Story = StoryObj<typeof meta>

export const FirstSelected: Story = {}

export const SecondSelected: Story = { args: { value: 'expense' } }

export const Disabled: Story = { args: { disabled: true } }

export const WithIconOnlyOption: Story = {
    args: { options: [...options, { value: 'space', label: 'To another space', icon: Boxes, iconOnly: true }] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: 'To another space' })).toBeVisible()
    },
}

export const LongLabels: Story = {
    args: {
        options: [
            { value: 'income', label: 'A very long income label' },
            { value: 'expense', label: 'A very long expense label' },
        ],
    },
}

export const Changed: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Expense' }))
        await expect(args.onChange).toHaveBeenCalledWith('expense')
    },
}

export const DisabledDoesNotChange: Story = {
    args: { disabled: true },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: 'Expense' }), { pointerEventsCheck: 0 })
        await expect(args.onChange).not.toHaveBeenCalled()
    },
}
