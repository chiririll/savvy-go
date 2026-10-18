import type { ReactNode } from 'react'
import { AuthCardShell } from '@/components/features/login/AuthCardShell'

interface OnboardingLayoutProps {
    title: string
    description: string
    children: ReactNode
    /** Width of the card, for screens with more on them. */
    className?: string
}

/** The card every onboarding screen sits in. */
export function OnboardingLayout(props: OnboardingLayoutProps) {
    return <AuthCardShell compact {...props} />
}
