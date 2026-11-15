import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { COLOR_OPTIONS } from '@/constants'

import { SpaceLookFields } from './SpaceLookFields'

const meta = {
    title: 'features/spaces/SpaceLookFields',
    component: SpaceLookFields,
    parameters: { layout: 'padded' },
    args: { emoji: '🏠', color: COLOR_OPTIONS[0], onChange: fn() },
    render: (args) => {
        const [look, setLook] = useState({ emoji: args.emoji, color: args.color })
        return (
            <SpaceLookFields
                {...args}
                {...look}
                onChange={(next) => {
                    setLook(next)
                    args.onChange(next)
                }}
            />
        )
    },
} satisfies Meta<typeof SpaceLookFields>

export default meta
type Story = StoryObj<typeof meta>

export const Chosen: Story = {}
export const NothingChosen: Story = {
    args: { emoji: '', color: '' },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByRole('button', { name: /remove emoji/i })).toBeNull()
        await expect(canvas.getByRole('button', { name: /no color/i })).toHaveAttribute('aria-pressed', 'true')
    },
}

export const ColorPicked: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: COLOR_OPTIONS[2] }))
        await expect(args.onChange).toHaveBeenLastCalledWith({ emoji: '🏠', color: COLOR_OPTIONS[2] })
    },
}

export const Cleared: Story = {
    play: async ({ canvasElement, args }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /remove emoji/i }))
        await expect(args.onChange).toHaveBeenLastCalledWith({ emoji: '', color: COLOR_OPTIONS[0] })
        await userEvent.click(canvas.getByRole('button', { name: /no color/i }))
        await expect(args.onChange).toHaveBeenLastCalledWith({ emoji: '', color: '' })
    },
}

// Without a name the preview shows the same letter the space will get.
export const NamedNoEmoji: Story = {
    args: { emoji: '', name: 'trip home' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByText('T')).toBeInTheDocument()
    },
}

// A member who is not an admin sees the look but cannot change it.
export const Disabled: Story = {
    args: { disabled: true },
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('button', { name: COLOR_OPTIONS[1] })).toBeDisabled()
        await expect(canvas.queryByRole('button', { name: /remove emoji/i })).toBeNull()
    },
}
