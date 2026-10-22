import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ResponsiveDialog } from '@/components/shared/dialogs/ResponsiveDialog'
import { useSwitchSpace } from '@/hooks/use-spaces'
import type { Space } from '@/types/spaces'
import { DoneFooter, DoneStep } from './create/DoneStep'
import { useInviteAfterCreate } from './create/useInviteAfterCreate'
import { useWizardStep } from './create/useWizardStep'

interface Props {
    open: boolean
    onOpenChange: (open: boolean) => void
    /** A user without any space cannot close the modal. */
    dismissible?: boolean
}

/**
 * Builds a filled space in steps and sends it in one request. The draft lives
 * in this browser only (stores/space-draft.ts) and survives closing the modal.
 */
export function CreateSpaceModal({ open, onOpenChange, dismissible = true }: Props) {
    const { t } = useTranslation('settings')
    const switchSpace = useSwitchSpace()
    const [created, setCreated] = useState<Space | null>(null)

    const done = useInviteAfterCreate(created, () => void finish())
    const wizard = useWizardStep({ open, onClose: () => onOpenChange(false), dismissible, onCreated: setCreated })

    const finish = async () => {
        const space = created
        setCreated(null)
        done.reset()
        onOpenChange(false)
        if (space) await switchSpace(space.id)
    }

    const requestClose = (next: boolean) => {
        if (next) return onOpenChange(true)
        if (created) return done.leave()
        if (dismissible) onOpenChange(false)
    }

    const body = created ? (
        <DoneStep
            space={created}
            inviting={done.inviting}
            invitations={done.invitations}
            onInvitationsChange={done.setInvitations}
            sent={done.sent}
        />
    ) : (
        wizard.body
    )
    const footer = created ? (
        <DoneFooter
            inviting={done.inviting}
            sent={done.sent}
            isSending={done.isSending}
            onInvite={() => done.setInviting(true)}
            onLeave={done.leave}
        />
    ) : (
        wizard.footer
    )

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={requestClose}
            title={created ? t('spaces.create.done.title') : t('spaces.create.title')}
            description={created ? undefined : t('spaces.create.description')}
            className="sm:max-w-2xl"
            footer={footer}
        >
            {body}
        </ResponsiveDialog>
    )
}
