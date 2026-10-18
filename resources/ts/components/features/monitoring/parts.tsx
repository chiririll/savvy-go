import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

import { SEVERITY_STROKE, type Severity } from './severity'

interface StatCardProps {
    icon: LucideIcon
    label: string
    value: string
    hint?: string
    tone?: Severity
}

export function StatCard({ icon: Icon, label, value, hint, tone }: StatCardProps) {
    const toneClass = tone && tone !== 'ok' && SEVERITY_STROKE[tone]
    return (
        <div className="rounded-xl border bg-card p-4 shadow-sm">
            <div className="flex items-center gap-2 text-muted-foreground">
                <Icon className={cn('size-4', toneClass)} />
                <span className="text-xs font-medium uppercase tracking-wide">{label}</span>
            </div>
            <p className={cn('mt-3 text-2xl font-semibold tabular-nums tracking-tight', toneClass)}>{value}</p>
            {hint && <p className="mt-1 text-xs text-muted-foreground">{hint}</p>}
        </div>
    )
}

export function LegendDot({ className, label, value }: { className: string; label: string; value: string }) {
    return (
        <div className="flex items-center gap-2">
            <span className={cn('size-2.5 shrink-0 rounded-sm', className)} />
            <span className="text-muted-foreground">{label}</span>
            <span className="ml-auto font-medium tabular-nums">{value}</span>
        </div>
    )
}

export function Caption({ children }: { children: ReactNode }) {
    return <p className="text-xs uppercase tracking-wide text-muted-foreground">{children}</p>
}

export function PanelSubtitle({ children }: { children: ReactNode }) {
    return <p className="mt-1 text-xs text-muted-foreground">{children}</p>
}

interface PanelHeaderProps {
    icon: LucideIcon
    title: string
    subtitle: ReactNode
    children?: ReactNode
    align?: 'center' | 'start'
}

/** Icon tile, title and subtitle; extra content (chips, badges) goes to the right. */
export function PanelHeader({ icon: Icon, title, subtitle, children, align = 'center' }: PanelHeaderProps) {
    return (
        <header
            className={cn(
                'flex flex-wrap gap-3 border-b px-6 py-5',
                align === 'center' ? 'items-center' : 'items-start',
            )}
        >
            <div
                className={cn(
                    'flex size-10 shrink-0 items-center justify-center rounded-xl border bg-muted/50',
                    'text-muted-foreground',
                )}
            >
                <Icon className="size-5" />
            </div>
            <div className="min-w-0">
                <h2 className="text-sm font-semibold leading-none">{title}</h2>
                {subtitle}
            </div>
            {children}
        </header>
    )
}
