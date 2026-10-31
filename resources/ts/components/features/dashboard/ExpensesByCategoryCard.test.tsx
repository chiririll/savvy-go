// @vitest-environment jsdom
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { ExpensesByCategoryCard } from './ExpensesByCategoryCard'

vi.mock('react-i18next', async (importActual) => ({
    ...(await importActual<typeof import('react-i18next')>()),
    useTranslation: () => ({ t: (key: string) => key, i18n: { language: 'en' } }),
}))

vi.mock('@/components/shared/charts/ReactECharts', () => ({ default: () => null }))
vi.mock('@/lib/chart-theme', () => ({ useChartTheme: () => ({ tooltip: {}, surface: '#fff' }), legendTextStyle: () => ({}) }))

afterEach(cleanup)

describe('ExpensesByCategoryCard', () => {
    it('shows the empty state when the period has no data (null list)', () => {
        const summary = { data: null, total: 0, currency: 'USD' } as never
        render(<ExpensesByCategoryCard summary={summary} currency={undefined} />)
        expect(screen.getByText('dashboard.noDataPeriod')).toBeTruthy()
    })

    it('shows an error, not "no data", when the summary failed to load', () => {
        render(<ExpensesByCategoryCard summary={undefined} isError currency={undefined} />)
        expect(screen.getByRole('alert').textContent).toBe('common:errors.somethingWrong')
        expect(screen.queryByText('dashboard.noDataPeriod')).toBeNull()
    })
})
