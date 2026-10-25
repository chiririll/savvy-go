import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs'

const meta = {
    title: 'ui/Tabs',
    component: Tabs,
    args: { defaultValue: 'accounts' },
    render: (args) => (
        <Tabs {...args}>
            <TabsList>
                <TabsTrigger value="accounts">Accounts</TabsTrigger>
                <TabsTrigger value="budgets">Budgets</TabsTrigger>
                <TabsTrigger value="locked" disabled>Locked</TabsTrigger>
            </TabsList>
            <TabsContent value="accounts">Accounts panel</TabsContent>
            <TabsContent value="budgets">Budgets panel</TabsContent>
            <TabsContent value="locked">Locked panel</TabsContent>
        </Tabs>
    ),
} satisfies Meta<typeof Tabs>

export default meta
type Story = StoryObj<typeof meta>

export const FirstTab: Story = {}

export const SwitchedTab: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('tab', { name: 'Budgets' }))
        await expect(canvas.getByText('Budgets panel')).toBeVisible()
        await expect(canvas.queryByText('Accounts panel')).toBeNull()
    },
}

export const DisabledTab: Story = {
    play: async ({ canvasElement }) => {
        await expect(within(canvasElement).getByRole('tab', { name: 'Locked' })).toBeDisabled()
    },
}
