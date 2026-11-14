import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { AccountsTabs, ReconcileButton, ReconcileDialog, ReconciliationDayList } from '@/components/features/accounts'
import { useAccounts, useCurrencies, useAllReconciliationDays } from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'

/** The Reconciliations tab of the Accounts section. */
export default function ReconciliationsPage() {
    const { t } = useTranslation('pages')
    const isReadOnly = useReadOnly()
    const [reconcileOpen, setReconcileOpen] = useState(false)
    const { data: accounts } = useAccounts({ exclude_debts: true })
    const { data: currencies } = useCurrencies()
    const baseCurrency = currencies?.find((currency) => currency.isBase)
    const { data: days, isLoading } = useAllReconciliationDays()
    const items = accounts ?? []
    const active = items.filter((account) => account.isActive)

    return (
        <Page title={t('accounts.title')}>
            <PageHeader
                title={t('accounts.title')}
                description={t('accounts.description')}
                actions={isReadOnly ? undefined : <ReconcileButton onClick={() => setReconcileOpen(true)} />}
            />
            <div className="mx-auto w-full max-w-[800px]">
                <AccountsTabs active="reconciliations">
                    <ReconciliationDayList
                        days={days ?? []}
                        accounts={items}
                        baseCurrency={baseCurrency?.code}
                        isLoading={isLoading}
                    />
                </AccountsTabs>
            </div>
            <ReconcileDialog accounts={active} open={reconcileOpen} onOpenChange={setReconcileOpen} />
        </Page>
    )
}
