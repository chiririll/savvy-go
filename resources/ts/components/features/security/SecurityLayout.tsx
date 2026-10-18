import type { ReactNode } from 'react'
import { Skeleton } from '@/components/ui/skeleton'

export function ActionRow({
    label,
    description,
    badge,
    children,
}: {
    label: string
    description: ReactNode
    badge?: ReactNode
    children: ReactNode
}) {
    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="min-w-0 space-y-1">
                <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{label}</span>
                    {badge}
                </div>
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
            </div>
            {children}
        </div>
    )
}

export function RowSkeleton() {
    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-64" />
            </div>
            <Skeleton className="h-9 w-24" />
        </div>
    )
}
