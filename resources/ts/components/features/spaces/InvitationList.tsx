import { Children, type ReactNode } from 'react'
import { testId } from '@/lib/test-id'

/** The list of invitations, or the line saying there are none. Rows are `InvitationRow`s. */
export function InvitationList({ empty, children }: { empty: string; children?: ReactNode }) {
    return Children.toArray(children).length > 0 ? (
        <ul className="max-h-56 divide-y overflow-y-auto rounded-lg border" {...testId('invitation-list')}>
            {children}
        </ul>
    ) : (
        <p className="text-sm text-muted-foreground" {...testId('invitation-empty')}>
            {empty}
        </p>
    )
}
