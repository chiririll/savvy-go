import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { CategoryFormData } from '@/schemas'
import { Category, CategoryType } from '@/types'
import { CategoryForm } from './CategoryForm'
import { useCategories } from '@/hooks'
import { localizeDefaultName, toStoredDefaultName } from '@/lib/format/localized-name'

const FORM_ID = 'category-form'

interface CategoryFormDialogProps {
    category?: Category | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: CategoryFormData) => void
    isSubmitting?: boolean
    defaultType?: CategoryType
}

export function CategoryFormDialog({
    category,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
    defaultType,
}: CategoryFormDialogProps) {
    const { t } = useTranslation('pages')
    const { data: all } = useCategories()
    // Groups go one level deep: only a top-level category without children can be put under another.
    const hasChildren = category ? (all ?? []).some((c) => c.parentId === category.id) : false
    const parentOptions = hasChildren
        ? []
        : (all ?? []).filter((c) => c.parentId == null && c.id !== category?.id)

    return (
        <EntityFormDialog<Category, CategoryFormData, CategoryFormData>
            entity={category}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={(data) => onSubmit({
                ...data,
                name: toStoredDefaultName(data.name, category?.name),
            })}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={category ? t('categories.editTitle') : t('categories.createTitle')}
            description={t('categories.description')}
            fallbackValues={defaultType ? { type: defaultType } : undefined}
            toFormValues={(item) => ({
                name: localizeDefaultName(item.name),
                type: item.type,
                icon: item.icon ?? undefined,
                color: item.color ?? undefined,
                parentId: item.parentId,
            })}
        >
            {({ formKey, formProps }) => (
                <CategoryForm key={formKey} {...formProps} isDefault={category?.isDefault} parentOptions={parentOptions} />
            )}
        </EntityFormDialog>
    )
}
