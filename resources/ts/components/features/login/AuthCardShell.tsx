import type { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitcher } from '@/components/shared'
import { Logo } from '@/components/shared/Logo'
import { cn } from '@/lib/utils'

interface Props {
    title: string
    description: string
    contentClassName: string
    children: ReactNode
}

/** The centred card with the logo that the login screens share. */
export function AuthCardShell({ title, description, contentClassName, children }: Props) {
    return (
        <div className="min-h-screen flex items-center justify-center bg-background p-4">
            <Card className="w-full max-w-md">
                <CardHeader className="text-center">
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
                <CardContent className={contentClassName}>{children}</CardContent>
            </Card>
            <LanguageSwitcher variant="auth" />
        </div>
    )
}
