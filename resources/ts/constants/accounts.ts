import { Landmark, Wallet, Bitcoin, HandCoins, CreditCard, Orbit } from 'lucide-react'
import type { AccountType, FormAccountType } from '@/types'

export interface AccountTypeConfig {
    icon: typeof Landmark
    label: string
    color: string
    bgColor: string
    textColor: string
}

// All account types including debt (for display purposes)
export const ACCOUNT_TYPE_CONFIG: Record<AccountType, AccountTypeConfig> = {
    bank: {
        icon: Landmark,
        label: 'Bank',
        color: 'bg-info/15 text-info',
        bgColor: 'bg-info/15',
        textColor: 'text-info',
    },
    cash: {
        icon: Wallet,
        label: 'Cash',
        color: 'bg-success/15 text-success',
        bgColor: 'bg-success/15',
        textColor: 'text-success',
    },
    crypto: {
        icon: Bitcoin,
        label: 'Crypto',
        color: 'bg-warning/15 text-warning',
        bgColor: 'bg-warning/15',
        textColor: 'text-warning',
    },
    credit: {
        icon: CreditCard,
        label: 'Credit card',
        color: 'bg-info/15 text-info',
        bgColor: 'bg-info/15',
        textColor: 'text-info',
    },
    space: {
        icon: Orbit,
        label: 'Other space',
        color: 'bg-muted text-muted-foreground',
        bgColor: 'bg-muted',
        textColor: 'text-muted-foreground',
    },
    debt: {
        icon: HandCoins,
        label: 'Debt',
        color: 'bg-purple-100 text-purple-700',
        bgColor: 'bg-purple-100',
        textColor: 'text-purple-600',
    },
}

export const FORM_ACCOUNT_TYPE_CONFIG: Record<FormAccountType, AccountTypeConfig> = {
    bank: ACCOUNT_TYPE_CONFIG.bank,
    cash: ACCOUNT_TYPE_CONFIG.cash,
    crypto: ACCOUNT_TYPE_CONFIG.crypto,
    credit: ACCOUNT_TYPE_CONFIG.credit,
}
