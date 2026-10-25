import type { Meta, StoryObj } from '@storybook/react-vite'
import { expect, userEvent, within } from 'storybook/test'

import type { ActivityHeatmapData } from '@/api/reports'
import { fail, hang, reply } from '@/test/api-mocks'
import { currencies } from '@/test/fixtures'
import { useCurrenciesStore } from '@/stores/currencies'

import { DEFAULT_FILTERS } from '../types'
import { ActivityHeatmap } from './ActivityHeatmap'

const days = (currency: string | null, values: number[]): ActivityHeatmapData => ({
    items: values.map((value, i) => ({
        date: `2030-01-${String(i + 1).padStart(2, '0')}`,
        value,
        count: value > 0 ? 2 : 0,
    })),
    max: Math.max(0, ...values),
    currency,
})

const meta = {
    title: 'pages/reports/ActivityHeatmap',
    component: ActivityHeatmap,
    parameters: { layout: 'padded' },
    args: { filters: DEFAULT_FILTERS },
    beforeEach: ({ msw }) => {
        useCurrenciesStore.getState().setAll(currencies)
        msw.use(reply('get', '/reports/activity-heatmap', days('USD', [0, 12.5, 40.25, 0, 99.99, 5, 70])))
    },
} satisfies Meta<typeof ActivityHeatmap>

export default meta
type Story = StoryObj<typeof meta>

const cells = (root: HTMLElement) => root.querySelectorAll('.grid.size-full > div')

const hoverDay = async (canvasElement: HTMLElement, index: number) => {
    await within(canvasElement).findByText('Less')
    await userEvent.hover(cells(canvasElement)[index])
}

const tooltip = async (text: string | RegExp) => (await within(document.body).findAllByText(text))[0]

export const Default: Story = {
    play: async ({ canvasElement }) => {
        const canvas = within(canvasElement)
        await expect(await canvas.findByText('Less')).toBeVisible()
        await expect(canvas.getByText('More')).toBeVisible()
        await expect(cells(canvasElement)).toHaveLength(7)
    },
}

export const TooltipShowsTheAmount: Story = {
    play: async ({ canvasElement }) => {
        await hoverDay(canvasElement, 4)
        await expect(await tooltip('99.99 $')).toBeVisible()
    },
}

export const TooltipOfAQuietDay: Story = {
    play: async ({ canvasElement }) => {
        await hoverDay(canvasElement, 0)
        await expect(await tooltip(/no expenses/i)).toBeVisible()
    },
}

export const InYen: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/activity-heatmap', days('JPY', [0, 1200, 56_000]))),
    play: async ({ canvasElement }) => {
        await hoverDay(canvasElement, 2)
        await expect(await tooltip('56,000 ¥')).toBeVisible()
    },
}

export const InDinar: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/activity-heatmap', days('KWD', [0, 1.5, 12.345]))),
    play: async ({ canvasElement }) => {
        await hoverDay(canvasElement, 2)
        await expect(await tooltip(/12\.345/)).toBeVisible()
    },
}

// A month of nothing still draws its days, all of them grey.
export const NothingSpent: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/activity-heatmap', days('USD', [0, 0, 0, 0]))),
    play: async ({ canvasElement }) => {
        await within(canvasElement).findByText('Less')
        await expect(canvasElement.querySelectorAll('.grid.size-full > div.bg-muted')).toHaveLength(4)
    },
}

export const Empty: Story = {
    beforeEach: ({ msw }) => msw.use(reply('get', '/reports/activity-heatmap', days('USD', []))),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/no data/i)).toBeVisible()
    },
}

export const Failed: Story = {
    beforeEach: ({ msw }) => msw.use(fail('get', '/reports/activity-heatmap')),
    play: async ({ canvasElement }) => {
        await expect(await within(canvasElement).findByText(/heatmap/i)).toBeVisible()
    },
}

export const Loading: Story = { beforeEach: ({ msw }) => msw.use(hang('get', '/reports/activity-heatmap')) }
