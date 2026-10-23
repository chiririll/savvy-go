import { useState } from 'react'
import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, screen, userEvent, waitFor, within } from 'storybook/test'

import { overlayA11y } from '@/test/a11y'
import { reply } from '@/test/api-mocks'
import { accounts, categories, currencies, tags } from '@/test/fixtures'
import { toggleIdInArray } from '@/lib/ids'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS, type ReportFilters } from '../types'
import { FiltersBar } from './FiltersBar'

// The page keeps the filters; this stands in for it and prints what it holds.
function Page({ initial = DEFAULT_FILTERS }: { initial?: ReportFilters }) {
    const [filters, setFilters] = useState(initial)
    return (
        <>
            <FiltersBar
                filters={filters}
                onFilterChange={(key, value) => setFilters((f) => ({ ...f, [key]: value }))}
                onToggleArrayFilter={(key, id) => setFilters((f) => ({ ...f, [key]: toggleIdInArray(f[key], id) }))}
                onReset={() => setFilters((f) => ({ ...f, accountIds: [], categoryIds: [], tagIds: [] }))}
            />
            <output data-testid="state">
                {filters.periodType}|{filters.compareWith}|{filters.accountIds.join(',')}|{filters.categoryIds.join(',')}|
                {filters.tagIds.join(',')}
            </output>
        </>
    )
}

const meta = {
    title: 'pages/reports/FiltersBar',
    component: Page,
    parameters: { a11y: overlayA11y, layout: 'padded' },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/accounts', accounts), reply('get', '/categories', categories), reply('get', '/tags', tags))
    },
} satisfies Meta<typeof Page>

export default meta
type Story = StoryObj<typeof meta>

// The bar is a row on a desktop and a sheet behind a button on a phone.
const isPhone = () => window.innerWidth < 768

export const Default: Story = {
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText('Last 30 days')).toBeVisible()
    },
}

export const InRussian: Story = { globals: { locale: 'ru' } }

export const PicksAPeriodType: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await userEvent.click(await canvas.findByText('YTD'))
        await expect(canvas.getByTestId('state')).toHaveTextContent(/^ytd\|/)
    },
}

export const BadgesAreButtons: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        const ytd = await canvas.findByRole('button', { name: 'YTD' })
        await expect(canvas.getByRole('button', { name: 'Last 30 days' })).toHaveAttribute('aria-pressed', 'true')
        await expect(ytd).toHaveAttribute('aria-pressed', 'false')
        ytd.focus()
        await expect(ytd).toHaveFocus()
        await userEvent.keyboard('{Enter}')
        await expect(canvas.getByTestId('state')).toHaveTextContent(/^ytd\|/)
        await expect(ytd).toHaveAttribute('aria-pressed', 'true')
    },
}

export const MonthShowsItsPicker: Story = {
    render: () => <Page initial={{ ...DEFAULT_FILTERS, periodType: 'month' }} />,
    play: async ({ canvasElement }) => {
        if (isPhone()) return
        // A month select sits next to the badges.
        await expect(await within(canvasElement).findAllByRole('combobox')).not.toHaveLength(0)
    },
}

export const CustomRangeShowsDates: Story = {
    render: () => <Page initial={{ ...DEFAULT_FILTERS, periodType: 'custom', customStartDate: '2030-01-01', customEndDate: '2030-01-31' }} />,
    play: async ({ canvasElement }) => {
        if (isPhone()) return
        const canvas = within(canvasElement)
        await waitFor(() => expect(canvas.getByDisplayValue('2030-01-01')).toBeVisible())
        await expect(canvas.getByDisplayValue('2030-01-31')).toBeVisible()
    },
}

export const ChangesComparison: Story = {
    play: async ({ canvasElement }) => {
        if (isPhone()) return
        const canvas = within(canvasElement)
        await userEvent.click((await canvas.findAllByRole('combobox'))[0])
        await userEvent.click(await screen.findByRole('option', { name: /same period last year/i }))
        await expect(canvas.getByTestId('state')).toHaveTextContent('same_period_last_year')
    },
}

export const FiltersByAccountAndResets: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        if (isPhone()) {
            await userEvent.click(await canvas.findByRole('button'))
            return
        }
        await userEvent.click(await canvas.findByRole('button', { name: /accounts/i }))
        await userEvent.click(await screen.findByText(accounts[0].name))
        await expect(canvas.getByTestId('state')).toHaveTextContent(/\|1\|/)
        // The count shows on the button, and a reset appears.
        await userEvent.click(await canvas.findByRole('button', { name: /reset/i }))
        await expect(canvas.getByTestId('state')).not.toHaveTextContent(/\|1\|/)
        await waitFor(() => expect(canvas.queryByRole('button', { name: /reset/i })).toBeNull())
    },
}

export const SheetOnAPhone: Story = {
    play: async ({ canvasElement }) => {
        if (!isPhone()) return
        const canvas = within(canvasElement)
        const open = (await canvas.findAllByRole('button')).at(-1)!
        await userEvent.click(open)
        const sheet = within(await screen.findByRole('dialog'))
        await expect(sheet.getByText('Filters')).toBeVisible()
        await userEvent.click(await sheet.findByText(categories[0].name))
        await expect(canvas.getByTestId('state')).toHaveTextContent(new RegExp(`\\|${categories[0].id}\\|`))
        await userEvent.click(sheet.getByRole('button', { name: /reset filters/i }))
        await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
    },
}
