import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'

interface OnboardingLayoutProps {
    title: string
    description: string
    children: ReactNode
    /** Width of the card, for screens with more on them. */
    className?: string
}

/** The card every onboarding screen sits in. */
export function OnboardingLayout({ title, description, children, className }: OnboardingLayoutProps) {
    return (
        <div className="min-h-screen flex items-start sm:items-center justify-center bg-background p-3 sm:p-4">
            <Card className={cn('w-full max-w-md', className)}>
                <CardHeader className="text-center px-4 sm:px-6">
                    <div className="flex justify-center mb-4">
                        <div className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                            <Logo className="size-7" />
                        </div>
                    </div>
                    <CardTitle className="text-2xl">{title}</CardTitle>
                    <CardDescription>{description}</CardDescription>
                </CardHeader>
                <CardContent className="px-4 sm:px-6">
                    {children}
                    <LanguageSwitcher variant="auth" className="pt-4" />
                </CardContent>
            </Card>
        </div>
    )
}
