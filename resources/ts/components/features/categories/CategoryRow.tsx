import { testId } from '@/lib/test-id'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Star, Trash2 } from 'lucide-react'
import { FeedRow, RowActions, CategorySelect, ResponsiveDialog } from '@/components/shared'
import { DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { localizeDefaultName } from '@/lib/localized-name'
import { Category } from '@/types'
import { categoryTypeLabel } from '@/lib/labels'

interface CategoryRowProps {
    category: Category
    onEdit?: (category: Category) => void
    onDelete?: (id: number, successorId?: number) => void
    onSetDefault?: (id: number) => void
    isSettingDefault?: boolean
    isReadOnly?: boolean
    deleteDisabled?: boolean
    deleteDisabledLabel?: string
    /**
     * The categories are kept identical with other spaces: deleting also
     * deletes them there, so the transactions of any of those spaces may need
     * a replacement even when this space has none.
     */
    synced?: boolean
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
    synced,
}: CategoryRowProps) {
    const { t } = useTranslation(['common', 'pages'])
    const name = localizeDefaultName(category.name)
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const canSetDefault = !!onSetDefault && !isReadOnly && !category.isDefault
    const needsReassign = canDelete && !deleteDisabled && ((category.transactionsCount ?? 0) > 0 || !!synced)

    return (
        <FeedRow
            icon={category.icon}
            iconColor={category.color}
            title={name}
            badge={category.isDefault ? (
                <Star
                    className="size-3.5 shrink-0 fill-warning text-warning"
                    aria-label={t('pages:categories.columns.default')}
                />
            ) : undefined}
            subtitle={categoryTypeLabel(t, category.type)}
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
                                optional={!!synced && (category.transactionsCount ?? 0) === 0}
                                synced={!!synced}
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
    /** A replacement may be left out: this space has no transactions in the category. */
    optional?: boolean
    synced?: boolean
    onConfirm: (successorId?: number) => void
}

function ReassignDeleteItem({ category, optional, synced, onConfirm }: ReassignDeleteItemProps) {
    const { t } = useTranslation(['common', 'pages'])
    const name = localizeDefaultName(category.name)
    const count = category.transactionsCount ?? 0
    const [open, setOpen] = useState(false)
    const [successorId, setSuccessorId] = useState<number | null>(null)
    const close = () => {
        setOpen(false)
        setSuccessorId(null)
    }
    const description = [
        synced && count === 0
            ? t('pages:categories.syncedDeleteDescription', { name })
            : t('pages:categories.reassignDescription', { name, count }),
        synced && count > 0 && t('pages:categories.syncedDeleteNote'),
    ]
        .filter(Boolean)
        .join(' ')

    return (
        <>
            <DropdownMenuItem
                variant="destructive"
                {...testId('delete-reassign')}
                onSelect={(event) => {
                    event.preventDefault()
                    setOpen(true)
                }}
            >
                <Trash2 className="mr-2 size-4" />
                {t('actions.delete')}
            </DropdownMenuItem>
            <ResponsiveDialog
                open={open}
                onOpenChange={(next) => (next ? setOpen(true) : close())}
                title={t('pages:categories.reassignTitle')}
                description={description}
                footer={
                    <>
                        <Button variant="outline" onClick={close}>
                            {t('actions.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            disabled={!optional && !successorId}
                            onClick={() => {
                                onConfirm(successorId ?? undefined)
                                close()
                            }}
                        >
                            {successorId || !optional ? t('pages:categories.reassignAndDelete') : t('actions.delete')}
                        </Button>
                    </>
                }
            >
                <CategorySelect
                    value={successorId}
                    onChange={setSuccessorId}
                    type={category.type}
                    excludeIds={[category.id]}
                    placeholder={t('pages:categories.reassignPlaceholder')}
                    sortByPopularity
                    plain
                />
            </ResponsiveDialog>
        </>
    )
}
