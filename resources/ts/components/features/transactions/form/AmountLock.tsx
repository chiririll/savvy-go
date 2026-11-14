import { Lock, LockOpen } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'
import type { PaymentLock } from './use-unlocked-payments'

/**
 * The lock at the end of a payment amount. Closed: the amount is fixed and the lines split it. Open: the amount is
 * the sum of the lines and cannot be typed. A row that cannot follow the lines keeps it closed and says why.
 */
export function AmountLock({ lock, index }: { lock: PaymentLock; index: number }) {
    const { t } = useTranslation('forms')
    const label = lock.follows
        ? t('transactions.amountFollowsLines')
        : lock.blocker === 'several'
          ? t('transactions.amountFixedSeveral')
          : lock.blocker === 'noLine'
            ? t('transactions.amountFixedNoLine')
            : lock.blocker === 'foreignChange'
              ? t('transactions.amountFixedForeignChange')
              : t('transactions.amountFixed')

    return (
        // A disabled button takes no hover, so the reason sits on the wrapper.
        <span title={label}>
            <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-7 text-muted-foreground"
                aria-label={label}
                aria-pressed={lock.follows}
                disabled={lock.blocker !== null}
                onClick={lock.toggle}
                {...testId(`payments.${index}.lock`)}
            >
                {lock.follows ? <LockOpen /> : <Lock />}
            </Button>
        </span>
    )
}
