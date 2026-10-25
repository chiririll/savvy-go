import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Page, PageHeader } from '@/components/shared'
import { AccountFormDialog, AccountList } from '@/components/features/accounts'
import { Button } from '@/components/ui/button'
import {
    useAccounts,
    useCreateAccount,
    useDeleteAccount,
    useReorderAccounts,
    useUpdateAccount,
    useResourceFormDialog,
} from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import type { Account } from '@/types'
import type { AccountFormData } from '@/schemas'

export default function AccountsPage() {
    const { t } = useTranslation('pages')
    const { data: accounts, isLoading } = useAccounts({ exclude_debts: true })
    const deleteAccount = useDeleteAccount()
    const createAccount = useCreateAccount()
    const updateAccount = useUpdateAccount()
    const reorderAccounts = useReorderAccounts()
    const isReadOnly = useReadOnly()
    const items = accounts ?? []
    const form = useResourceFormDialog<Account, AccountFormData>({
        items,
        isLoading,
        create: createAccount,
        update: updateAccount,
    })

    return (
        <Page title={t('accounts.title')}>
            <PageHeader
                title={t('accounts.title')}
                description={t('accounts.description')}
                createLabel={t('accounts.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
            />

            <div className="mx-auto w-full max-w-[800px]">
                <AccountList
                    accounts={items}
                    isLoading={isLoading}
                    emptyTitle={t('accounts.emptyTitle')}
                    emptyDescription={t('accounts.emptyDescription')}
                    emptyAction={
                        !isReadOnly ? (
                            <Button onClick={form.openCreate}>
                                <Plus className="size-4" />
                                {t('accounts.create')}
                            </Button>
                        ) : undefined
                    }
                    onEdit={form.openEdit}
                    onDelete={(id) => deleteAccount.mutate(id)}
                    onReorder={
                        isReadOnly
                            ? undefined
                            : (reordered) => {
                                  reorderAccounts.mutate(reordered.map((account) => account.id))
                              }
                    }
                    isReadOnly={isReadOnly}
                />
            </div>

            <AccountFormDialog
                account={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
            />
        </Page>
    )
}
