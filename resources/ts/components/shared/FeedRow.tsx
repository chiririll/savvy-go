import { type CSSProperties, type ReactNode, useState } from 'react'
import { FileX, Plus } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
    Empty,
    EmptyDescription,
    EmptyHeader,
    EmptyMedia,
    EmptyTitle,
} from '@/components/ui/empty'
import { useIsMobile, useLongPress } from '@/hooks'
import { cn } from '@/lib/utils'

interface FeedRowActionsContext {
    menuOpen: boolean
    setMenuOpen: (open: boolean) => void
    isMobile: boolean
}

interface FeedRowProps {
    icon: ReactNode
    iconClassName?: string
    iconStyle?: CSSProperties
    title: ReactNode
    titleClassName?: string
    badge?: ReactNode
    meta?: ReactNode
    subtitle?: ReactNode
    amount?: ReactNode
    amountClassName?: string
    extraAmount?: ReactNode
    leading?: ReactNode
    below?: ReactNode
    /** Greyed out: an inactive item. */
    muted?: boolean
    onOpen?: () => void
    hasActions?: boolean
    actions?: (context: FeedRowActionsContext) => ReactNode
}

export function FeedStatusBadge({
    children,
    variant,
}: {
    children: ReactNode
    variant?: 'secondary' | 'outline'
}) {
    return (
        <Badge variant={variant} className="h-5 shrink-0 px-1.5 text-[10px]">
            {children}
        </Badge>
    )
}

export function FeedRow({
    icon,
    iconClassName,
    iconStyle,
    title,
    titleClassName,
    badge,
    meta,
    subtitle,
    amount,
    amountClassName,
    extraAmount,
    leading,
    below,
    muted,
    onOpen,
    hasActions = false,
    actions,
}: FeedRowProps) {
    const isMobile = useIsMobile()
    const [menuOpen, setMenuOpen] = useState(false)
    const interactive = !!onOpen
    const longPress = useLongPress({ enabled: hasActions && isMobile })

    return (
        <div className="min-w-0">
            <div
                className={cn(
                    'relative flex min-w-0 items-center gap-2.5 rounded-lg py-2 sm:gap-3 sm:px-1.5',
                    interactive && 'cursor-pointer hover:bg-muted/60',
                    isMobile && hasActions && 'select-none [-webkit-touch-callout:none]',
                )}
                {...(isMobile && hasActions ? longPress.handlers : {})}
                onClick={() => {
                    if (longPress.consume()) {
                        setMenuOpen(true)
                        return
                    }
                    onOpen?.()
                }}
                onContextMenu={(event) => {
                    if (!isMobile || !hasActions) {
                        return
                    }
                    event.preventDefault()
                    longPress.mark()
                    setMenuOpen(true)
                }}
                onKeyDown={interactive ? (event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault()
                        onOpen?.()
                    }
                } : undefined}
                role={interactive ? 'button' : undefined}
                tabIndex={interactive ? 0 : undefined}
            >
                {leading}

                <div
                    className={cn(
                        'flex size-10 shrink-0 items-center justify-center rounded-full text-base',
                        muted && 'opacity-50 grayscale',
                        iconClassName,
                    )}
                    style={iconStyle}
                >
                    {icon}
                </div>

                <div className={cn('min-w-0 flex-1', muted && 'opacity-60')}>
                    <div className="flex min-w-0 items-center gap-1.5">
                        <p className={cn('truncate font-semibold', titleClassName)}>{title}</p>
                        {badge}
                    </div>
                    {meta && (
                        <div className="mt-0.5 flex min-w-0 flex-wrap items-center gap-1">
                            {meta}
                        </div>
                    )}
                    {subtitle && (
                        <div className="truncate text-xs text-muted-foreground">
                            {subtitle}
                        </div>
                    )}
                </div>

                {(amount != null || extraAmount) && (
                    <div className={cn('min-w-0 shrink-0 text-right', muted && 'opacity-60')}>
                        {amount != null && (
                            <p className={cn('font-mono font-semibold', muted ? 'text-muted-foreground' : amountClassName)}>
                                {amount}
                            </p>
                        )}
                        {extraAmount && (
                            <p className="font-mono text-xs text-muted-foreground">
                                {extraAmount}
                            </p>
                        )}
                    </div>
                )}

                {hasActions && actions && (
                    <div
                        className={cn(!isMobile && '-mr-1 shrink-0')}
                        onClick={(event) => event.stopPropagation()}
                    >
                        {actions({ menuOpen, setMenuOpen, isMobile })}
                    </div>
                )}
            </div>
            {below}
        </div>
    )
}

/** A thin progress bar under a row; exceeded turns it red. */
export function FeedProgress({
    value,
    exceeded,
    muted,
}: {
    value: number
    exceeded?: boolean
    muted?: boolean
}) {
    return (
        <div className="-mt-1 mb-1.5 h-1 overflow-hidden rounded-full bg-muted sm:mx-1.5" role="presentation">
            <div
                className={cn(
                    'h-full rounded-full transition-[width]',
                    exceeded ? 'bg-red-500' : 'bg-green-500',
                    muted && 'bg-muted-foreground/40',
                )}
                style={{ width: `${Math.max(0, Math.min(value, 100))}%` }}
            />
        </div>
    )
}

/** A titled section of feed rows. */
export function FeedGroup({ title, children }: { title: ReactNode; children: ReactNode }) {
    return (
        <section className="min-w-0">
            <h2 className="px-1.5 pb-1 text-sm font-semibold capitalize">{title}</h2>
            <div className="divide-y divide-border/60">{children}</div>
        </section>
    )
}

export interface FeedGroupKey {
    key: string
    title: ReactNode
}

/** Splits items into groups in order of first appearance. */
export function groupFeedItems<T>(items: T[], groupBy: (item: T) => FeedGroupKey) {
    const groups = new Map<string, FeedGroupKey & { items: T[] }>()
    for (const item of items) {
        const group = groupBy(item)
        const existing = groups.get(group.key)
        if (existing) {
            existing.items.push(item)
        } else {
            groups.set(group.key, { ...group, items: [item] })
        }
    }
    return [...groups.values()]
}

export function FeedRowSkeleton({ showHeading = false }: { showHeading?: boolean }) {
    return (
        <div className="space-y-1">
            {showHeading && <Skeleton className="mb-2 h-4 w-28" />}
            {Array.from({ length: showHeading ? 3 : 4 }).map((_, row) => (
                <div key={row} className="flex items-center gap-3 px-1.5 py-2">
                    <Skeleton className="size-10 rounded-full" />
                    <div className="min-w-0 flex-1 space-y-1.5">
                        <Skeleton className="h-4 w-36" />
                        <Skeleton className="h-3 w-24" />
                    </div>
                    <Skeleton className="h-4 w-16" />
                </div>
            ))}
        </div>
    )
}

export function FeedList<T>({
    items,
    isLoading,
    emptyTitle,
    emptyDescription,
    emptyAction,
    onCreate,
    createLabel,
    isReadOnly,
    getKey,
    groupBy,
    children,
}: {
    items: T[]
    isLoading?: boolean
    emptyTitle: string
    emptyDescription: string
    emptyAction?: ReactNode
    onCreate?: () => void
    createLabel?: string
    isReadOnly?: boolean
    getKey: (item: T) => string | number
    /** Shows items in titled sections, in order of first appearance. */
    groupBy?: (item: T) => FeedGroupKey
    children: (item: T) => ReactNode
}) {
    if (isLoading) {
        return <FeedRowSkeleton showHeading={!!groupBy} />
    }

    if (items.length === 0) {
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

    if (groupBy) {
        return (
            <div className="space-y-5">
                {groupFeedItems(items, groupBy).map((group) => (
                    <FeedGroup key={group.key} title={group.title}>
                        {group.items.map((item) => (
                            <div key={getKey(item)}>{children(item)}</div>
                        ))}
                    </FeedGroup>
                ))}
            </div>
        )
    }

    return (
        <div className="divide-y divide-border/60">
            {items.map((item) => (
                <div key={getKey(item)}>
                    {children(item)}
                </div>
            ))}
        </div>
    )
}

export function FeedEmpty({
    title,
    description,
    action,
    onCreate,
    createLabel,
    isReadOnly,
}: {
    title: string
    description: string
    action?: ReactNode
    onCreate?: () => void
    createLabel?: string
    isReadOnly?: boolean
}) {
    return (
        <Empty className="border rounded-lg py-16">
            <EmptyHeader>
                <EmptyMedia variant="icon">
                    <FileX />
                </EmptyMedia>
                <EmptyTitle>{title}</EmptyTitle>
                <EmptyDescription>{description}</EmptyDescription>
            </EmptyHeader>
            {action ?? (
                onCreate && !isReadOnly ? (
                    <Button onClick={onCreate}>
                        <Plus className="size-4" />
                        {createLabel}
                    </Button>
                ) : undefined
            )}
        </Empty>
    )
}
