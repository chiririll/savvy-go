import { createElement, Fragment } from 'react'
import { SortableRows, getItemId } from './SortableRows'
import { cn } from '@/lib/utils'
import { flexRender, useTable } from '@tanstack/react-table'
import type { RowData } from '@tanstack/react-table'
import { appTableFeatures } from './table-features'
import type { AppColumnDef, AppRow } from './table-features'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useTranslation } from 'react-i18next'
import { DataTableSkeleton, DataTableEmpty, DataTablePagination } from './DataTableParts'
import { SortableTableRow } from './SortableTableRow'

interface DataTableProps<T extends RowData> {
    data: T[]
    columns: AppColumnDef<T>[]
    isLoading?: boolean
    emptyTitle?: string
    emptyDescription?: string
    emptyAction?: React.ReactNode
    renderSubComponent?: (props: { row: AppRow<T> }) => React.ReactNode
    getRowCanExpand?: (row: AppRow<T>) => boolean
    getRowClassName?: (row: AppRow<T>) => string | undefined
    manualPagination?: boolean
    onReorder?: (items: T[]) => void
}

export function DataTable<T extends RowData>({
    data,
    columns,
    isLoading,
    emptyTitle: emptyTitleProp,
    emptyDescription: emptyDescriptionProp,
    emptyAction,
    renderSubComponent,
    getRowCanExpand,
    getRowClassName,
    manualPagination = false,
    onReorder,
}: DataTableProps<T>) {
    const { t } = useTranslation()
    const emptyTitle = emptyTitleProp ?? t('table.emptyTitle')
    const emptyDescription = emptyDescriptionProp ?? t('table.emptyDescription')
    const isReorderable = !!onReorder
    const disablePagination = manualPagination || isReorderable
    const table = useTable({
        features: appTableFeatures,
        data,
        columns,
        getRowCanExpand,
        getRowId: isReorderable ? (row) => getItemId(row) : undefined,
        manualPagination: disablePagination,
    })

    if (isLoading) {
        return <DataTableSkeleton columns={columns.length + (isReorderable ? 1 : 0)} />
    }

    if (data.length === 0) {
        return <DataTableEmpty title={emptyTitle} description={emptyDescription} action={emptyAction} />
    }

    const tableContent = (
        <div className="rounded-lg border">
            <Table>
                <TableHeader>
                    {table.getHeaderGroups().map((headerGroup) => (
                        <TableRow key={headerGroup.id}>
                            {isReorderable && (
                                <TableHead className="w-10">
                                    <span className="sr-only">{t('actions.reorder')}</span>
                                </TableHead>
                            )}
                            {headerGroup.headers.map((header) => (
                                <TableHead key={header.id}>
                                    {header.isPlaceholder
                                        ? null
                                        : flexRender(header.column.columnDef.header, header.getContext())}
                                </TableHead>
                            ))}
                        </TableRow>
                    ))}
                </TableHeader>
                <TableBody>
                    {table.getRowModel().rows.map((row) =>
                        isReorderable ? (
                            <SortableTableRow
                                key={row.id}
                                row={row}
                                getRowClassName={getRowClassName}
                                renderSubComponent={renderSubComponent}
                            />
                        ) : (
                            <Fragment key={row.id}>
                                <TableRow
                                    data-state={row.getIsSelected() && 'selected'}
                                    className={cn(getRowClassName?.(row))}
                                >
                                    {row.getVisibleCells().map((cell) => (
                                        <TableCell key={cell.id}>
                                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                        </TableCell>
                                    ))}
                                </TableRow>
                                {row.getIsExpanded() && renderSubComponent && (
                                    <TableRow className="bg-muted/30 hover:bg-muted/30">
                                        <TableCell colSpan={row.getVisibleCells().length} className="p-0">
                                            {createElement(renderSubComponent, { row })}
                                        </TableCell>
                                    </TableRow>
                                )}
                            </Fragment>
                        ),
                    )}
                </TableBody>
            </Table>
            {!disablePagination && <DataTablePagination table={table} />}
        </div>
    )

    if (!isReorderable) {
        return tableContent
    }

    return (
        <SortableRows data={data} onReorder={onReorder}>
            {tableContent}
        </SortableRows>
    )
}
