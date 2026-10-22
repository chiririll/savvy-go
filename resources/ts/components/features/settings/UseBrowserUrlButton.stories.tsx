import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Input } from '@/components/ui/input'

import { UseBrowserUrlButton } from './UseBrowserUrlButton'

const url = 'https://savvy.example.com'

const meta = {
    title: 'features/settings/UseBrowserUrlButton',
    component: UseBrowserUrlButton,
    args: { url, current: '', onUse: fn() },
    render: (args) => (
        <div className="relative max-w-sm">
            <Input aria-label="Server URL" defaultValue={args.current} className="pr-10" />
            <UseBrowserUrlButton {...args} />
        </div>
    ),
} satisfies Meta<typeof UseBrowserUrlButton>

export default meta
type Story = StoryObj<typeof meta>

export const Offered: Story = {}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const OtherAddress: Story = { args: { current: 'http://localhost:8080' } }

export const AlreadyThisAddress: Story = {
    args: { current: url },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

// Stray spaces in the field do not make it a different address.
export const AlreadyThisAddressPadded: Story = {
    args: { current: `  ${url} ` },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('button')).toBeNull()
    },
}

export const Used: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button'))
        await expect(args.onUse).toHaveBeenCalledWith(url)
    },
}
