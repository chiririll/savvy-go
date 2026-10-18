import type { ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { InputOTP, InputOTPGroup, InputOTPSlot, InputOTPSeparator } from '@/components/ui/input-otp'
import { cn } from '@/lib/utils'

export function Section({
    icon: Icon,
    title,
    description,
    children,
}: {
    icon: LucideIcon
    title: string
    description: string
    children: ReactNode
}) {
    return (
        <section className="rounded-xl border bg-card shadow-sm">
            <header className="flex items-start gap-3 border-b px-5 py-4">
                <div
                    className={cn(
                        'flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50',
                        'text-muted-foreground',
                    )}
                >
                    <Icon className="size-4" />
                </div>
                <div className="space-y-0.5">
                    <h3 className="text-sm font-semibold leading-none">{title}</h3>
                    <p className="text-xs text-muted-foreground">{description}</p>
                </div>
            </header>
            <div className="p-5">{children}</div>
        </section>
    )
}

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

export function OtpField({ value, onChange }: { value: string; onChange: (value: string) => void }) {
    return (
        <div className="flex justify-center py-2">
            <InputOTP maxLength={6} value={value} onChange={onChange}>
                <InputOTPGroup>
                    <InputOTPSlot index={0} />
                    <InputOTPSlot index={1} />
                    <InputOTPSlot index={2} />
                </InputOTPGroup>
                <InputOTPSeparator />
                <InputOTPGroup>
                    <InputOTPSlot index={3} />
                    <InputOTPSlot index={4} />
                    <InputOTPSlot index={5} />
                </InputOTPGroup>
            </InputOTP>
        </div>
    )
}
