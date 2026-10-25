import { type CSSProperties, type ReactNode } from 'react'
import { GripVertical } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import {
    DndContext,
    KeyboardSensor,
    PointerSensor,
    closestCenter,
    useSensor,
    useSensors,
    type DragEndEvent,
} from '@dnd-kit/core'
import {
    SortableContext,
    arrayMove,
    sortableKeyboardCoordinates,
    useSortable,
    verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { restrictToParentElement, restrictToVerticalAxis } from '@dnd-kit/modifiers'
import { CSS } from '@dnd-kit/utilities'
import { FeedEmpty, FeedRowSkeleton } from '@/components/shared'
import { cn } from '@/lib/utils'
import { Account } from '@/types'
import { AccountRow } from './AccountRow'

interface AccountListProps {
    accounts: Account[]
    isLoading?: boolean
    emptyTitle: string
    emptyDescription: string
    emptyAction?: ReactNode
    onCreate?: () => void
    createLabel?: string
    onEdit?: (account: Account) => void
    onDelete?: (id: number) => void
    onReorder?: (accounts: Account[]) => void
    isReadOnly?: boolean
}

function SortableAccountRow({
    account,
    onEdit,
    onDelete,
    isReadOnly,
}: {
    account: Account
    onEdit?: (account: Account) => void
    onDelete?: (id: number) => void
    isReadOnly?: boolean
}) {
    const { t } = useTranslation()
    const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
        id: String(account.id),
    })
    const style: CSSProperties = {
        transform: CSS.Transform.toString(transform),
        transition,
    }

    return (
        <div ref={setNodeRef} style={style} className={cn(isDragging && 'relative z-10 opacity-50')}>
            <AccountRow
                account={account}
                onEdit={onEdit}
                onDelete={onDelete}
                isReadOnly={isReadOnly}
                dragHandle={
                    <div
                        className="shrink-0"
                        onClick={(event) => event.stopPropagation()}
                        onPointerDown={(event) => event.stopPropagation()}
                    >
                        <button
                            type="button"
                            className={cn(
                                'flex size-8 cursor-grab touch-none items-center justify-center rounded-md',
                                'text-muted-foreground hover:bg-accent hover:text-foreground',
                                'active:cursor-grabbing',
                            )}
                            aria-label={t('actions.reorder')}
                            {...attributes}
                            {...listeners}
                        >
                            <GripVertical className="size-4" />
                        </button>
                    </div>
                }
            />
        </div>
    )
}

export function AccountList({
    accounts,
    isLoading,
    emptyTitle,
    emptyDescription,
    emptyAction,
    onCreate,
    createLabel,
    onEdit,
    onDelete,
    onReorder,
    isReadOnly,
}: AccountListProps) {
    const canReorder = !!onReorder && !isReadOnly
    const sensors = useSensors(
        useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
        useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
    )

    const handleDragEnd = (event: DragEndEvent) => {
        const { active, over } = event
        if (!over || active.id === over.id || !onReorder) {
            return
        }

        const oldIndex = accounts.findIndex((account) => String(account.id) === String(active.id))
        const newIndex = accounts.findIndex((account) => String(account.id) === String(over.id))
        if (oldIndex < 0 || newIndex < 0) {
            return
        }

        onReorder(arrayMove(accounts, oldIndex, newIndex))
    }

    if (isLoading) {
        return <FeedRowSkeleton />
    }

    if (accounts.length === 0) {
        return (
            <FeedEmpty
                title={emptyTitle}
                description={emptyDescription}
                action={emptyAction}
                onCreate={onCreate}
                createLabel={createLabel}
                isReadOnly={isReadOnly}
            />
        )
    }

    const rows = (
        <div className="divide-y divide-border/60">
            {accounts.map((account) =>
                canReorder ? (
                    <SortableAccountRow
                        key={account.id}
                        account={account}
                        onEdit={onEdit}
                        onDelete={onDelete}
                        isReadOnly={isReadOnly}
                    />
                ) : (
                    <AccountRow
                        key={account.id}
                        account={account}
                        onEdit={onEdit}
                        onDelete={onDelete}
                        isReadOnly={isReadOnly}
                    />
                ),
            )}
        </div>
    )

    if (!canReorder) {
        return rows
    }

    return (
        <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            modifiers={[restrictToVerticalAxis, restrictToParentElement]}
            onDragEnd={handleDragEnd}
        >
            <SortableContext
                items={accounts.map((account) => String(account.id))}
                strategy={verticalListSortingStrategy}
            >
                {rows}
            </SortableContext>
        </DndContext>
    )
}
