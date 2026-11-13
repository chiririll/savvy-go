import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { pages } from '@/app/pages'

interface AccountsTabsProps {
    active: 'accounts' | 'inactive' | 'reconciliations'
}

/** The tabs of the Accounts section: each one is a page of its own, so each has its own address. */
export function AccountsTabs({ active }: AccountsTabsProps) {
    const { t } = useTranslation('pages')
    return (
        <Tabs value={active} className="mb-4">
            <TabsList>
                <TabsTrigger value="accounts" asChild>
                    <Link to={pages.accounts.path}>{t('accounts.tabs.accounts')}</Link>
                </TabsTrigger>
                <TabsTrigger value="inactive" asChild>
                    <Link to={pages.accountsInactive.path}>{t('accounts.tabs.inactive')}</Link>
                </TabsTrigger>
                <TabsTrigger value="reconciliations" asChild>
                    <Link to={pages.accountsReconciliations.path}>{t('accounts.tabs.reconciliations')}</Link>
                </TabsTrigger>
            </TabsList>
        </Tabs>
    )
}
