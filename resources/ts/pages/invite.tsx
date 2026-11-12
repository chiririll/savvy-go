import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitcher } from '@/components/shared'
import { InviteAlreadyMember } from '@/components/features/invite/InviteAlreadyMember'
import { InviteDetails } from '@/components/features/invite/InviteDetails'
import { useInviteActions } from '@/components/features/invite/use-invite-actions'
import { invitationsApi } from '@/api/spaces'
import { useAuthStore, useUser } from '@/stores/auth'

/** Opened from an invitation link: join the space, or create an account for it. */
export default function InvitePage() {
    const { t } = useTranslation('settings')
    const { token = '' } = useParams<{ token: string }>()
    const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
    const authLoading = useAuthStore((state) => state.isLoading)
    const checkAuth = useAuthStore((state) => state.checkAuth)
    const user = useUser()
    const actions = useInviteActions(token)

    useEffect(() => {
        void checkAuth()
    }, [checkAuth])

    const {
        data: invitation,
        isPending,
        isError,
    } = useQuery({
        queryKey: ['invitation', token],
        queryFn: () => invitationsApi.preview(token),
        retry: false,
    })

    const renderBody = () => {
        if (isPending || authLoading) {
            return (
                <CardContent className="flex justify-center py-10">
                    <Loader2 className="size-6 animate-spin text-muted-foreground" />
                </CardContent>
            )
        }
        if (actions.alreadyMember) return <InviteAlreadyMember spaceName={invitation?.spaceName ?? ''} />
        if (isError || !invitation) {
            return (
                <CardHeader>
                    <CardTitle>{t('invite.invalidTitle')}</CardTitle>
                    <CardDescription>{t('invite.invalidText')}</CardDescription>
                </CardHeader>
            )
        }
        return (
            <InviteDetails
                token={token}
                invitation={invitation}
                user={user}
                isAuthenticated={isAuthenticated}
                busy={actions.busy}
                error={actions.error}
                onJoin={() => void actions.join()}
                onRegister={(input) => void actions.register(input)}
            />
        )
    }

    return (
        <div className="flex min-h-screen items-center justify-center p-4">
            <LanguageSwitcher variant="auth" />
            <Card className="w-full max-w-md">{renderBody()}</Card>
        </div>
    )
}
