import { useTranslation } from 'react-i18next'
import { FeedList, Page, PageHeader } from '@/components/shared'
import { compareRecurring, recurringGroup, RecurringFormDialog, RecurringRow } from '@/components/features/recurring'
import { useCreateRecurring, useDeleteRecurring, useRecurring, useUpdateRecurring, useResourceFormDialog } from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { RecurringFormData } from '@/schemas'
import { RecurringTransaction } from '@/types'

export default function RecurringPage() {
    const { t } = useTranslation('pages')
    const { data: recurring, isLoading } = useRecurring()
    const deleteRecurring = useDeleteRecurring()
    const createRecurring = useCreateRecurring()
    const updateRecurring = useUpdateRecurring()
    const isReadOnly = useReadOnly()
    const items = [...(recurring ?? [])].sort(compareRecurring)
    const form = useResourceFormDialog<RecurringTransaction, RecurringFormData>({
        items,
        isLoading,
        create: createRecurring,
        update: updateRecurring,
    })

    return (
        <Page title={t('recurring.title')}>
            <PageHeader
                title={t('recurring.title')}
                description={t('recurring.description')}
                createLabel={t('recurring.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
            />

            <div className="mx-auto w-full max-w-[800px]">
                <FeedList
                    items={items}
                    isLoading={isLoading}
                    emptyTitle={t('table.emptyTitle', { ns: 'common' })}
                    emptyDescription={t('table.emptyDescription', { ns: 'common' })}
                    onCreate={form.openCreate}
                    createLabel={t('recurring.create')}
                    isReadOnly={isReadOnly}
                    getKey={(item) => item.id}
                    groupBy={recurringGroup}
                >
                    {(item) => (
                        <RecurringRow
                            recurring={item}
                            onEdit={form.openEdit}
                            onDelete={(id) => deleteRecurring.mutate(id)}
                            isReadOnly={isReadOnly}
                        />
                    )}
                </FeedList>
            </div>

            <RecurringFormDialog
                recurring={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
            />
        </Page>
    )
}
