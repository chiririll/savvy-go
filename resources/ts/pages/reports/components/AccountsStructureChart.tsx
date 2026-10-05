import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNetWorth } from '@/hooks'
import type { ReportFilters } from '../types'
import { StructureChart } from './StructureChart'

// Shades of the net-worth blue, from dark (largest account) to light; light enough
// to tell apart, dark enough to keep white labels readable on the treemap.
const HUE = 217
const LIGHTNESS_FROM = 32
const LIGHTNESS_TO = 66

function blueShade(index: number, count: number) {
    const step = count > 1 ? index / (count - 1) : 0
    const lightness = LIGHTNESS_FROM + (LIGHTNESS_TO - LIGHTNESS_FROM) * step
    return `hsl(${HUE}, 85%, ${lightness.toFixed(0)}%)`
}

/** Assets by account. Accounts with a negative balance can't take a share of the whole, so they are left out. */
export function AccountsStructureChart({ filters }: { filters: ReportFilters }) {
    const { t } = useTranslation('pages')
    const { data, isLoading } = useNetWorth(filters)

    const items = useMemo(() => {
        const accounts = (data?.accounts ?? []).filter((account) => account.balance > 0)
        return accounts.map((account, index) => ({
            name: account.name,
            value: account.balance,
            color: blueShade(index, accounts.length),
        }))
    }, [data])
    const total = useMemo(() => items.reduce((sum, item) => sum + item.value, 0), [items])

    return (
        <StructureChart
            title={t('reports.netWorth.accountsBreakdown')}
            subtitle={t('reports.netWorth.distribution')}
            noData={t('reports.netWorth.noAccounts')}
            items={items}
            total={total}
            currency={data?.currency}
            isLoading={isLoading}
        />
    )
}
