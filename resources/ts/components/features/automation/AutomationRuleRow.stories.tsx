import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { AutomationRuleRow } from './AutomationRuleRow'
import { rule } from './fixtures'

const meta = {
    title: 'features/automation/AutomationRuleRow',
    component: AutomationRuleRow,
    parameters: { a11y: overlayA11y },
    args: { rule: rule(), onEdit: fn(), onDelete: fn(), onToggle: fn() },
} satisfies Meta<typeof AutomationRuleRow>

export default meta
type Story = StoryObj<typeof meta>

export const Active: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('#10')).toBeVisible()
        await expect(canvas.getByRole('checkbox')).toBeChecked()
    },
}

export const Inactive: Story = { args: { rule: rule({ isActive: false, runsCount: 0 }) } }
export const LongName: Story = {
    args: { rule: rule({ name: 'A rule with a very long name that explains far too much about what it does' }) },
}

export const Toggled: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('checkbox'))
        await expect(args.onToggle).toHaveBeenCalledWith(1)
        // The toggle does not open the rule.
        await expect(args.onEdit).not.toHaveBeenCalled()
    },
}

export const Opened: Story = {
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByText('Groceries from the shop'))
        await expect(args.onEdit).toHaveBeenCalledWith(args.rule)
    },
}

export const Deleted: Story = {
    tags: ['desktop-only'],
    play: async ({ canvasElement, args }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /more actions/i }))
        await expect(await screen.findByRole('menuitem', { name: /view logs/i })).toHaveAttribute(
            'href',
            '/automation/1/logs',
        )
        await userEvent.click(screen.getByRole('menuitem', { name: /delete/i }))
        await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: /delete/i }))
        await expect(args.onDelete).toHaveBeenCalledWith(1)
    },
}

export const ReadOnly: Story = {
    args: { isReadOnly: true },
    play: async ({ canvasElement, args }) => {
        await expect(within(canvasElement).getByRole('checkbox')).toBeDisabled()
        await userEvent.click(within(canvasElement).getByText('Groceries from the shop'))
        await expect(args.onEdit).not.toHaveBeenCalled()
    },
}
