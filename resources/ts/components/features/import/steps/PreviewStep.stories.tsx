import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, fn, screen, userEvent, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'
import { categories } from '@/test/fixtures'
import type { ImportCategoryMap } from '@/types/import'

import { preview } from '../fixtures'
import { PreviewStep } from './PreviewStep'

const changed = fn()

// Existing categories are linked by default, missing ones are created, as the wizard starts them.
const initialMap = (result: ReturnType<typeof preview>): ImportCategoryMap =>
    Object.fromEntries(result.summary.categories.map((c) => [c.name, c.matchId ?? 'create']))

const Harness = ({ result = preview(), isLoading = false }) => {
    const [map, setMap] = useState(initialMap(result))
    return (
        <PreviewStep
            previewResult={result}
            categoryMap={map}
            onCategoryMapChange={(next) => {
                setMap(next)
                changed(next)
            }}
            isLoading={isLoading}
        />
    )
}

const meta = {
    title: 'features/import/PreviewStep',
    parameters: { layout: 'padded', a11y: overlayA11y },
    beforeEach: ({ msw }) => {
        changed.mockClear()
        msw.use(reply('get', '/categories', categories))
    },
} satisfies Meta

export default meta
type Story = StoryObj<typeof meta>

export const Preview: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('Will be created').previousElementSibling).toHaveTextContent('3')
        await expect(canvas.getByText(/1 new category will be created/i)).toBeVisible()
        await expect(canvas.getByText(/row 3: new currency jpy/i)).toBeVisible()
        await expect(canvas.getByText(/row 6: bad date/i)).toBeVisible()
        // With the category panel, new categories are not listed again under new items.
        await expect(canvas.getByText('Currencies:')).toBeVisible()
        await expect(canvas.queryByText('Categories:')).toBeNull()
    },
}

export const CategoryLinked: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByRole('combobox', { name: 'Eating out' }))
        // Only categories of the same type can take them.
        await expect(screen.queryByRole('option', { name: 'Salary' })).toBeNull()
        await userEvent.click(await screen.findByRole('option', { name: 'Transport' }))
        await expect(changed).toHaveBeenLastCalledWith({ Groceries: 1, 'Eating out': 3, Salary: 2 })
        await expect(canvas.queryByText(/will be created\./i)).toBeNull()
    },
}

export const CategorySkipped: Story = {
    render: () => <Harness />,
    play: async ({ canvasElement }) => {
        await userEvent.click(await within(canvasElement).findByRole('combobox', { name: 'Groceries' }))
        await userEvent.click(await screen.findByRole('option', { name: /no category/i }))
        await expect(changed).toHaveBeenLastCalledWith({ Groceries: 'skip', 'Eating out': 'create', Salary: 2 })
    },
}

// A file without a category column has nothing to map.
export const WithoutCategories: Story = {
    render: () => (
        <Harness result={preview({ categories: [], categoriesToCreate: ['Eating out'], currenciesToCreate: [] })} />
    ),
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.queryByText(/categories from the file/i)).toBeNull()
        await expect(canvas.getByText('Categories:')).toBeVisible()
    },
}

// A big file is checked from a sample, and the count is an estimate.
export const Sampled: Story = {
    render: () => <Harness result={preview({ totalRows: 250000, sampled: 1000 })} />,
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(canvas.getByText('~250,000')).toBeVisible()
        await expect(canvas.getByText(/checked a sample of 1,000 of 250,000 rows/i)).toBeVisible()
        await expect(canvas.getByText(/duplicates \(skipped\) in sample/i)).toBeVisible()
    },
}

export const Importing: Story = { render: () => <Harness isLoading /> }
