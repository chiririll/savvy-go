import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { testId } from '@/lib/test-id'
import type { Account } from '@/types'

const OPTION_CLASS = [
    'flex w-full items-center justify-between gap-2 rounded-sm px-2 py-1.5 text-left text-sm',
    'hover:bg-accent focus-visible:bg-accent focus-visible:outline-none',
].join(' ')

interface AddAccountPopoverProps {
    /** The accounts that can be added: the ones not in the rows. */
    accounts: readonly Account[]
    onPick: (accountId: number) => void
}

/** `+ Add account`: a list of the accounts taken out of the reconciliation. */
export function AddAccountPopover({ accounts, onPick }: AddAccountPopoverProps) {
    const { t } = useTranslation('pages')
    const [open, setOpen] = useState(false)

    return (
        <Popover modal open={open} onOpenChange={setOpen}>
            <PopoverTrigger asChild>
                <Button type="button" variant="outline" size="sm" {...testId('reconcile-add')}>
                    <Plus className="size-4" />
                    {t('accounts.reconcile.add')}
                </Button>
            </PopoverTrigger>
            <PopoverContent align="end" className="max-h-64 w-64 overflow-y-auto p-1">
                <ul>
                    {accounts.map((account) => (
                        <li key={account.id}>
                            <button
                                type="button"
                                className={OPTION_CLASS}
                                onClick={() => {
                                    onPick(account.id)
                                    setOpen(false)
                                }}
                            >
                                <span className="truncate">{account.name}</span>
                                <span className="shrink-0 text-xs text-muted-foreground">{account.currency?.code}</span>
                            </button>
                        </li>
                    ))}
                </ul>
            </PopoverContent>
        </Popover>
    )
}
