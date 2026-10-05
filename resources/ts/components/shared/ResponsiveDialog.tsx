import type { ReactNode } from 'react'
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog'
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetFooter,
    SheetHeader,
    SheetTitle,
} from '@/components/ui/sheet'
import { useIsMobile } from '@/hooks/use-mobile'
import { cn } from '@/lib/utils'

interface ResponsiveDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    title: string
    description?: string
    /** Action buttons; rendered in a sticky footer. */
    footer?: ReactNode
    className?: string
    children?: ReactNode
}

/**
 * Generic modal: a centered dialog on desktop, a bottom sheet on mobile.
 * Same layout as FormDialog, for dialogs that are not entity forms.
 */
export function ResponsiveDialog({
    open,
    onOpenChange,
    title,
    description,
    footer,
    className,
    children,
}: ResponsiveDialogProps) {
    const isMobile = useIsMobile()
    const body = children != null && (
        <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4">{children}</div>
    )

    if (isMobile) {
        return (
            <Sheet open={open} onOpenChange={onOpenChange}>
                <SheetContent
                    side="bottom"
                    className="flex max-h-[90dvh] flex-col gap-0 overflow-hidden rounded-t-xl p-0"
                >
                    <div className="flex justify-center pt-3">
                        <div className="h-1 w-10 rounded-full bg-muted-foreground/25" />
                    </div>
                    <SheetHeader className={cn('shrink-0 space-y-1.5 px-6 pb-4 pr-12 pt-3 text-left', children != null && 'border-b')}>
                        <SheetTitle>{title}</SheetTitle>
                        {description && <SheetDescription>{description}</SheetDescription>}
                    </SheetHeader>
                    {body}
                    {footer && (
                        <SheetFooter className="shrink-0 flex-row flex-wrap items-center justify-end gap-2 border-t px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
                            {footer}
                        </SheetFooter>
                    )}
                </SheetContent>
            </Sheet>
        )
    }

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className={cn('flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-md', className)}>
                <DialogHeader className={cn('shrink-0 space-y-1.5 px-6 pb-4 pr-12 pt-6', children != null && 'border-b')}>
                    <DialogTitle>{title}</DialogTitle>
                    {description && <DialogDescription>{description}</DialogDescription>}
                </DialogHeader>
                {body}
                {footer && (
                    <DialogFooter className="shrink-0 flex-row flex-wrap items-center border-t px-6 py-4 sm:justify-end">
                        {footer}
                    </DialogFooter>
                )}
            </DialogContent>
        </Dialog>
    )
}
