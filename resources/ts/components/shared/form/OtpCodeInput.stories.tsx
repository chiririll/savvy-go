import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { OtpCodeInput } from './OtpCodeInput'

const meta = {
    tags: ['desktop-only'],
    title: 'shared/form/OtpCodeInput',
    component: OtpCodeInput,
    args: { value: '', onChange: fn(), onComplete: fn() },
    render: (args) => {
        const [value, setValue] = useState(args.value)
        return <OtpCodeInput {...args} value={value} onChange={(next) => { setValue(next); args.onChange(next) }} />
    },
} satisfies Meta<typeof OtpCodeInput>

export default meta
type Story = StoryObj<typeof meta>

export const Empty: Story = {}
export const Partial: Story = { args: { value: '123' } }
export const Complete: Story = { args: { value: '123456' } }

export const Completed: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox'), '246810')
        await expect(args.onComplete).toHaveBeenCalledOnce()
    },
}

export const PartialDoesNotComplete: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox'), '2468')
        await expect(args.onComplete).not.toHaveBeenCalled()
    },
}
