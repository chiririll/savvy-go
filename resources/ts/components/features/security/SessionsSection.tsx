import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { LogOut } from 'lucide-react'
import { testId } from '@/lib/test-id'
import { ResponsiveDialog, SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { useLogoutOthers } from '@/hooks'


export function SessionsSection() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const logoutOthers = useLogoutOthers()
    const [confirmOpen, setConfirmOpen] = useState(false)

    const handleConfirm = async () => {
        try {
            await logoutOthers.mutateAsync()
            setConfirmOpen(false)
        } catch {
            // toasted by the mutation
        }
    }

    return (
        <>
            <SettingsSection
                icon={LogOut}
                title={t('security.sessions.title')}
                description={t('security.sessions.description')}
            >
                <div className="space-y-4">
                    <p className="text-sm leading-relaxed text-muted-foreground">
                        {t('security.sessions.logoutOthersDescription')}
                    </p>
                    <Button variant="outline" onClick={() => setConfirmOpen(true)} {...testId('logout-others')}>
                        <LogOut className="mr-2 size-4" />
                        {t('security.sessions.logoutOthers')}
                    </Button>
                </div>
            </SettingsSection>

            <ResponsiveDialog
                open={confirmOpen}
                onOpenChange={setConfirmOpen}
                title={t('security.sessions.confirmTitle')}
                description={t('security.sessions.confirmDescription')}
                footer={
                    <>
                        <Button variant="outline" onClick={() => setConfirmOpen(false)}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button variant="destructive" disabled={logoutOthers.isPending} onClick={handleConfirm}>
                            {t('security.sessions.confirm')}
                        </Button>
                    </>
                }
            />
        </>
    )
}
