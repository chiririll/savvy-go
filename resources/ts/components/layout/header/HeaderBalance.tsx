import { Wallet } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { useAccounts, useTotalBalance } from '@/hooks'
import { ACCOUNT_TYPE_CONFIG } from '@/constants'
import { cn, formatCurrency } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'
import type { AccountType } from '@/types'

export function HeaderBalance() {
    const { t } = useTranslation()
    const { data: balance, isLoading } = useTotalBalance()
    const { data: accounts, isLoading: isAccountsLoading } = useAccounts({
        active: true,
        exclude_debts: true,
    })

    if (isLoading) {
        return (
            <div className="flex items-center gap-2 text-sm">
                <Wallet className="size-4 text-muted-foreground" />
                <Skeleton className="h-4 w-20" />
            </div>
        )
    }

    if (!balance) return null

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button
                    variant="ghost"
                    size="sm"
                    className="gap-2 font-normal"
                    aria-label={t('header.accountBalances')}
                >
                    <Wallet className="size-4 text-muted-foreground" />
                    <span className="font-mono font-medium">
                        {formatCurrency(balance.totalBalance ?? 0, balance.currency)}
                    </span>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent
                align="end"
                collisionPadding={8}
                className="w-64 max-w-[calc(100vw-1rem)]"
            >
                {isAccountsLoading ? (
                    Array.from({ length: 3 }).map((_, index) => (
                        <div
                            key={index}
                            className="flex items-center justify-between gap-3 px-2 py-1.5"
                        >
                            <Skeleton className="h-4 w-24" />
                            <Skeleton className="h-4 w-16" />
                        </div>
                    ))
                ) : !accounts?.length ? (
                    <p className="px-2 py-3 text-center text-sm text-muted-foreground">
                        {t('header.accountsEmpty')}
                    </p>
                ) : (
                    accounts.map((account) => {
                        const config = ACCOUNT_TYPE_CONFIG[account.type as AccountType]
                        const Icon = config?.icon || Wallet

                        return (
                            <div
                                key={account.id}
                                className="flex min-w-0 items-center justify-between gap-3 px-2 py-1.5"
                            >
                                <div className="flex min-w-0 items-center gap-2">
                                    <Icon className={cn('size-3.5 shrink-0', config?.textColor)} />
                                    <span className="truncate text-sm">{account.name}</span>
                                </div>
                                <span className="shrink-0 font-mono text-sm tabular-nums">
                                    {formatCurrency(account.currentBalance ?? 0, account.currency)}
                                </span>
                            </div>
                        )
                    })
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                    <Link to="/accounts">{t('nav:accounts')}</Link>
                </DropdownMenuItem>
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
