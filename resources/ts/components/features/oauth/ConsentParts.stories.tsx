import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent } from 'storybook/test'

import { Card } from '@/components/ui/card'
import { overlayA11y } from '@/test/a11y'

import { ConsentConfirmDialog, ConsentHeader } from './ConsentParts'

const allow = fn()
const close = fn()

const meta = {
    title: 'features/oauth/ConsentParts',
    parameters: { a11y: overlayA11y },
    beforeEach: () => {
        allow.mockClear()
        close.mockClear()
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Header: Story = {
    render: () => (
        <Card className="max-w-lg">
            <ConsentHeader title="Claude wants access" description="to your Go Savvy data" />
        </Card>
    ),
}

export const HeaderInRussian: Story = { ...Header, globals: { locale: 'ru' } }

const dialog = (props: Partial<React.ComponentProps<typeof ConsentConfirmDialog>> = {}) => (
    <ConsentConfirmDialog
        open
        client="Claude"
        changes={['transactions']}
        deletes={[]}
        onOpenChange={close}
        onAllow={allow}
        {...props}
    />
)

export const ConfirmChange: Story = { render: () => dialog() }
export const ConfirmDelete: Story = { render: () => dialog({ deletes: ['transactions', 'accounts'] }) }
export const ConfirmInRussian: Story = { render: () => dialog(), globals: { locale: 'ru' } }
export const ConfirmClosed: Story = { render: () => dialog({ open: false }) }

export const ConfirmAllowed: Story = {
    render: () => dialog(),
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /allow/i }))
        await expect(allow).toHaveBeenCalled()
    },
}

export const ConfirmBack: Story = {
    render: () => dialog(),
    play: async () => {
        await userEvent.click(await screen.findByRole('button', { name: /back/i }))
        await expect(close).toHaveBeenCalledWith(false)
    },
}
