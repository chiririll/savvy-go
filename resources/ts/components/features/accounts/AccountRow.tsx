import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { FeedRow, RowActions } from '@/components/shared'
import { ACCOUNT_TYPE_CONFIG } from '@/constants'
import { formatCurrency } from '@/lib/utils'
import { Account } from '@/types'
import { accountTypeLabel } from '@/lib/labels'

const TYPE_ICON_TONES = {
    bank: 'bg-blue-100 text-blue-600 dark:bg-blue-950/40',
    cash: 'bg-green-100 text-green-600 dark:bg-green-950/40',
    crypto: 'bg-orange-100 text-orange-600 dark:bg-orange-950/40',
    debt: 'bg-purple-100 text-purple-600 dark:bg-purple-950/40',
} as const

interface AccountRowProps {
    account: Account
    onEdit?: (account: Account) => void
    onDelete?: (id: number) => void
    isReadOnly?: boolean
    dragHandle?: ReactNode
}

export function AccountRow({
    account,
    onEdit,
    onDelete,
    isReadOnly,
    dragHandle,
}: AccountRowProps) {
    const { t } = useTranslation(['common', 'pages'])
    const TypeIcon = ACCOUNT_TYPE_CONFIG[account.type].icon
    const currency = account.currency?.code ?? t('na')
    const typeLabel = accountTypeLabel(t, account.type)
    const showInitial = account.initialBalance !== account.currentBalance
    const canEdit = !!onEdit && !isReadOnly
    const canDelete = !!onDelete && !isReadOnly
    const hasActions = canEdit || canDelete

    return (
        <FeedRow
            leading={dragHandle}
            icon={<TypeIcon className="size-4" />}
            iconClassName={TYPE_ICON_TONES[account.type]}
            title={account.name}
            titleClassName={account.isActive ? undefined : 'text-muted-foreground'}
            subtitle={`${currency} · ${typeLabel}`}
            amount={formatCurrency(account.currentBalance, account.currency)}
            amountClassName={account.currentBalance >= 0 ? 'text-green-600' : 'text-red-600'}
            extraAmount={showInitial
                ? t('pages:accounts.initial', {
                    amount: formatCurrency(account.initialBalance, account.currency),
                })
                : undefined}
            onOpen={canEdit ? () => onEdit(account) : undefined}
            hasActions={hasActions}
            actions={({ menuOpen, setMenuOpen, isMobile }) => (
                <RowActions
                    open={menuOpen}
                    onOpenChange={setMenuOpen}
                    showTrigger={!isMobile}
                    onEdit={canEdit ? () => onEdit(account) : undefined}
                    onDelete={canDelete ? () => onDelete(account.id) : undefined}
                    deleteTitle={t('pages:accounts.deleteTitle')}
                    deleteDescription={t('pages:accounts.deleteDescription', { name: account.name })}
                />
            )}
        />
    )
}
