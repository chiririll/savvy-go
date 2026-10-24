import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import type { ApiPermission } from '@/types'

import { PermissionPicker } from './PermissionPicker'

const options = {
    resources: ['transactions', 'accounts', 'budgets'],
    presets: [
        { name: 'viewer', permissions: ['accounts:read', 'budgets:read', 'transactions:read'] as ApiPermission[] },
        { name: 'editor', permissions: ['accounts:write', 'budgets:write', 'transactions:write'] as ApiPermission[] },
        { name: 'admin', permissions: ['accounts:delete', 'budgets:delete', 'transactions:delete'] as ApiPermission[] },
    ],
}

type Props = Partial<React.ComponentProps<typeof PermissionPicker>>

// The parent keeps the value, as the dialogs do.
function Stateful({ value: initial = options.presets[0].permissions, ...rest }: Props) {
    const [value, setValue] = useState<ApiPermission[]>(initial)
    return (
        <div className="max-w-lg">
            <PermissionPicker value={value} onChange={setValue} options={options} {...rest} />
            <p data-testid="value">{[...value].sort().join(',')}</p>
        </div>
    )
}

const meta = {
    tags: ['desktop-only'],
    title: 'features/api-tokens/PermissionPicker',
    component: Stateful,
} satisfies Meta<typeof Stateful>

export default meta
type Story = StoryObj<typeof meta>

export const Viewer: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('tab', { name: /viewer/i })).toHaveAttribute('aria-selected', 'true')
    },
}
export const Editor: Story = { args: { value: options.presets[1].permissions } }
export const Custom: Story = {
    args: { value: ['transactions:read'] },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('tab', { name: /custom/i })).toHaveAttribute('aria-selected', 'true')
    },
}
export const Nothing: Story = { args: { value: [] } }

// An app that may not delete is not offered the preset that does.
export const CappedAtWrite: Story = {
    args: { maxLevel: 'write' },
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).queryByRole('tab', { name: /admin/i })).toBeNull()
    },
}

export const PickingAPreset: Story = {
    play: async ({ canvasElement }) => {
        const picker = within(canvasElement)
        await userEvent.click(picker.getByRole('tab', { name: /editor/i }))
        await expect(picker.getByTestId('value')).toHaveTextContent('accounts:write,budgets:write,transactions:write')
    },
}

export const PickingCustomKeepsTheRights: Story = {
    play: async ({ canvasElement }) => {
        const picker = within(canvasElement)
        await userEvent.click(picker.getByRole('tab', { name: /custom/i }))
        await expect(picker.getByRole('tab', { name: /custom/i })).toHaveAttribute('aria-selected', 'true')
        await expect(picker.getByTestId('value')).toHaveTextContent('accounts:read,budgets:read,transactions:read')
    },
}

export const DetailsOpened: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /details|permissions/i }))
        await expect(await within(canvasElement).findAllByRole('checkbox')).not.toHaveLength(0)
    },
}
