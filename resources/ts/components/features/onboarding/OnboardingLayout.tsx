import type { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'

interface OnboardingLayoutProps {
    title: string
    description: string
    children: ReactNode
}

/** The card every onboarding screen sits in. */
export function OnboardingLayout({ title, description, children }: OnboardingLayoutProps) {
    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
                    <div className="flex justify-center mb-4">
                        <div className="flex size-12 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                            <Logo className="size-7" />
                        </div>
                    </div>
                    <CardTitle className="text-2xl">{title}</CardTitle>
                    <CardDescription>{description}</CardDescription>
                </CardHeader>
                <CardContent>
                    {children}
                    <LanguageSwitcher variant="auth" className="pt-4" />
                </CardContent>
            </Card>
        </div>
    )
}
