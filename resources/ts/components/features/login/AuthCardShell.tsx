import type { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/appearance/Logo'
import { cn } from '@/lib/utils'

interface Props {
    title: string
    description: string
    contentClassName?: string
    /** Width of the card, for screens with more on them. */
    className?: string
    /** Tighter padding and top alignment on a phone, for long screens. */
    compact?: boolean
    children: ReactNode
}

/** The centred card with the logo that the login and onboarding screens share. */
export function AuthCardShell({ title, description, contentClassName, className, compact, children }: Props) {
    return (
        <div
            className={cn(
                'min-h-screen flex justify-center bg-background',
                compact ? 'items-start sm:items-center p-3 sm:p-4' : 'items-center p-4',
            )}
        >
            <Card className={cn('w-full max-w-md', className)}>
                <CardHeader className={cn('text-center', compact && 'px-4 sm:px-6')}>
                    <div className="flex justify-center mb-4">
                        <div
                            className={cn(
                                'flex size-12 items-center justify-center rounded-xl bg-primary',
                                'text-primary-foreground',
                            )}
                        >
                            <Logo className="size-7" />
                        </div>
                    </div>
                    <CardTitle className="text-2xl">{title}</CardTitle>
                    <CardDescription>{description}</CardDescription>
                </CardHeader>
                <CardContent className={cn(compact && 'px-4 sm:px-6', contentClassName)}>{children}</CardContent>
            </Card>
            <LanguageSwitcher variant="auth" />
        </div>
    )
}
