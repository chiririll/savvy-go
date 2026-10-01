import { ColumnDef } from '@tanstack/react-table'
import { KeyRound } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { DropdownMenuItem } from '@/components/ui/dropdown-menu'
import { RowActions, UserAvatar } from '@/components/shared'
import { User } from '@/types/users'
import i18n from '@/lib/i18n'

export const createUserColumns = (
    onDelete: (id: number) => void,
    onEdit: (user: User) => void,
    onResetPassword: (user: User) => void,
    currentUserId?: number,
    isReadOnly?: boolean
): ColumnDef<User>[] => [
    {
        accessorKey: 'name',
        header: () => i18n.t('pages:users.columns.user'),
        cell: ({ row }) => (
            <div className="flex items-center gap-3">
                <UserAvatar user={row.original} className="size-10" />
                <div>
                    <div className="flex items-center gap-2">
                        <p className="font-medium">{row.original.name}</p>
                        {row.original.isInactive && (
                            <Badge variant="secondary">{i18n.t('status.inactive')}</Badge>
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
        cell: ({ row }) => (
            <span className="text-muted-foreground">{row.original.email}</span>
        ),
    },
    {
        accessorKey: 'role',
        header: () => i18n.t('pages:users.columns.role'),
        cell: ({ row }) => (
            <span>{i18n.t(`roles.${row.original.role}`)}</span>
        ),
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
