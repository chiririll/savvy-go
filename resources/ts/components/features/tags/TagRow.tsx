import { Hash } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { FeedRow, RowActions } from '@/components/shared'
import { Tag } from '@/types'
import { localizeDefaultName } from '@/lib/localized-name'

interface TagRowProps {
    tag: Tag
    onEdit?: (tag: Tag) => void
    onDelete?: (id: number) => void
    isReadOnly?: boolean
}

export function TagRow({ tag, onEdit, onDelete, isReadOnly }: TagRowProps) {
    const { t } = useTranslation(['common', 'pages'])
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const count = tag.transactionsCount ?? 0

    return (
        <FeedRow
            icon={<Hash className="size-4" />}
            iconClassName="bg-muted text-muted-foreground"
            title={`#${localizeDefaultName(tag.name, 'tags')}`}
            amount={count}
            amountClassName="text-muted-foreground"
            onOpen={canEdit ? () => onEdit(tag) : undefined}
            hasActions={canEdit || canDelete}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(tag) : undefined}
                    onDelete={canDelete ? () => onDelete(tag.id) : undefined}
                    deleteTitle={t('pages:tags.deleteTitle')}
                    deleteDescription={t('pages:tags.deleteDescription', { name: localizeDefaultName(tag.name, 'tags') })}
                />
            )}
        />
    )
}
