import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Card } from '@/components/ui/card'
import type { ApiPermission } from '@/types'

import { ConsentForm } from './ConsentForm'

const options = {
    resources: ['transactions', 'accounts'],
    presets: [
        { name: 'read_only' as const, permissions: ['accounts:read', 'transactions:read'] as ApiPermission[] },
        { name: 'full' as const, permissions: ['accounts:write', 'transactions:write'] as ApiPermission[] },
    ],
}
const spaces = [{ id: 1, name: 'Home' }, { id: 2, name: 'Trip' }]

type Args = Partial<React.ComponentProps<typeof ConsentForm>>

// The page owns the choices; the story keeps them so that ticking something shows.
function Stateful(args: Args) {
    const [permissions, setPermissions] = useState<ApiPermission[]>(args.permissions ?? ['transactions:read'])
    const [spaceChoice, setSpaceChoice] = useState(args.spaceChoice ?? { all: true, ids: [] })
    return (
        <Card className="max-w-lg">
            <ConsentForm
                spaces={spaces}
                options={options}
                changes={[]}
                host="claude.ai"
                error={null}
                busy={null}
                onDecide={fn()}
                onAsk={fn()}
                {...args}
                permissions={permissions}
                onPermissionsChange={setPermissions}
                spaceChoice={spaceChoice}
                onSpaceChoiceChange={setSpaceChoice}
            />
        </Card>
    )
}

const meta = {
    tags: ['desktop-only'],
    title: 'features/oauth/ConsentForm',
    component: Stateful,
    parameters: { layout: 'padded' },
    args: { onDecide: fn(), onAsk: fn() },
} satisfies Meta<typeof Stateful>

export default meta
type Story = StoryObj<typeof meta>

export const ReadOnly: Story = {}
export const InRussian: Story = { globals: { locale: 'ru' } }
export const NoHost: Story = { args: { host: null } }
export const OptionsStillLoading: Story = { args: { options: null } }
export const WithError: Story = { args: { error: 'The request has expired' } }
export const AllowBusy: Story = { args: { busy: 'allow' } }
export const DenyBusy: Story = { args: { busy: 'deny' } }

export const ChangesData: Story = {
    args: { permissions: ['transactions:write'], changes: ['transactions'] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByText(/read-only/i)).toBeNull()
    },
}

export const NoSpaceChosen: Story = {
    args: { spaceChoice: { all: false, ids: [] } },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /allow access/i })).toBeDisabled()
    },
}

export const NoPermissions: Story = {
    args: { permissions: [] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('button', { name: /allow access/i })).toBeDisabled()
    },
}

export const Allowed: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /allow access/i }))
        await expect(args.onDecide).toHaveBeenCalledWith(true)
    },
}

export const Denied: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /^cancel$/i }))
        await expect(args.onDecide).toHaveBeenCalledWith(false)
    },
}

// Allowing something that changes data asks once more instead of deciding.
export const ChangesAskFirst: Story = {
    args: { permissions: ['transactions:write'], changes: ['transactions'] },
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /allow access/i }))
        await expect(args.onAsk).toHaveBeenCalled()
        await expect(args.onDecide).not.toHaveBeenCalled()
    },
}

export const PickingASpace: Story = {
    args: { spaceChoice: { all: false, ids: [] } },
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('checkbox', { name: /trip/i }))
        await expect(within(canvasElement).getByRole('button', { name: /allow access/i })).toBeEnabled()
    },
}
