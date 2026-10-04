import { createContext, useContext, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
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
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { useIsMobile } from '@/hooks/use-mobile'
import { cn } from '@/lib/utils'

const FormDialogFooterStartContext = createContext<HTMLElement | null | undefined>(undefined)

export function FormDialogFooterStart({ children }: { children: ReactNode }) {
    const slot = useContext(FormDialogFooterStartContext)

    if (slot === undefined) {
        return children
    }

    if (!slot) {
        return null
    }

    return createPortal(children, slot)
}

interface FormDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    title: string
    description?: string
    formId: string
    isSubmitting?: boolean
    isEdit?: boolean
    className?: string
    headerExtra?: ReactNode
    children: ReactNode
}

function FormDialogActions({
    formId,
    isSubmitting,
    isEdit,
    onCancel,
}: {
    formId: string
    isSubmitting?: boolean
    isEdit: boolean
    onCancel: () => void
}) {
    const { t } = useTranslation('common')
    const isReadOnly = useReadOnly()

    return (
        <>
            <Button type="button" variant="outline" onClick={onCancel}>
                {t('actions.cancel')}
            </Button>
            {!isReadOnly && (
                <Button type="submit" form={formId} disabled={isSubmitting}>
                    {isSubmitting
                        ? t('actions.saving')
                        : isEdit
                            ? t('actions.save')
                            : t('actions.create')}
                </Button>
            )}
        </>
    )
}

export function FormDialog({
    open,
    onOpenChange,
    title,
    description,
    formId,
    isSubmitting,
    isEdit = false,
    className,
    headerExtra,
    children,
}: FormDialogProps) {
    const isMobile = useIsMobile()
    const [footerStart, setFooterStart] = useState<HTMLDivElement | null>(null)
    const footerStartSlot = (
        <div ref={setFooterStart} className="mr-auto flex min-w-0 flex-wrap items-center gap-x-4 gap-y-2 empty:hidden" />
    )
    const body = (
        <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4">
            {children}
        </div>
    )
    const actions = (
        <FormDialogActions
            formId={formId}
            isSubmitting={isSubmitting}
            isEdit={isEdit}
            onCancel={() => onOpenChange(false)}
        />
    )

    return (
        <FormDialogFooterStartContext.Provider value={footerStart}>
            {isMobile ? (
                <Sheet open={open} onOpenChange={onOpenChange}>
                    <SheetContent
                        side="bottom"
                        className="flex max-h-[90dvh] flex-col gap-0 overflow-hidden rounded-t-xl p-0"
                    >
                        <div className="flex justify-center pt-3">
                            <div className="h-1 w-10 rounded-full bg-muted-foreground/25" />
                        </div>
                        <SheetHeader className="shrink-0 space-y-1.5 border-b px-6 pb-4 pr-12 pt-3 text-left">
                            <SheetTitle>{title}</SheetTitle>
                            {description && (
                                <SheetDescription>{description}</SheetDescription>
                            )}
                            {headerExtra}
                        </SheetHeader>
                        {body}
                        <SheetFooter className="shrink-0 flex-row flex-wrap items-center justify-end gap-2 border-t px-6 py-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
                            {footerStartSlot}
                            {actions}
                        </SheetFooter>
                    </SheetContent>
                </Sheet>
            ) : (
                <Dialog open={open} onOpenChange={onOpenChange}>
                    <DialogContent className={cn('flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-lg', className)}>
                        <DialogHeader className="shrink-0 space-y-1.5 border-b px-6 pb-4 pr-12 pt-6">
                            <DialogTitle>{title}</DialogTitle>
                            {description && (
                                <DialogDescription>{description}</DialogDescription>
                            )}
                            {headerExtra}
                        </DialogHeader>
                        {body}
                        <DialogFooter className="shrink-0 flex-row flex-wrap items-center border-t px-6 py-4 sm:justify-end">
                            {footerStartSlot}
                            {actions}
                        </DialogFooter>
                    </DialogContent>
                </Dialog>
            )}
        </FormDialogFooterStartContext.Provider>
    )
}
