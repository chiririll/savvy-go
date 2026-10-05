import { testId, testIdMenu } from '@/lib/test-id'
import { Plus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { useCreateTransactionDialog } from '@/components/features/transactions'
import { SPACE_TRANSFER_OPTION, TRANSACTION_TYPE_OPTIONS } from '@/constants'
import { useTransferTargets } from '@/hooks'
import { createMenuLabel } from '@/lib/labels'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'

export function CreateTransactionMenu() {
    const { t } = useTranslation('nav')
    const { openCreate } = useCreateTransactionDialog()
    const isReadOnly = useReadOnly()
    const canTransfer = useTransferTargets().length > 0
    const options = [...TRANSACTION_TYPE_OPTIONS, ...(canTransfer ? [SPACE_TRANSFER_OPTION] : [])]

    if (isReadOnly) {
        return null
    }

    return (
        <DropdownMenu>
            <DropdownMenuTrigger asChild>
                <Button size="sm" className="gap-1" {...testIdMenu('layout-transaction')}>
                    <Plus className="size-4" />
                    <span className="hidden sm:inline">{t('newTransaction')}</span>
                </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
                {options.map(({ value, icon: Icon, color }) => (
                    <DropdownMenuItem
                        key={value}
                        {...testId(`layout-transaction-${value}`)}
                        onClick={() => openCreate({ type: value })}
                    >
                        <Icon className={`size-4 mr-2 ${color}`} />
                        {createMenuLabel(t, value)}
                    </DropdownMenuItem>
                ))}
            </DropdownMenuContent>
        </DropdownMenu>
    )
}
