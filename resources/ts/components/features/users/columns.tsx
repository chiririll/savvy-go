import { ColumnDef } from '@tanstack/react-table'
import { KeyRound, Mail, MailCheck, MailX } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { RowActions, UserAvatar } from '@/components/shared'
import { User } from '@/types/users'
import i18n from '@/lib/i18n'
import { userRoleLabel } from '@/lib/labels'

interface UserColumnsOptions {
    onDelete: (id: number) => void
    onEdit: (user: User) => void
    onResetPassword: (user: User) => void
    currentUserId?: number
    isReadOnly?: boolean
    /** Given when the server can send mail: adds "email a reset link". */
    onEmailReset?: (user: User) => void
    /** An admin vouching for (true) or withdrawing (false) a user's address. */
    onSetEmailVerified?: (user: User, verified: boolean) => void
    /** Whether confirming an address means anything here (mail works or confirmation is required): shows the badge and the menu item. */
    showUnverified?: boolean
}

export const createUserColumns = ({
    onDelete,
    onEdit,
    onResetPassword,
    currentUserId,
    isReadOnly,
    onEmailReset,
    onSetEmailVerified,
    showUnverified,
}: UserColumnsOptions): ColumnDef<User>[] => [
    {
        accessorKey: 'name',
        header: () => i18n.t('pages:users.columns.user'),
        cell: ({ row }) => (
            <div className="flex items-center gap-3">
                <UserAvatar user={row.original} className="size-10" />
                <div>
                    <div className="flex items-center gap-2">
                        <p className="font-medium">{row.original.name}</p>
                        {row.original.isInactive && <Badge variant="secondary">{i18n.t('status.inactive')}</Badge>}
                        {showUnverified && !row.original.emailVerified && (
                            <Badge variant="outline">{i18n.t('pages:users.emailUnverified')}</Badge>
                        )}
                    </div>
                    <p className="text-xs text-muted-foreground">{row.original.email}</p>
                </div>
            </div>
        ),
    },
    {
        accessorKey: 'email',
        header: () => i18n.t('pages:users.columns.email'),
        cell: ({ row }) => <span className="text-muted-foreground">{row.original.email}</span>,
    },
    {
        accessorKey: 'role',
        header: () => i18n.t('pages:users.columns.role'),
        cell: ({ row }) => <span>{userRoleLabel(i18n.t, row.original.role)}</span>,
    },
    {
        id: 'actions',
        header: '',
        cell: ({ row }) => {
            const isCurrentUser = currentUserId === row.original.id
            const canReset = !isCurrentUser && !row.original.isSsoOnly && !isReadOnly

            return (
                <RowActions
                    onEdit={() => onEdit(row.original)}
                    onDelete={() => onDelete(row.original.id)}
                    deleteTitle={i18n.t('pages:users.deleteTitle')}
                    deleteDescription={i18n.t('pages:users.deleteDescription', { name: row.original.name })}
                    deleteDisabled={isCurrentUser}
                    deleteDisabledLabel={i18n.t('actions.cannotDeleteSelf')}
                    isReadOnly={isReadOnly}
                >
                    {canReset && onEmailReset && (
                        <DropdownMenuItem onClick={() => onEmailReset(row.original)}>
                            <Mail className="mr-2 size-4" />
                            {i18n.t('pages:users.sendResetEmail')}
                        </DropdownMenuItem>
                    )}
                    {!isReadOnly && showUnverified && onSetEmailVerified && (
                        <DropdownMenuItem onClick={() => onSetEmailVerified(row.original, !row.original.emailVerified)}>
                            {row.original.emailVerified ? (
                                <MailX className="mr-2 size-4" />
                            ) : (
                                <MailCheck className="mr-2 size-4" />
                            )}
                            {row.original.emailVerified
                                ? i18n.t('pages:users.markUnverified')
                                : i18n.t('pages:users.markVerified')}
                        </DropdownMenuItem>
                    )}
                    {canReset && (
                        <DropdownMenuItem onClick={() => onResetPassword(row.original)}>
                            <KeyRound className="mr-2 size-4" />
                            {i18n.t('pages:users.resetPassword')}
                        </DropdownMenuItem>
                    )}
                </RowActions>
            )
        },
    },
]
