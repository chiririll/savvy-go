import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import type { Action, Condition, ConditionGroup } from '@/types/automation'

import { ActionBuilder } from './ActionBuilder'
import { ConditionBuilder } from './ConditionBuilder'
import { pickerApi } from './fixtures'

const changed = fn()

const Conditions = ({ initial }: { initial: Condition[] }) => {
    const [value, setValue] = useState<ConditionGroup>({ match: 'all', conditions: initial })
    return (
        <ConditionBuilder
            value={value}
            onChange={(next) => {
                setValue(next)
                changed(next)
            }}
        />
    )
}

const Actions = ({ initial }: { initial: Action[] }) => {
    const [value, setValue] = useState(initial)
    return (
        <ActionBuilder
            value={value}
            onChange={(next) => {
                setValue(next)
                changed(next)
            }}
        />
    )
}

// The condition and action editors of a rule.
const meta = {
    title: 'features/automation/Builders',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        changed.mockClear()
        msw.use(...pickerApi())
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

const lastCall = () => changed.mock.calls[changed.mock.calls.length - 1][0]

export const NoConditions: Story = {
    render: () => <Conditions initial={[]} />,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('button', { name: /add condition/i }))
        await expect(lastCall()).toEqual({
            match: 'all',
            conditions: [{ field: 'type', op: 'equals', value: 'expense' }],
        })
    },
}

// One of each kind of value editor.
export const EveryValueEditor: Story = {
    render: () => (
        <Conditions
            initial={[
                { field: 'type', op: 'in', value: ['expense'] },
                { field: 'amount', op: 'between', value: [10, 100] },
                { field: 'description', op: 'contains', value: 'shop' },
                { field: 'account_id', op: 'equals', value: 2 },
                { field: 'category_id', op: 'is_null', value: null },
                { field: 'tags', op: 'has_any', value: [1] },
            ]}
        />
    ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('spinbutton', { name: 'From' })).toHaveValue(10)
        await expect(canvas.getByRole('spinbutton', { name: 'To' })).toHaveValue(100)
        await expect(canvas.getByDisplayValue('shop')).toBeVisible()
    },
}

export const MatchAny: Story = {
    render: () => <Conditions initial={[{ field: 'type', op: 'equals', value: 'income' }]} />,
    play: async ({ canvasElement }) => {
        const any = within(canvasElement).getByRole('button', { name: /match any/i })
        await expect(any).toHaveAttribute('aria-pressed', 'false')
        await userEvent.click(any)
        await expect(lastCall().match).toBe('any')
        await expect(any).toHaveAttribute('aria-pressed', 'true')
    },
}

// A new field starts from its first operator, with no value.
export const FieldChanged: Story = {
    render: () => <Conditions initial={[{ field: 'type', op: 'equals', value: 'expense' }]} />,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getByRole('combobox', { name: 'Field' }))
        await userEvent.click(await screen.findByRole('option', { name: /amount/i }))
        await expect(lastCall().conditions).toEqual([{ field: 'amount', op: 'equals', value: null }])
        // Only the operators of that field are offered.
        await userEvent.click(within(canvasElement).getByRole('combobox', { name: 'Condition' }))
        await expect(await screen.findAllByRole('option')).toHaveLength(7)
    },
}

export const TypesPicked: Story = {
    render: () => <Conditions initial={[{ field: 'type', op: 'in', value: [] }]} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('button', { name: /^income$/i }))
        await userEvent.click(canvas.getByRole('button', { name: /^transfer$/i }))
        await expect(lastCall().conditions[0].value).toEqual(['income', 'transfer'])
        await userEvent.click(canvas.getByRole('button', { name: /^income$/i }))
        await expect(lastCall().conditions[0].value).toEqual(['transfer'])
    },
}

export const AmountTyped: Story = {
    render: () => <Conditions initial={[{ field: 'amount', op: 'gt', value: 0 }]} />,
    play: async ({ canvasElement }) => {
        const field = within(canvasElement).getByRole('spinbutton', { name: 'Value' })
        await userEvent.clear(field)
        await userEvent.type(field, '250')
        await expect(lastCall().conditions[0].value).toBe(250)
    },
}

export const ConditionRemoved: Story = {
    render: () => (
        <Conditions
            initial={[
                { field: 'type', op: 'equals', value: 'expense' },
                { field: 'description', op: 'contains', value: 'shop' },
            ]}
        />
    ),
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /remove condition/i })[0])
        await expect(lastCall().conditions).toEqual([{ field: 'description', op: 'contains', value: 'shop' }])
    },
}

export const NoActions: Story = {
    render: () => <Actions initial={[]} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText(/no actions yet/i)).toBeVisible()
        await userEvent.click(canvas.getByRole('button', { name: /add action/i }))
        await expect(lastCall()).toEqual([{ type: 'set_category' }])
    },
}

export const EveryAction: Story = {
    render: () => (
        <Actions
            initial={[
                { type: 'set_category', category_id: 1 },
                { type: 'add_tags', tag_ids: [1, 2] },
                { type: 'remove_tags', tag_ids: [] },
                { type: 'set_description', template: 'Shop: {description}' },
                { type: 'create_transfer', to_account_id: 2, amount_formula: 'amount * 0.1' },
            ]}
        />
    ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getAllByRole('combobox', { name: 'Action' })).toHaveLength(5)
        await expect(canvas.getByDisplayValue('amount * 0.1')).toBeVisible()
    },
}

export const DescriptionTemplateTyped: Story = {
    render: () => <Actions initial={[{ type: 'set_description' }]} />,
    play: async ({ canvasElement }) => {
        await userEvent.type(within(canvasElement).getByPlaceholderText(/description template/i), 'Rent')
        await expect(lastCall()).toEqual([{ type: 'set_description', template: 'Rent' }])
    },
}

export const ActionRemoved: Story = {
    render: () => <Actions initial={[{ type: 'set_category' }, { type: 'add_tags' }]} />,
    play: async ({ canvasElement }) => {
        await userEvent.click(within(canvasElement).getAllByRole('button', { name: /remove action/i })[1])
        await expect(lastCall()).toEqual([{ type: 'set_category' }])
    },
}
