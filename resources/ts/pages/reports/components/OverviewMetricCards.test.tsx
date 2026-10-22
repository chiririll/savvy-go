// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { OverviewMetricCards, type OverviewMetricSpec } from './OverviewMetricCards'
import type { OverviewMetrics } from '@/api/reports'

vi.mock('./MetricCard', () => ({
    MetricCard: (p: { title: string; value: number; currency?: string | null }) => (
        <div data-testid="card">{`${p.title}|${p.value}|${p.currency ?? '-'}`}</div>
    ),
}))

const metric = (value: number) => ({ value, previous: null, sparkline: [] })
const data = {
    income: metric(10),
    expenses: metric(4),
    netCashFlow: metric(6),
    savingsRate: metric(60),
    currency: 'EUR',
} as OverviewMetrics

const specs: OverviewMetricSpec[] = [
    { key: 'income', title: 'In', type: 'income' },
    { key: 'savingsRate', title: 'Rate', type: 'percent' },
]

describe('OverviewMetricCards', () => {
    afterEach(cleanup)

    it('renders one card per spec, with the currency except for a percent', () => {
        render(<OverviewMetricCards metrics={specs} data={data} isLoading={false} compareWith="none" className="" />)
        expect(screen.getAllByTestId('card').map((c) => c.textContent)).toEqual(['In|10|EUR', 'Rate|60|-'])
    })

    it('renders a skeleton per spec while loading', () => {
        const { container } = render(
            <OverviewMetricCards metrics={specs} isLoading compareWith="none" className="" />,
        )
        expect(screen.queryAllByTestId('card')).toHaveLength(0)
        expect(container.querySelectorAll('[data-slot="skeleton"]')).toHaveLength(specs.length)
    })
})
