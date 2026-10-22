import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent } from 'storybook/test'

import { Button } from '@/components/ui/button'

import { ResponsiveDialog } from './ResponsiveDialog'

// A dialog on a wide screen and a bottom sheet on a narrow one; the story tests run at both sizes.
const meta = {
    title: 'shared/dialogs/ResponsiveDialog',
    component: ResponsiveDialog,
    args: {
        open: true,
        onOpenChange: fn(),
        title: 'Share space',
        description: 'Invite people by email.',
        children: <p>Body of the dialog</p>,
        footer: <Button>Send</Button>,
    },
} satisfies Meta<typeof ResponsiveDialog>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const WithoutDescription: Story = { args: { description: undefined } }
export const WithoutFooter: Story = { args: { footer: undefined } }
export const WithoutBody: Story = { args: { children: undefined } }
export const Closed: Story = {
    args: { open: false },
    play: async () => {
        await expect(screen.queryByRole('dialog')).toBeNull()
    },
}

export const LongBody: Story = {
    args: { children: Array.from({ length: 40 }, (_, i) => <p key={i}>Line {i + 1}</p>) },
    play: async () => {
        // The footer must stay on screen while the body scrolls.
        await expect(await screen.findByRole('button', { name: 'Send' })).toBeVisible()
    },
}

export const Dismissed: Story = {
    play: async ({ args }) => {
        await screen.findByRole('dialog')
        await userEvent.keyboard('{Escape}')
        await expect(args.onOpenChange).toHaveBeenCalledWith(false)
    },
}
