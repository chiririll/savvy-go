import { describe, expect, it } from 'vitest'

import { pages } from '@/app/pages'
import { TABS, reportTabUrl } from './types'

describe('reportTabUrl', () => {
    it('gives the overview the bare /reports address', () => {
        expect(reportTabUrl('overview')).toBe('/reports')
    })

    it('gives every other report its own address below /reports', () => {
        expect(reportTabUrl('cashflow')).toBe('/reports/cashflow')
        expect(reportTabUrl('networth')).toBe('/reports/networth')
    })

    it('matches the pages the app registers', () => {
        expect(TABS.map(reportTabUrl)).toEqual([
            pages.reports.path,
            pages.reportsCashFlow.path,
            pages.reportsExpenses.path,
            pages.reportsIncome.path,
            pages.reportsNetWorth.path,
        ])
    })

    it('never gives two reports the same address', () => {
        const urls = TABS.map(reportTabUrl)
        expect(new Set(urls).size).toBe(TABS.length)
    })
})
