import type { Meta, StoryObj } from '@storybook/react-vite'
import { Bell } from 'lucide-react'
import { expect, fn, userEvent, within } from 'storybook/test'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

import { SettingsRowSkeleton, SettingsSection, ToggleRow } from './SettingsSection'

const meta = {
    title: 'shared/SettingsSection',
    component: SettingsSection,
    args: { icon: Bell, title: 'Notifications', description: 'Choose what you are told about.', children: 'Body' },
} satisfies Meta<typeof SettingsSection>

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {}
export const WithoutDescription: Story = { args: { description: undefined } }
export const WithActions: Story = { args: { actions: <Button size="sm">Save</Button> } }
export const LongText: Story = {
    args: {
        title: 'A section title long enough to need more than one line on a narrow screen',
        description: 'And a description that goes on for a while, so the header has to wrap it somewhere sensible.',
        actions: <Button size="sm">Save changes</Button>,
    },
}

export const Toggles: Story = {
    render: (args) => (
        <SettingsSection {...args} contentClassName="p-0 divide-y">
            <ToggleRow id="a" label="Weekly digest" description="A summary every Monday." checked onChange={fn()} />
            <ToggleRow id="b" label="Budget alerts" description="When a budget is nearly spent." checked={false} onChange={fn()} />
            <ToggleRow
                id="c"
                label="Large expenses"
                description="Locked by the administrator."
                checked
                disabled
                badge={<Badge variant="secondary">Managed</Badge>}
                onChange={fn()}
            />
            <SettingsRowSkeleton />
        </SettingsSection>
    ),
}

export const ToggleChanges: Story = {
    render: (args) => (
        <SettingsSection {...args}>
            <ToggleRow id="a" label="Weekly digest" description="A summary every Monday." checked={false} onChange={fn()} />
        </SettingsSection>
    ),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('switch', { name: 'Weekly digest' }))
        await expect(within(canvasElement).getByRole('switch')).toBeVisible()
    },
}
