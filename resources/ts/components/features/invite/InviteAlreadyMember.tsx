import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

/** Shown when the signed-in account is already in the space. */
export function InviteAlreadyMember({ spaceName }: { spaceName: string }) {
    const { t } = useTranslation('settings')
    return (
        <>
            <CardHeader>
                <CardTitle>{t('invite.alreadyMemberTitle')}</CardTitle>
                <CardDescription>{t('invite.alreadyMemberText', { space: spaceName })}</CardDescription>
            </CardHeader>
            <CardContent>
                <Button asChild className="w-full">
                    <Link to="/">{t('invite.goHome')}</Link>
                </Button>
            </CardContent>
        </>
    )
}
