import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { NavLink, Outlet } from 'react-router-dom'
import { Page, PageHeader } from '@/components/shared'
import { FiltersBar } from './components'
import { toggleIdInArray, cn } from '@/lib/utils'
import { DEFAULT_FILTERS, TABS, reportTabUrl, type ReportFilters } from './types'
import { reportTabLabel } from '@/lib/labels'

/** Each report is its own page under /reports; the filters above them are shared. */
export default function ReportsPage() {
    const { t } = useTranslation('pages')
    const [filters, setFilters] = useState<ReportFilters>(DEFAULT_FILTERS)

    const updateFilter = <K extends keyof ReportFilters>(key: K, value: ReportFilters[K]) => {
        setFilters((f) => ({ ...f, [key]: value }))
    }

    const toggleArrayFilter = (key: 'accountIds' | 'categoryIds' | 'tagIds', id: number) => {
        setFilters((f) => {
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
            <PageHeader title={t('reports.title')} description={t('reports.description')} />

            {/* Tab navigation: a scrollable pill row on phones */}
            <nav
                className={cn(
                    'mb-4 flex h-10 w-full items-center justify-start gap-1 overflow-x-auto rounded-lg bg-muted p-[3px]',
                    'md:h-9 md:w-fit [scrollbar-width:none] [&::-webkit-scrollbar]:hidden',
                )}
            >
                {TABS.map((tab) => (
                    <NavLink
                        key={tab}
                        to={reportTabUrl(tab)}
                        end
                        className={({ isActive }) =>
                            cn(
                                'inline-flex h-full flex-none items-center rounded-md px-4 text-sm font-medium',
                                'whitespace-nowrap text-muted-foreground transition-colors hover:text-foreground md:px-3',
                                isActive && 'bg-background text-foreground shadow-sm',
                            )
                        }
                    >
                        {reportTabLabel(t, tab)}
                    </NavLink>
                ))}
            </nav>

            {/* Global Filters Bar */}
            <FiltersBar
                filters={filters}
                onFilterChange={updateFilter}
                onToggleArrayFilter={toggleArrayFilter}
                onReset={resetFilters}
            />

            <Outlet context={filters} />
        </Page>
    )
}
