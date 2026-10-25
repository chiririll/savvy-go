import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { FieldHelp } from './FieldHelp'

const meta = {
    tags: ['desktop-only'],
    title: 'shared/form/FieldHelp',
    component: FieldHelp,
    args: { children: 'The rate used to convert amounts into the base currency.' },
} satisfies Meta<typeof FieldHelp>

export default meta
type Story = StoryObj<typeof meta>

export const Closed: Story = {}

export const Opened: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(await screen.findByText(args.children as string)).toBeVisible()
    },
}

export const LongText: Story = {
    args: { children: 'A much longer explanation. '.repeat(12) },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await screen.findByText(/much longer explanation/)
    },
}
