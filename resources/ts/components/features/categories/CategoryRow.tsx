import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Star, Trash2 } from 'lucide-react'
import { FeedRow, RowActions, CategorySelect } from '@/components/shared'
import { DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { categoryIconStyle } from '@/lib/category-color'
import { localizeDefaultName } from '@/lib/localized-name'
import { Category } from '@/types'

interface CategoryRowProps {
    category: Category
    onEdit?: (category: Category) => void
    onDelete?: (id: number, successorId?: number) => void
    onSetDefault?: (id: number) => void
    isSettingDefault?: boolean
    isReadOnly?: boolean
    deleteDisabled?: boolean
    deleteDisabledLabel?: string
}

export function CategoryRow({
    category,
    onEdit,
    onDelete,
    onSetDefault,
    isSettingDefault,
    isReadOnly,
    deleteDisabled,
    deleteDisabledLabel,
}: CategoryRowProps) {
    const { t } = useTranslation(['common', 'pages'])
    const name = localizeDefaultName(category.name)
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const canSetDefault = !!onSetDefault && !isReadOnly && !category.isDefault
    const needsReassign = canDelete && !deleteDisabled && (category.transactionsCount ?? 0) > 0

    return (
        <FeedRow
            icon={<span aria-hidden>{category.icon}</span>}
            iconStyle={categoryIconStyle(category.color)}
            title={name}
            badge={category.isDefault ? (
                <Star
                    className="size-3.5 shrink-0 fill-amber-400 text-amber-500"
                    aria-label={t('pages:categories.columns.default')}
                />
            ) : undefined}
            subtitle={t(`pages:categories.types.${category.type}`)}
            onOpen={canEdit ? () => onEdit(category) : undefined}
            hasActions={canEdit || canDelete || canSetDefault}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(category) : undefined}
                    onDelete={canDelete && !needsReassign ? () => onDelete(category.id) : undefined}
                    deleteTitle={t('pages:categories.deleteTitle')}
                    deleteDescription={t('pages:categories.deleteDescription', { name })}
                    deleteDisabled={deleteDisabled}
                    deleteDisabledLabel={deleteDisabledLabel}
                >
                    {canSetDefault && (
                        <DropdownMenuItem
                            disabled={isSettingDefault}
                            onClick={() => onSetDefault(category.id)}
                        >
                            <Star className="mr-2 size-4" />
                            {t('pages:categories.setAsDefault')}
                        </DropdownMenuItem>
                    )}
                    {needsReassign && (
                        <>
                            {(canEdit || canSetDefault) && <DropdownMenuSeparator />}
                            <ReassignDeleteItem
                                category={category}
                                onConfirm={(successorId) => onDelete(category.id, successorId)}
                            />
                        </>
                    )}
                </RowActions>
            )}
        />
    )
}

interface ReassignDeleteItemProps {
    category: Category
    onConfirm: (successorId: number) => void
}

function ReassignDeleteItem({ category, onConfirm }: ReassignDeleteItemProps) {
    const { t } = useTranslation(['common', 'pages'])
    const name = localizeDefaultName(category.name)
    const count = category.transactionsCount ?? 0
    const [successorId, setSuccessorId] = useState<number | null>(null)

    return (
        <AlertDialog onOpenChange={(open) => !open && setSuccessorId(null)}>
            <AlertDialogTrigger asChild>
                <DropdownMenuItem
                    variant="destructive"
                    onSelect={(event) => event.preventDefault()}
                >
                    <Trash2 className="mr-2 size-4" />
                    {t('actions.delete')}
                </DropdownMenuItem>
            </AlertDialogTrigger>
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{t('pages:categories.reassignTitle')}</AlertDialogTitle>
                    <AlertDialogDescription>
                        {t('pages:categories.reassignDescription', { name, count })}
                    </AlertDialogDescription>
                </AlertDialogHeader>
                <CategorySelect
                    value={successorId}
                    onChange={setSuccessorId}
                    type={category.type}
                    excludeIds={[category.id]}
                    placeholder={t('pages:categories.reassignPlaceholder')}
                    sortByPopularity
                    plain
                />
                <AlertDialogFooter>
                    <AlertDialogCancel>{t('actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                        variant="destructive"
                        disabled={!successorId}
                        onClick={() => successorId && onConfirm(successorId)}
                    >
                        {t('pages:categories.reassignAndDelete')}
                    </AlertDialogAction>
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}
