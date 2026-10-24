import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'

import { CategoriesStep } from './CategoriesStep'
import type { DraftContent } from './draft'

const seeded: DraftContent = {
    baseCurrency: 'EUR',
    currencies: [],
    accounts: [],
    categories: [
        { name: '#OTHER', type: 'expense', icon: '📦', color: '#6b7280', isDefault: true },
        { name: 'Groceries', type: 'expense', icon: '🛒', color: '#16a34a', isDefault: false },
        { name: '#OTHER_INCOME', type: 'income', icon: '💰', color: '#2563eb', isDefault: true },
    ],
    tags: [],
}

const changed = fn()

const Harness = () => {
    const [content, setContent] = useState(seeded)
    return (
        <CategoriesStep
            content={content}
            onChange={(patch) => {
                setContent((c) => ({ ...c, ...patch }))
                changed(patch)
            }}
        />
    )
}

const meta = {
    title: 'features/spaces/create/CategoriesStep',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: () => changed.mockClear(),
    render: () => <Harness />,
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Seeded: Story = {
    play: async ({ canvasElement }) => {
        // The default category of each type stays.
        await expect(within(canvasElement).getByRole('button', { name: /remove other$/i })).toBeDisabled()
    },
}

// Grouped by type, with the add form pinned below the list.
export const Grouped: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByRole('heading', { name: 'Expenses' })).toBeVisible()
        await expect(canvas.getByRole('heading', { name: 'Incomes' })).toBeVisible()
        await expect(canvas.getByTestId('create-space-category-form')).toBeVisible()
    },
}

// On a phone the closed type select shows the icon only, the opened one icon and name.
export const TypeSelectOnAPhone: Story = {
    tags: ['mobile-only'],
    play: async ({ canvasElement }) => {
        const trigger = within(canvasElement).getByRole('combobox', { name: 'Type' })
        await expect(trigger.querySelector('[data-type-name]')).not.toBeVisible()
        await userEvent.click(trigger)
        await expect(await screen.findByRole('option', { name: /income/i })).toBeVisible()
    },
}

export const CategoryAddedAndRemoved: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(canvas.getByRole('combobox', { name: 'Type' }))
        await userEvent.click(await screen.findByRole('option', { name: /income/i }))
        await userEvent.type(canvas.getByRole('textbox', { name: /new category/i }), ' Bonus {Enter}')
        await expect(changed).toHaveBeenLastCalledWith({
            categories: [
                ...seeded.categories,
                { name: 'Bonus', type: 'income', icon: '🏷️', color: '#6b7280', isDefault: false },
            ],
        })
        await userEvent.click(canvas.getByRole('button', { name: 'Remove Groceries' }))
        await expect(canvas.queryByDisplayValue('Groceries')).toBeNull()
    },
}

export const BlankCategoryIsNotAdded: Story = {
    play: async ({ canvasElement }) => {
        await userEvent.type(within(canvasElement).getByRole('textbox', { name: /new category/i }), '   {Enter}')
        await expect(changed).not.toHaveBeenCalled()
    },
}
