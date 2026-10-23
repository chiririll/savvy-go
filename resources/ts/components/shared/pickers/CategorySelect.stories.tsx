import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, within } from 'storybook/test'

import { categories } from '@/test/fixtures'
import { hang, reply } from '@/test/api-mocks'
import { overlayA11y } from '@/test/a11y'
import { withForm } from '@/test/form-decorator'

import { CategorySelect } from './CategorySelect'

const meta = {
    title: 'shared/pickers/CategorySelect',
    component: CategorySelect,
    parameters: { a11y: overlayA11y },
    decorators: [withForm({ label: 'Category' })],
    args: { onChange: () => {} },
    render: (args) => {
        const [value, setValue] = useState(args.value)
        return <CategorySelect {...args} value={value} onChange={setValue} />
    },
    beforeEach: ({ msw }) => msw.use(reply('get', '/categories', categories)),
} satisfies Meta<typeof CategorySelect>

export default meta
type Story = StoryObj<typeof meta>

const open = async (canvasElement: HTMLElement) => {
    const trigger = within(canvasElement).getByRole('combobox')
    await userEvent.click(trigger)
    return trigger
}

export const Empty: Story = {}
export const Selected: Story = { args: { value: 2 } }
export const Disabled: Story = { args: { disabled: true, value: 1 } }
export const CustomPlaceholder: Story = { args: { placeholder: 'Any category' } }
export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/categories')) }
export const Plain: Story = {
    args: { plain: true },
    decorators: [(Story) => <label>Category <Story /></label>],
}

export const MostUsedFirst: Story = {
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        const options = await screen.findAllByRole('option')
        await expect(options[0]).toHaveTextContent('Transport')
    },
}

export const KeepsGivenOrder: Story = {
    args: { sortByPopularity: false },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        const options = await screen.findAllByRole('option')
        await expect(options[0]).toHaveTextContent('Groceries')
    },
}

export const OnlyExpense: Story = {
    args: { type: 'expense' },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await screen.findAllByRole('option')
        await expect(screen.queryByRole('option', { name: /Salary/ })).toBeNull()
    },
}

export const OnlyIncome: Story = {
    args: { type: 'income' },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findAllByRole('option')).toHaveLength(1)
    },
}

export const WithExcluded: Story = {
    args: { excludeIds: [1, 3] },
    play: async ({ canvasElement }) => {
        await open(canvasElement)
        await expect(await screen.findAllByRole('option')).toHaveLength(1)
    },
}

export const Picked: Story = {
    play: async ({ canvasElement }) => {
        const trigger = await open(canvasElement)
        await userEvent.click(await screen.findByRole('option', { name: /Salary/ }))
        await expect(trigger).toHaveTextContent('Salary')
    },
}
