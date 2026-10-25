import { useTranslation } from 'react-i18next'
import { ArrowDownLeft, ArrowLeftRight, ArrowUpRight, Plus, Wallet } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useCreateTransactionDialog } from '@/components/features/transactions'
import { formatCurrency } from '@/lib/utils'
import { accountTypeLabelLoose } from '@/lib/labels'
import { ACCOUNT_TYPE_CONFIG } from '@/constants'
import type { Account, AccountType } from '@/types'

type OpenCreate = ReturnType<typeof useCreateTransactionDialog>['openCreate']

function AccountIcon({ type }: { type: Account['type'] }) {
    const config = ACCOUNT_TYPE_CONFIG[type as AccountType]
    const Icon = config?.icon || Wallet

    return (
        <div className={`flex size-9 items-center justify-center rounded-lg shrink-0 ${config?.color || 'bg-muted'}`}>
            <Icon className="size-4" />
        </div>
    )
}

function QuickAddMenu({ accountId, openCreate }: { accountId: Account['id']; openCreate: OpenCreate }) {
    const { t: tNav } = useTranslation('nav')
    const { t: tCommon } = useTranslation('common')

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="size-8 shrink-0" aria-label={tCommon('actions.create')}>
                    <Plus className="size-4" />
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => openCreate({ type: 'income', account_id: accountId })}>
                    <ArrowDownLeft className="size-4 mr-2 text-success" />
                    {tNav('income')}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => openCreate({ type: 'expense', account_id: accountId })}>
                    <ArrowUpRight className="size-4 mr-2 text-destructive" />
                    {tNav('expense')}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => openCreate({ type: 'transfer', account_id: accountId })}>
                    <ArrowLeftRight className="size-4 mr-2 text-info" />
                    {tNav('transfer')}
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    )
}

function AccountRow({ account, openCreate }: { account: Account; openCreate: OpenCreate }) {
    const { t } = useTranslation('pages')

    return (
        <div className="flex min-w-0 items-center justify-between gap-2">
            <div className="flex items-center gap-3 min-w-0 flex-1">
                <AccountIcon type={account.type} />
                <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{account.name}</p>
                    <p className="text-xs text-muted-foreground capitalize">
                        {accountTypeLabelLoose(t, account.type)}
                    </p>
                </div>
            </div>
            <div className="flex min-w-0 items-center gap-2">
                <div className="min-w-0 text-right">
                    <p className="text-sm font-mono font-medium break-all">
                        {formatCurrency(account.currentBalance ?? 0, account.currency, { showSymbol: false })}
                    </p>
                    <p className="text-xs text-muted-foreground">{account.currency?.symbol ?? ''}</p>
                </div>
                <QuickAddMenu accountId={account.id} openCreate={openCreate} />
            </div>
        </div>
    )
}

export function AccountBalancesCard({ accounts, openCreate }: { accounts?: Account[]; openCreate: OpenCreate }) {
    const { t } = useTranslation('pages')

    return (
        <Card className="min-w-0">
            <CardHeader>
                <CardTitle>{t('dashboard.accountBalances')}</CardTitle>
            </CardHeader>
            <CardContent className="min-w-0">
                <div className="min-w-0 space-y-3">
                    {accounts && accounts.length > 0 ? (
                        accounts.map((account) => (
                            <AccountRow key={account.id} account={account} openCreate={openCreate} />
                        ))
                    ) : (
                        <div className="text-center text-muted-foreground py-4">
                            {t('dashboard.noAccounts')}
                        </div>
                    )}
                </div>
            </CardContent>
        </Card>
    )
}
