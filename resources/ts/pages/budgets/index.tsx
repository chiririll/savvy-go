import { useTranslation } from 'react-i18next'
import { FeedList, Page, PageHeader } from '@/components/shared'
import { BudgetFormDialog, BudgetRow, budgetGroup, compareBudgets } from '@/components/features/budgets'
import { useBudgets, useCreateBudget, useDeleteBudget, useUpdateBudget, useResourceFormDialog } from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import type { Budget } from '@/types'
import type { BudgetFormData } from '@/schemas'

export default function BudgetsPage() {
    const { t } = useTranslation('pages')
    const { data: budgets, isLoading } = useBudgets()
    const deleteBudget = useDeleteBudget()
    const createBudget = useCreateBudget()
    const updateBudget = useUpdateBudget()
    const isReadOnly = useReadOnly()
    const items = [...(budgets ?? [])].sort(compareBudgets)
    const form = useResourceFormDialog<Budget, BudgetFormData>({
        items,
        isLoading,
        create: createBudget,
        update: updateBudget,
    })

    return (
        <Page title={t('budgets.title')}>
            <PageHeader
                title={t('budgets.title')}
                description={t('budgets.description')}
                createLabel={t('budgets.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
            />

            <div className="mx-auto w-full max-w-[800px]">
                <FeedList
                    items={items}
                    isLoading={isLoading}
                    emptyTitle={t('table.emptyTitle', { ns: 'common' })}
                    emptyDescription={t('table.emptyDescription', { ns: 'common' })}
                    onCreate={form.openCreate}
                    createLabel={t('budgets.create')}
                    isReadOnly={isReadOnly}
                    getKey={(budget) => budget.id}
                    groupBy={budgetGroup}
                >
                    {(budget) => (
                        <BudgetRow
                            budget={budget}
                            onEdit={form.openEdit}
                            onDelete={(id) => deleteBudget.mutate(id)}
                            isReadOnly={isReadOnly}
                        />
                    )}
                </FeedList>
            </div>

            <BudgetFormDialog
                budget={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
            />
        </Page>
    )
}
