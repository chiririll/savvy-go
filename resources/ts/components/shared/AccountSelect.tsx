import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { FormControl } from '@/components/ui/form'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAccounts, useCreateAccount } from '@/hooks'
import { AccountFormDialog } from '@/components/features/accounts'
import { ACCOUNT_TYPE_CONFIG } from '@/constants'
import { cn, formatCurrency } from '@/lib/utils'
import { Plus, Wallet } from 'lucide-react'
import type { AccountType } from '@/types'

interface AccountSelectProps {
    value?: number | null
    onChange: (value: number) => void
    excludeId?: number | null
    excludeDebts?: boolean
    activeOnly?: boolean
    placeholder?: string
    disabled?: boolean
    plain?: boolean
    showBalance?: boolean
    allowCreate?: boolean
}

const NEW_ACCOUNT_VALUE = '__new__'

export function AccountSelect({
    value,
    onChange,
    excludeId,
    excludeDebts = true,
    activeOnly = true,
    placeholder,
    disabled,
    plain,
    showBalance,
    allowCreate,
}: AccountSelectProps) {
    const { t } = useTranslation(['forms', 'pages'])
    const [createOpen, setCreateOpen] = useState(false)
    const createAccount = useCreateAccount()
    const { data: accounts } = useAccounts({ active: activeOnly, exclude_debts: excludeDebts })

    const selectedId = Number(value) > 0 ? Number(value) : null
    const filteredAccounts = accounts?.filter((account) => {
        if (selectedId && account.id === selectedId) {
            return true
        }
        if (excludeId && account.id === excludeId) {
            return false
        }
        return true
    })

    const trigger = (
        <SelectTrigger className="w-full">
            <SelectValue placeholder={placeholder ?? t('forms:selectAccount')} />
        </SelectTrigger>
    )

    const select = (
        <Select
            key={selectedId ?? 'empty'}
            onValueChange={(val) => {
                if (val === NEW_ACCOUNT_VALUE) {
                    setCreateOpen(true)
                    return
                }
                onChange(Number(val))
            }}
            value={selectedId ? String(selectedId) : undefined}
            disabled={disabled}
        >
            {plain ? trigger : <FormControl>{trigger}</FormControl>}
            <SelectContent>
                {filteredAccounts?.map((account) => {
                    const config = ACCOUNT_TYPE_CONFIG[account.type as AccountType]
                    const Icon = config?.icon || Wallet
                    return (
                        <SelectItem
                            key={account.id}
                            value={account.id.toString()}
                        >
                            <div className="flex items-center gap-2">
                                <Icon className={cn('size-4', config?.textColor)} />
                                <span>{account.name}</span>
                                {showBalance ? (
                                    <span className="text-muted-foreground text-xs font-mono">
                                        {formatCurrency(account.currentBalance, account.currency)}
                                    </span>
                                ) : (
                                    <span className="text-muted-foreground">
                                        {account.currency?.symbol}
                                    </span>
                                )}
                            </div>
                        </SelectItem>
                    )
                })}
                {allowCreate && (
                    <SelectItem value={NEW_ACCOUNT_VALUE}>
                        <div className="flex items-center gap-2">
                            <Plus className="size-4" />
                            <span>{t('pages:accounts.create')}</span>
                        </div>
                    </SelectItem>
                )}
            </SelectContent>
        </Select>
    )

    if (!allowCreate) {
        return select
    }

    return (
        <>
            {select}
            <AccountFormDialog
                open={createOpen}
                onOpenChange={setCreateOpen}
                isSubmitting={createAccount.isPending}
                onSubmit={async (data) => {
                    const account = await createAccount.mutateAsync(data)
                    setCreateOpen(false)
                    onChange(account.id)
                }}
            />
        </>
    )
}
