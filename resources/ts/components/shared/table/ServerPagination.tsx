import { useTranslation } from 'react-i18next'
import {
    Pagination,
    PaginationContent,
    PaginationEllipsis,
    PaginationItem,
    PaginationLink,
    PaginationNext,
    PaginationPrevious,
} from '@/components/ui/pagination'

interface PaginationMeta {
    current_page: number
    last_page: number
    from?: number
    to?: number
    total?: number
}

interface ServerPaginationProps {
    meta: PaginationMeta
    onPageChange: (page: number) => void
    showInfo?: boolean
    infoLabel?: string
    delta?: number
}

function generatePaginationItems(
    currentPage: number,
    totalPages: number,
    delta: number = 1
): (number | 'ellipsis')[] {
    if (totalPages <= 1) return [1]

    const items: (number | 'ellipsis')[] = []

    // Always show first page
    items.push(1)

    // Calculate range around current page
    const rangeStart = Math.max(2, currentPage - delta)
    const rangeEnd = Math.min(totalPages - 1, currentPage + delta)

    // Add ellipsis after first page if needed
    if (rangeStart > 2) {
        items.push('ellipsis')
    }

    // Add pages in range
    for (let i = rangeStart; i <= rangeEnd; i++) {
        items.push(i)
    }

    // Add ellipsis before last page if needed
    if (rangeEnd < totalPages - 1) {
        items.push('ellipsis')
    }

    // Always show last page
    if (totalPages > 1) {
        items.push(totalPages)
    }

    return items
}

/** The links have no href, so they are made buttons the keyboard can reach and Enter or Space press. */
function pressable(onPress: () => void, disabled = false) {
    return {
        role: 'button',
        tabIndex: disabled ? -1 : 0,
        'aria-disabled': disabled || undefined,
        onClick: () => !disabled && onPress(),
        onKeyDown: (event: React.KeyboardEvent) => {
            if (disabled || (event.key !== 'Enter' && event.key !== ' ')) return
            event.preventDefault()
            onPress()
        },
    } as const
}

export function ServerPagination({
    meta,
    onPageChange,
    showInfo = true,
    infoLabel,
    delta = 1,
}: ServerPaginationProps) {
    const { t } = useTranslation()
    const label = infoLabel ?? t('table.items')
    const { current_page, last_page, from, to, total } = meta

    if (last_page <= 1) return null

    const canGoPrev = current_page > 1
    const canGoNext = current_page < last_page

    return (
        <div className="flex items-center justify-between mt-4">
            {showInfo && from !== undefined && to !== undefined && total !== undefined ? (
                <div className="text-sm text-muted-foreground">
                    {t('table.showing', { from, to, total, label })}
                </div>
            ) : (
                <div />
            )}

            <Pagination className="mx-0 w-auto justify-end">
                <PaginationContent>
                    <PaginationItem>
                        <PaginationPrevious
                            {...pressable(() => onPageChange(current_page - 1), !canGoPrev)}
                            className={!canGoPrev ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                        />
                    </PaginationItem>

                    {generatePaginationItems(current_page, last_page, delta).map((item, idx) => (
                        <PaginationItem key={idx}>
                            {item === 'ellipsis' ? (
                                <PaginationEllipsis />
                            ) : (
                                <PaginationLink
                                    isActive={item === current_page}
                                    {...pressable(() => onPageChange(item))}
                                    className="cursor-pointer"
                                >
                                    {item}
                                </PaginationLink>
                            )}
                        </PaginationItem>
                    ))}

                    <PaginationItem>
                        <PaginationNext
                            {...pressable(() => onPageChange(current_page + 1), !canGoNext)}
                            className={!canGoNext ? 'pointer-events-none opacity-50' : 'cursor-pointer'}
                        />
                    </PaginationItem>
                </PaginationContent>
            </Pagination>
        </div>
    )
}
