import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { spaceRoleLabel } from '@/lib/labels'
import type { SpaceRole } from '@/types/spaces'

interface Props {
    /** The invited address; null is a link anyone can use. */
    email: string | null
    role: SpaceRole
    /** What follows the role in the second line, such as "sent". */
    note?: ReactNode
    /** Replaces the second line, such as a refusal. */
    problem?: string
    children?: ReactNode
}

/** One line of a list of invitations: who, with what role, and the actions for it. */
export function InvitationRow({ email, role, note, problem, children }: Props) {
    const { t } = useTranslation('settings')
    return (
        <li className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
            <div className="min-w-0">
                <p className="truncate">{email ?? t('spaces.invitations.anyone')}</p>
                <p className={problem ? 'text-xs text-destructive' : 'truncate text-xs text-muted-foreground'}>
                    {problem ?? (
                        <>
                            {spaceRoleLabel(t, role)}
                            {note && <> · {note}</>}
                        </>
                    )}
                </p>
            </div>
            {children && <div className="flex shrink-0 items-center gap-1">{children}</div>}
        </li>
    )
}
