import type { ReactNode } from 'react'
import {
    DndContext,
    KeyboardSensor,
    PointerSensor,
    closestCenter,
    useSensor,
    useSensors,
    type DragEndEvent,
} from '@dnd-kit/core'
import { SortableContext, arrayMove, sortableKeyboardCoordinates, verticalListSortingStrategy } from '@dnd-kit/sortable'
import { restrictToParentElement, restrictToVerticalAxis } from '@dnd-kit/modifiers'

export function getItemId<T>(item: T): string {
    if (item && typeof item === 'object' && 'id' in item) {
        return String((item as { id: string | number }).id)
    }

    throw new Error('Reorderable rows must have an id')
}

interface SortableRowsProps<T> {
    data: T[]
    onReorder: (items: T[]) => void
    children: ReactNode
}

/** Lets the rows inside be dragged up and down; every item needs an id. */
export function SortableRows<T>({ data, onReorder, children }: SortableRowsProps<T>) {
    const sensors = useSensors(
        useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
        useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
    )

    const handleDragEnd = ({ active, over }: DragEndEvent) => {
        if (!over || active.id === over.id) {
            return
        }

        const oldIndex = data.findIndex((item) => getItemId(item) === String(active.id))
        const newIndex = data.findIndex((item) => getItemId(item) === String(over.id))
        if (oldIndex < 0 || newIndex < 0) {
            return
        }

        onReorder(arrayMove(data, oldIndex, newIndex))
    }

    return (
        <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            modifiers={[restrictToVerticalAxis, restrictToParentElement]}
            onDragEnd={handleDragEnd}
        >
            <SortableContext items={data.map(getItemId)} strategy={verticalListSortingStrategy}>
                {children}
            </SortableContext>
        </DndContext>
    )
}
