import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronRight, Loader2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { InfiniteScrollSentinel } from '@/components/shared'
import { Skeleton } from '@/components/ui/skeleton'
import { formatDateTime } from '@/lib/format'
import type { SpaceAction, SpaceMember } from '@/types/spaces'
import { actionSourceLabel, actionTitle, groupActions } from './actions'

interface ActionsListProps {
    actions: SpaceAction[]
    /** Names the people behind the entries; shown only when the list holds more than one person's. */
    members?: SpaceMember[]
    showWho: boolean
    /** Filters are on, so an empty list means no match rather than an empty log. */
    filtered?: boolean
    isLoading?: boolean
    hasMore?: boolean
    isLoadingMore?: boolean
    onLoadMore?: () => void
}

/** The change log of a space, newest first. */
export function ActionsList({ actions, members = [], showWho, filtered, isLoading, hasMore, isLoadingMore, onLoadMore }: ActionsListProps) {
    const { t } = useTranslation('settings')
    const [expanded, setExpanded] = useState<Set<string>>(new Set())
    const toggle = (key: string) =>
        setExpanded((prev) => {
            const next = new Set(prev)
            if (!next.delete(key)) next.add(key)
            return next
        })

    if (isLoading) {
        return (
            <div className="space-y-2">
                {[0, 1, 2].map((i) => (
                    <Skeleton key={i} className="h-14 w-full" />
                ))}
            </div>
        )
    }
    if (actions.length === 0) {
        return <p className="text-sm text-muted-foreground">{filtered ? t('spaces.actions.filters.emptyFiltered') : t('spaces.actions.empty')}</p>
    }

    const nameOf = (a: SpaceAction): string => {
        const member = members.find((m) => m.userId === a.userId)
        if (member) return member.name
        if (a.userId === null) return t('spaces.actions.system')
        return t('spaces.actions.formerMember')
    }

    const row = (a: SpaceAction, opts: { count?: number } = {}) => (
        <div className="flex flex-1 flex-col gap-1 sm:flex-row sm:items-center sm:gap-3">
            <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                    {actionTitle(t, a)}
                    {opts.count === undefined && a.entityId !== null && <span className="text-muted-foreground"> #{a.entityId}</span>}
                    {opts.count !== undefined && <span className="text-muted-foreground"> ×{opts.count}</span>}
                </p>
                <p className="truncate text-xs text-muted-foreground">
                    {[showWho ? nameOf(a) : null, a.tokenLabel, opts.count === undefined ? a.details : null].filter(Boolean).join(' · ')}
                </p>
            </div>
            <div className="flex items-center gap-2 sm:ml-auto">
                <Badge variant="secondary">{actionSourceLabel(t, a.source)}</Badge>
                <time dateTime={a.at} className="text-xs text-muted-foreground">
                    {formatDateTime(a.at)}
                </time>
            </div>
        </div>
    )

    return (
        <div className="space-y-3">
            <ul className="divide-y rounded-lg border">
                {groupActions(actions).map((g) =>
                    g.items.length === 1 ? (
                        <li key={g.key} className="flex px-3 py-2">{row(g.first)}</li>
                    ) : (
                        <li key={g.key}>
                            <button
                                type="button"
                                aria-expanded={expanded.has(g.key)}
                                onClick={() => toggle(g.key)}
                                className="flex w-full items-center gap-1 px-3 py-2 text-left hover:bg-muted/50"
                            >
                                <ChevronRight className={`size-4 shrink-0 transition-transform ${expanded.has(g.key) ? 'rotate-90' : ''}`} />
                                {row(g.first, { count: g.items.length })}
                            </button>
                            {expanded.has(g.key) && (
                                <ul className="divide-y border-t bg-muted/30 pl-5">{g.items.map((a) => (
                                        <li key={a.id} className="flex px-3 py-2">{row(a)}</li>
                                    ))}</ul>
                            )}
                        </li>
                    ),
                )}
            </ul>
            <InfiniteScrollSentinel enabled={!!hasMore && !isLoadingMore} onVisible={() => onLoadMore?.()} />
            {isLoadingMore && <Loader2 className="mx-auto size-4 animate-spin text-muted-foreground" />}
        </div>
    )
}
