import { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'

/** A card on a settings page: an icon, a title, a short description. */
export function SettingsSection({
    icon: Icon,
    title,
    description,
    actions,
    children,
}: {
    icon: LucideIcon
    title: string
    description: string
    actions?: ReactNode
    children: ReactNode
}) {
    return (
        <section className="rounded-xl border bg-card shadow-sm">
            {/* On a phone the actions go under the text, so it is not squeezed into a narrow column. */}
            <header className="flex flex-col gap-3 border-b px-5 py-4 sm:flex-row sm:items-start">
                <div className="flex flex-1 items-start gap-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50 text-muted-foreground">
                        <Icon className="size-4" />
                    </div>
                    <div className="min-w-0 flex-1 space-y-0.5">
                        <h3 className="text-sm font-semibold leading-none">{title}</h3>
                        <p className="text-xs text-muted-foreground">{description}</p>
                    </div>
                </div>
                {actions && <div className="flex shrink-0 sm:self-start">{actions}</div>}
            </header>
            <div className="p-5">{children}</div>
        </section>
    )
}

interface ToggleRowProps {
    id: string
    label: string
    description: string
    checked: boolean
    disabled?: boolean
    badge?: ReactNode
    onChange: (checked: boolean) => void
}

export function ToggleRow({ id, label, description, checked, disabled, badge, onChange }: ToggleRowProps) {
    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="space-y-1">
                <div className="flex items-center gap-2">
                    <Label htmlFor={id} className="cursor-pointer text-sm font-medium">
                        {label}
                    </Label>
                    {badge}
                </div>
                <p className="text-sm leading-relaxed text-muted-foreground">{description}</p>
            </div>
            <Switch id={id} checked={checked} disabled={disabled} onCheckedChange={onChange} />
        </div>
    )
}

export function SettingsRowSkeleton() {
    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="space-y-2">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-64" />
            </div>
            <Skeleton className="h-5 w-8 rounded-full" />
        </div>
    )
}
