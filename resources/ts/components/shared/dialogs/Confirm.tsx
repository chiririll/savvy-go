import type { ReactElement, ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { testId } from '@/lib/test-id'

interface ConfirmProps {
    /** The element that opens the dialog. Leave it out and control the dialog with `open`. */
    trigger?: ReactElement
    open?: boolean
    onOpenChange?: (open: boolean) => void
    title: string
    /** A string is the whole text; other content is wrapped in a block, so it may hold several paragraphs. */
    description?: ReactNode
    /** The label of the button that does it. */
    action: string
    /** The label of the button that does not; "Cancel" when left out. */
    cancel?: string
    destructive?: boolean
    onConfirm: () => void
    /**
     * For a request that may fail. Given (even as false), the dialog stays open after the click until the caller
     * closes it, and the action shows a spinner while this is true. Left out, the dialog closes at once.
     */
    pending?: boolean
    actionTestId?: string
    cancelTestId?: string
}

/**
 * A yes/no question with no inputs; a dialog with inputs is a ResponsiveDialog or FormDialog.
 * Not dismissed by a click outside, so an answer is never given by accident.
 */
export function Confirm({
    trigger,
    open,
    onOpenChange,
    title,
    description,
    action,
    cancel,
    destructive,
    onConfirm,
    pending,
    actionTestId,
    cancelTestId,
}: ConfirmProps) {
    const { t } = useTranslation('common')
    const variant = destructive ? 'destructive' : 'default'
    const busy = pending === true

    return (
        <AlertDialog open={open} onOpenChange={onOpenChange}>
            {trigger && (
                <AlertDialogTrigger asChild {...testId('confirm')}>
                    {trigger}
                </AlertDialogTrigger>
            )}
            <AlertDialogContent>
                <AlertDialogHeader>
                    <AlertDialogTitle>{title}</AlertDialogTitle>
                    {typeof description === 'string' ? (
                        <AlertDialogDescription>{description}</AlertDialogDescription>
                    ) : (
                        description != null && (
                            <AlertDialogDescription asChild>
                                <div className="space-y-2">{description}</div>
                            </AlertDialogDescription>
                        )
                    )}
                </AlertDialogHeader>
                <AlertDialogFooter>
                    <AlertDialogCancel disabled={busy} {...(cancelTestId ? testId(cancelTestId) : {})}>
                        {cancel ?? t('actions.cancel')}
                    </AlertDialogCancel>
                    {pending === undefined ? (
                        <AlertDialogAction
                            variant={variant}
                            onClick={onConfirm}
                            {...(actionTestId ? testId(actionTestId) : {})}
                        >
                            {action}
                        </AlertDialogAction>
                    ) : (
                        <Button
                            type="button"
                            variant={variant}
                            disabled={busy}
                            onClick={onConfirm}
                            {...(actionTestId ? testId(actionTestId) : {})}
                        >
                            {busy && <Loader2 className="mr-2 size-4 animate-spin" />}
                            {action}
                        </Button>
                    )}
                </AlertDialogFooter>
            </AlertDialogContent>
        </AlertDialog>
    )
}
