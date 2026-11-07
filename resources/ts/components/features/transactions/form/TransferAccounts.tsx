import type { LinkedSpace } from '@/types/spaces'
import type { Account } from '@/types'
import type { BalancePreview } from './balance-preview'
import type { BalanceMode } from './TransactionBalanceHint'
import { SpaceTransferTargetField } from './SpaceTransferTargetField'
import { TransferSection } from './TransferSection'
import type { useTransferRate } from './transfer-rate'

interface TransferAccountsProps {
    isSpaceTransfer: boolean
    /** The linked spaces a transfer can go to. */
    links: LinkedSpace[]
    /** The accounts of the chosen linked space; undefined until they are loaded. */
    destinationAccounts?: Account[]
    destinationSpaceId?: number | null
    from?: Account
    to?: Account
    rate: ReturnType<typeof useTransferRate>
    balances: BalancePreview[]
    balanceMode: BalanceMode
}

/** Where a transfer goes: the linked space (if it leaves this one), then both accounts and their amounts (the rate is in the row of the date). */
export function TransferAccounts({
    isSpaceTransfer,
    links,
    destinationAccounts,
    destinationSpaceId,
    from,
    to,
    rate,
    balances,
    balanceMode,
}: TransferAccountsProps) {
    return (
        <div className="space-y-4">
            {isSpaceTransfer && (
                <SpaceTransferTargetField links={links} noAccounts={destinationAccounts?.length === 0} />
            )}
            <TransferSection
                from={from}
                to={to}
                rate={rate}
                destinationSpaceId={destinationSpaceId}
                balances={balances}
                balanceMode={balanceMode}
            />
        </div>
    )
}
