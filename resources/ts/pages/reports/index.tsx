import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Activity } from 'lucide-react'
import { FiltersBar } from './components'
import { OverviewTab, CashFlowTab, ExpensesTab, IncomeTab, NetWorthTab } from './tabs'
import { toggleIdInArray } from '@/lib/utils'
import { DEFAULT_FILTERS, TABS, type ReportFilters, type ReportTab } from './types'
import { reportTabLabel } from '@/lib/labels'

export default function ReportsPage() {
    const { t } = useTranslation('pages')
    const [filters, setFilters] = useState<ReportFilters>(DEFAULT_FILTERS)
    const [activeTab, setActiveTab] = useState<ReportTab>('overview')

    const updateFilter = <K extends keyof ReportFilters>(key: K, value: ReportFilters[K]) => {
        setFilters(f => ({ ...f, [key]: value }))
    }

    const toggleArrayFilter = (key: 'accountIds' | 'categoryIds' | 'tagIds', id: number) => {
        setFilters(f => {
            const current = f[key]
            const newIds = toggleIdInArray(current, id)
            return { ...f, [key]: newIds }
        })
    }

    const resetFilters = () => {
        setFilters(DEFAULT_FILTERS)
    }

    return (
        <Page title={t('reports.title')}>
            <PageHeader
                title={t('reports.title')}
                description={t('reports.description')}
            />

            {/* Global Filters Bar */}
            <FiltersBar
                filters={filters}
                onFilterChange={updateFilter}
                onToggleArrayFilter={toggleArrayFilter}
                onReset={resetFilters}
            />

            {/* Tab Navigation */}
            <Tabs value={activeTab} onValueChange={(val) => setActiveTab(val as ReportTab)} className="mb-6">
                <TabsList className="h-auto flex-wrap md:flex-nowrap md:h-9 md:w-fit">
                    {TABS.map(tab => (
                        <TabsTrigger key={tab} value={tab}>
                            {reportTabLabel(t, tab)}
                        </TabsTrigger>
                    ))}
                </TabsList>
            </Tabs>

            {/* Tab Content */}
            {activeTab === 'overview' && (
                <OverviewTab filters={filters} />
            )}

            {activeTab === 'cashflow' && (
                <CashFlowTab filters={filters} />
            )}

            {activeTab === 'expenses' && (
                <ExpensesTab filters={filters} />
            )}

            {activeTab === 'income' && (
                <IncomeTab filters={filters} />
            )}

            {activeTab === 'networth' && (
                <NetWorthTab filters={filters} />
            )}
        </Page>
    )
}
