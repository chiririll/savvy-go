import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { spaceRoleLabel } from '@/lib/i18n/labels'
import type { InvitationPreview } from '@/types'
import { InviteRegisterForm } from './InviteRegisterForm'
import type { RegisterInput } from './use-invite-actions'

interface Props {
    token: string
    invitation: InvitationPreview
    /** The signed-in account, if any. */
    user: { name: string; email: string } | null
    isAuthenticated: boolean
    busy: boolean
    error: string | null
    onJoin: () => void
    onRegister: (input: RegisterInput) => void
}

/** The invitation itself: join when signed in, otherwise register or sign in. */
export function InviteDetails({ token, invitation, user, isAuthenticated, busy, error, onJoin, onRegister }: Props) {
    const { t } = useTranslation('settings')
    return (
        <>
            <CardHeader>
                <CardTitle>{t('invite.title', { space: invitation.spaceName })}</CardTitle>
                <CardDescription>
                    {t('invite.text', {
                        inviter: invitation.inviter || t('invite.someone'),
                        role: spaceRoleLabel(t, invitation.role),
                    })}
                </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
                {error && <p className="text-sm text-destructive">{error}</p>}
                {isAuthenticated ? (
                    <>
                        <p className="rounded-md border bg-muted/50 p-3 text-sm text-muted-foreground">
                            {t('invite.signedInAs', { who: user ? `${user.name} (${user.email})` : '' })}
                        </p>
                        <Button className="w-full" disabled={busy} onClick={onJoin}>
                            {t('invite.join')}
                        </Button>
                    </>
                ) : (
                    <>
                        {invitation.canRegister && (
                            <InviteRegisterForm email={invitation.email} busy={busy} onSubmit={onRegister} />
                        )}
                        <p className="text-center text-sm text-muted-foreground">
                            {t('invite.haveAccount')}{' '}
                            <Link
                                to="/login"
                                state={{ from: `/invite/${token}` }}
                                className="font-medium text-foreground underline"
                            >
                                {t('invite.signIn')}
                            </Link>
                        </p>
                    </>
                )}
            </CardContent>
        </>
    )
}
