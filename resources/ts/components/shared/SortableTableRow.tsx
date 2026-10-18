import { createElement, CSSProperties, Fragment } from 'react'
import { cn } from '@/lib/utils'
import { flexRender, Row } from '@tanstack/react-table'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { TableCell, TableRow } from '@/components/ui/table'
import { GripVertical } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function SortableTableRow<T>({
    row,
    getRowClassName,
    renderSubComponent,
}: {
    row: Row<T>
    getRowClassName?: (row: Row<T>) => string | undefined
    renderSubComponent?: (props: { row: Row<T> }) => React.ReactNode
}) {
    const { t } = useTranslation()
    const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
        id: row.id,
    })

    const style: CSSProperties = {
        transform: CSS.Transform.toString(transform),
        transition,
    }

    return (
        <Fragment>
            <TableRow
                ref={setNodeRef}
                style={style}
                data-state={row.getIsSelected() && 'selected'}
                className={cn(isDragging && 'relative z-10 opacity-50', getRowClassName?.(row))}
            >
                <TableCell className="w-10 pr-0">
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
                </TableCell>
                {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>{flexRender(cell.column.columnDef.cell, cell.getContext())}</TableCell>
                ))}
            </TableRow>
            {row.getIsExpanded() && renderSubComponent && (
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                    <TableCell colSpan={row.getVisibleCells().length + 1} className="p-0">
                        {createElement(renderSubComponent, { row })}
                    </TableCell>
                </TableRow>
            )}
        </Fragment>
    )
}
