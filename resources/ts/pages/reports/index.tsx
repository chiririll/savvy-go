import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
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

            {/* Tab Navigation: a scrollable pill row on phones */}
            <Tabs value={activeTab} onValueChange={(val) => setActiveTab(val as ReportTab)} className="mb-4">
                <TabsList className="h-10 w-full justify-start gap-1 overflow-x-auto md:h-9 md:w-fit [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
                    {TABS.map(tab => (
                        <TabsTrigger key={tab} value={tab} className="flex-none px-4 md:px-3">
                            {reportTabLabel(t, tab)}
                        </TabsTrigger>
                    ))}
                </TabsList>
            </Tabs>

            {/* Global Filters Bar */}
            <FiltersBar
                filters={filters}
                onFilterChange={updateFilter}
                onToggleArrayFilter={toggleArrayFilter}
                onReset={resetFilters}
            />

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
