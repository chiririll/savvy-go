import {
    columnVisibilityFeature,
    createExpandedRowModel,
    createPaginatedRowModel,
    rowExpandingFeature,
    rowPaginationFeature,
    rowSelectionFeature,
    tableFeatures,
} from '@tanstack/react-table'
import type { ColumnDef, ReactTable, Row, RowData } from '@tanstack/react-table'

// Every table in the app shares this feature set, so the row, column and table types stay
// one-parameter outside this file.
export const appTableFeatures = tableFeatures({
    rowExpandingFeature,
    rowPaginationFeature,
    rowSelectionFeature,
    columnVisibilityFeature,
    expandedRowModel: createExpandedRowModel(),
    paginatedRowModel: createPaginatedRowModel(),
})

export type AppTableFeatures = typeof appTableFeatures
export type AppColumnDef<T extends RowData> = ColumnDef<AppTableFeatures, T>
export type AppRow<T extends RowData> = Row<AppTableFeatures, T>
export type AppTable<T extends RowData> = ReactTable<AppTableFeatures, T>
