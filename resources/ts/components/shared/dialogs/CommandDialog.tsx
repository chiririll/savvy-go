import type { ReactNode, RefObject } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogTitle } from '@/components/ui/dialog'

interface CommandDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    /** Read by screen readers only; the input inside is the visible heading. */
    title: string
    description: string
    /** Focused when the dialog opens. */
    initialFocus?: RefObject<HTMLElement | null>
    children: ReactNode
}

/** A palette: a search input with its results, near the top of the screen, with no header or footer. */
export function CommandDialog({ open, onOpenChange, title, description, initialFocus, children }: CommandDialogProps) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent
                className="top-[20%] max-w-xl translate-y-0 gap-0 p-0"
                onOpenAutoFocus={(event) => {
                    event.preventDefault()
                    initialFocus?.current?.focus()
                }}
            >
                <DialogTitle className="sr-only">{title}</DialogTitle>
                <DialogDescription className="sr-only">{description}</DialogDescription>
                {children}
            </DialogContent>
        </Dialog>
    )
}
