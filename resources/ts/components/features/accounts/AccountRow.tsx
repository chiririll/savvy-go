import { type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { FeedRow, progressAmounts, RowActions } from '@/components/shared'
import { cardProgress } from '@/lib/credit-card'
import { CardStatusIcon } from './CardStatusIcon'
import { ACCOUNT_TYPE_CONFIG } from '@/constants'
import { formatCurrency } from '@/lib/utils'
import { Account } from '@/types'
import { accountTypeLabel } from '@/lib/labels'

const TYPE_ICON_TONES = {
    bank: 'bg-info/15 text-info',
    cash: 'bg-success/15 text-success',
    crypto: 'bg-warning/15 text-warning',
    credit: 'bg-info/15 text-info',
    debt: 'bg-purple-100 text-purple-600 dark:bg-purple-950/40',
    space: 'bg-muted text-muted-foreground',
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
    const card = account.creditCard
    // A card shows what is owed against its limit, like a debt or a budget.
    const cardAmounts = card
        ? {
            ...progressAmounts({ remaining: card.used, total: card.limit, currency: account.currency }),
            amountClassName: card.available < 0 ? 'text-destructive' : undefined,
        }
        : undefined

    return (
        <FeedRow
            leading={dragHandle}
            icon={<TypeIcon className="size-4" />}
            iconClassName={TYPE_ICON_TONES[account.type]}
            title={account.name}
            titleClassName={account.isActive ? undefined : 'text-muted-foreground'}
            subtitle={`${currency} · ${typeLabel}`}
            badge={<CardStatusIcon account={account} />}
            amount={cardAmounts?.amount ?? formatCurrency(account.currentBalance, account.currency)}
            amountClassName={cardAmounts ? cardAmounts.amountClassName : account.currentBalance >= 0 ? 'text-success' : 'text-destructive'}
            extraAmount={cardAmounts
                ? cardAmounts.extraAmount
                : showInitial
                    ? t('pages:accounts.initial', {
                        amount: formatCurrency(account.initialBalance, account.currency),
                    })
                    : undefined}
            progress={card && card.limit > 0
                ? {
                    value: cardProgress(card),
                    exceeded: card.status.level === 'danger' || card.available < 0,
                    warning: card.status.level === 'warning',
                }
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
