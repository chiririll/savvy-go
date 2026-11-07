import { Plus, X } from 'lucide-react'
import { Button } from '@/components/ui/button'

const SIZE = 'size-8 shrink-0'

interface PaymentRowButtonsProps {
    /** Removes the row; left out when the row is the only one and has to stay. */
    onRemove?: () => void
    removeLabel: string
    /** Adds a row after this one; left out on every row but the last. */
    onAdd?: () => void
    addLabel: string
    /** Shown on the (+) when it is off: why no row can be added. */
    addDisabledReason?: string
    addDisabled?: boolean
}

/** The (X) and (+) at the end of a payment row. */
export function PaymentRowButtons({
    onRemove,
    removeLabel,
    onAdd,
    addLabel,
    addDisabledReason,
    addDisabled,
}: PaymentRowButtonsProps) {
    if (!onRemove && !onAdd) {
        return null
    }

    return (
        <div className="flex shrink-0 items-center">
            {onRemove && (
                <Button type="button" variant="ghost" size="icon" className={SIZE} aria-label={removeLabel} title={removeLabel} onClick={onRemove}>
                    <X />
                </Button>
            )}
            {onAdd && (
                // A disabled button takes no hover, so the reason sits on the wrapper.
                <span title={addDisabled ? addDisabledReason : addLabel}>
                    <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className={SIZE}
                        aria-label={addLabel}
                        disabled={addDisabled}
                        onClick={onAdd}
                    >
                        <Plus />
                    </Button>
                </span>
            )}
        </div>
    )
}
