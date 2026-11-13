import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { Plus } from 'lucide-react'
import { Page, PageHeader } from '@/components/shared'
import { AccountFormDialog, AccountList, AccountsTabs, ReconcileButton, ReconcileDialog } from '@/components/features/accounts'
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

interface AccountsPageProps {
    /** The Inactive tab: only the accounts that are switched off. The Accounts tab leaves them out. */
    inactive?: boolean
}

export function AccountsView({ inactive = false }: AccountsPageProps) {
    const { t } = useTranslation('pages')
    const { data: accounts, isLoading } = useAccounts({ exclude_debts: true })
    const deleteAccount = useDeleteAccount()
    const createAccount = useCreateAccount()
    const updateAccount = useUpdateAccount()
    const reorderAccounts = useReorderAccounts()
    const isReadOnly = useReadOnly()
    const all = accounts ?? []
    const items = all.filter((account) => account.isActive !== inactive)
    const [reconcileOpen, setReconcileOpen] = useState(false)
    const form = useResourceFormDialog<Account, AccountFormData>({
        items: all,
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
                actions={
                    isReadOnly ? undefined : (
                        <ReconcileButton onClick={() => setReconcileOpen(true)} />
                    )
                }
            />

            <div className="mx-auto w-full max-w-[800px]">
                <AccountsTabs active={inactive ? 'inactive' : 'accounts'} />
                <AccountList
                    accounts={items}
                    isLoading={isLoading}
                    emptyTitle={inactive ? t('accounts.inactiveEmptyTitle') : t('accounts.emptyTitle')}
                    emptyDescription={inactive ? t('accounts.inactiveEmptyDescription') : t('accounts.emptyDescription')}
                    emptyAction={
                        !isReadOnly && !inactive ? (
                            <Button onClick={form.openCreate}>
                                <Plus className="size-4" />
                                {t('accounts.create')}
                            </Button>
                        ) : undefined
                    }
                    onEdit={form.openEdit}
                    onDelete={(id) => deleteAccount.mutate(id)}
                    onReorder={
                        isReadOnly || inactive
                            ? undefined
                            : (reordered) => {
                                  // The hidden accounts keep their places after the shown ones.
                                  const hidden = all.filter((account) => !account.isActive)
                                  reorderAccounts.mutate([...reordered, ...hidden].map((account) => account.id))
                              }
                    }
                    isReadOnly={isReadOnly}
                />
            </div>

            <ReconcileDialog accounts={all.filter((account) => account.isActive)} open={reconcileOpen} onOpenChange={setReconcileOpen} />

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
