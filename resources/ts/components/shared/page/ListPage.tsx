import type { ReactNode } from 'react'
import type { RowData } from '@tanstack/react-table'
import type { AppColumnDef, AppRow } from '../table/table-features'
import { Link } from 'react-router-dom'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Page } from './Page'
import { PageHeader } from './PageHeader'
import { DataTable } from '../table/DataTable'
import { Button } from '@/components/ui/button'

interface ListPageProps<T extends RowData> {
    title: string
    description?: string
    createLink?: string
    createLabel?: string
    onCreateClick?: () => void
    data: T[]
    columns: AppColumnDef<T>[]
    isLoading?: boolean
    emptyTitle?: string
    emptyDescription?: string
    toolbar?: ReactNode
    getRowClassName?: (row: AppRow<T>) => string | undefined
    onReorder?: (items: T[]) => void
}

export function ListPage<T extends RowData>({
    title,
    description,
    createLink,
    createLabel,
    onCreateClick,
    data,
    columns,
    isLoading,
    emptyTitle,
    emptyDescription,
    toolbar,
    getRowClassName,
    onReorder,
}: ListPageProps<T>) {
    const { t } = useTranslation()
    const newLabel = createLabel ?? t('actions.create')
    const emptyAction = onCreateClick ? (
        <Button onClick={onCreateClick}>
            <Plus className="size-4" />
            {newLabel}
        </Button>
    ) : createLink ? (
        <Button asChild>
            <Link to={createLink}>
                <Plus className="size-4" />
                {newLabel}
            </Link>
        </Button>
    ) : undefined

    return (
        <Page title={title}>
            <PageHeader
                title={title}
                description={description}
                createLink={createLink}
                createLabel={newLabel}
                onCreateClick={onCreateClick}
            />
            {toolbar && <div className="mb-6">{toolbar}</div>}
            <DataTable
                data={data}
                columns={columns}
                isLoading={isLoading}
                emptyTitle={emptyTitle ?? t('table.emptyTitle')}
                emptyDescription={emptyDescription ?? t('table.emptyDescription')}
                getRowClassName={getRowClassName}
                emptyAction={emptyAction}
                onReorder={onReorder}
            />
        </Page>
    )
}
