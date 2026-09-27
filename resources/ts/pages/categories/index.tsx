import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { parseAsStringLiteral, useQueryState } from 'nuqs'
import { FeedList, Page, PageHeader } from '@/components/shared'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Button } from '@/components/ui/button'
import { CategoryFormDialog, CategoryRow } from '@/components/features/categories'
import {
    useCategories,
    useCreateCategory,
    useDeleteCategory,
    useSetDefaultCategory,
    useUpdateCategory,
    useResourceFormDialog,
} from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { Category, CategoryType } from '@/types'
import { CategoryFormData } from '@/schemas'

const categoryTypeParam = parseAsStringLiteral(['income', 'expense'] as const).withDefault('expense')

export default function CategoriesPage() {
    const { t } = useTranslation('pages')
    const [type, setType] = useQueryState('type', categoryTypeParam)
    const { data: categories, isLoading } = useCategories()
    const deleteCategory = useDeleteCategory()
    const setDefaultCategory = useSetDefaultCategory()
    const createCategory = useCreateCategory()
    const updateCategory = useUpdateCategory()
    const isReadOnly = useReadOnly()
    const items = categories ?? []
    const visibleItems = useMemo(
        () => items.filter((category) => category.type === type),
        [items, type],
    )
    const form = useResourceFormDialog<Category, CategoryFormData>({
        items,
        isLoading,
        create: createCategory,
        update: updateCategory,
    })

    const typeCounts = useMemo(() => ({
        income: items.filter((category) => category.type === 'income').length,
        expense: items.filter((category) => category.type === 'expense').length,
    }), [items])

    return (
        <Page title={t('categories.title')}>
            <PageHeader
                title={t('categories.title')}
                description={t('categories.description')}
                createLabel={t('categories.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
            />

            <div className="mx-auto w-full max-w-[800px]">
                <Tabs
                    value={type}
                    onValueChange={(value) => {
                        void setType(value as CategoryType)
                    }}
                    className="mb-4 w-full max-w-full"
                >
                    <TabsList className="grid h-auto w-full grid-cols-2 sm:inline-flex sm:h-9 sm:w-fit">
                        <TabsTrigger value="income" className="min-w-0">
                            {t('categories.tabs.income')}
                        </TabsTrigger>
                        <TabsTrigger value="expense" className="min-w-0">
                            {t('categories.tabs.expenses')}
                        </TabsTrigger>
                    </TabsList>
                </Tabs>

                <FeedList
                    items={visibleItems}
                    isLoading={isLoading}
                    emptyTitle={type === 'income' ? t('categories.emptyIncomeTitle') : t('categories.emptyExpenseTitle')}
                    emptyDescription={type === 'income' ? t('categories.emptyIncomeDescription') : t('categories.emptyExpenseDescription')}
                    emptyAction={
                        !isReadOnly ? (
                            <Button onClick={form.openCreate}>
                                <Plus className="size-4" />
                                {t('categories.create')}
                            </Button>
                        ) : undefined
                    }
                    getKey={(category) => category.id}
                >
                    {(category) => (
                        <CategoryRow
                            category={category}
                            onEdit={form.openEdit}
                            onDelete={(id, successorId) => deleteCategory.mutate({ id, successorId })}
                            onSetDefault={(id) => setDefaultCategory.mutate(id)}
                            isSettingDefault={setDefaultCategory.isPending}
                            isReadOnly={isReadOnly}
                            deleteDisabled={category.isDefault || typeCounts[category.type] <= 1}
                            deleteDisabledLabel={
                                category.isDefault
                                    ? t('common:actions.cannotDeleteDefault')
                                    : t('common:actions.cannotDeleteLast')
                            }
                        />
                    )}
                </FeedList>
            </div>

            <CategoryFormDialog
                category={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
                defaultType={type}
            />
        </Page>
    )
}
