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
import { Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle } from '@/components/ui/sheet'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { useIsMobile } from '@/hooks/use-mobile'
import { cn } from '@/lib/utils'

export interface FormDialogContextValue {
    formId: string
    isSubmitting: boolean
    isEdit: boolean
    isReadOnly: boolean
    close: () => void
}

interface FormDialogInternalContext extends FormDialogContextValue {
    footerStart: HTMLElement | null
    setFooterStart: (el: HTMLDivElement | null) => void
}

const FormDialogContext = createContext<FormDialogInternalContext | null>(null)

/** State of the surrounding FormDialog, for footer overrides. */
export function useFormDialog(): FormDialogContextValue {
    const ctx = useContext(FormDialogContext)
    if (!ctx) {
        throw new Error('useFormDialog must be used inside FormDialog')
    }
    return ctx
}

/** Puts its children into the footer's left region, from anywhere inside the form. */
export function FormDialogFooterStart({ children }: { children: ReactNode }) {
    const ctx = useContext(FormDialogContext)

    // Outside a dialog the children stay where they are.
    if (!ctx) {
        return children
    }

    // The region is overridden, or not mounted yet.
    if (!ctx.footerStart) {
        return null
    }

    return createPortal(children, ctx.footerStart)
}

/** The footer row: border, padding and layout for the dialog and the mobile sheet. */
export function FormDialogFooter({ children }: { children: ReactNode }) {
    const isMobile = useIsMobile()

    if (isMobile) {
        return (
            <SheetFooter
                className={cn(
                    'shrink-0 flex-row flex-wrap items-center justify-end gap-2 border-t px-6 py-4',
                    'pb-[max(1rem,env(safe-area-inset-bottom))]',
                )}
            >
                {children}
            </SheetFooter>
        )
    }

    return (
        <DialogFooter className="shrink-0 flex-row flex-wrap items-center border-t px-6 py-4 sm:justify-end">
            {children}
        </DialogFooter>
    )
}

/** The left region, filled by FormDialogFooterStart. */
export function FormDialogFooterStartSlot() {
    const { setFooterStart } = useContext(FormDialogContext) ?? {}

    return (
        <div
            ref={setFooterStart}
            className="flex min-w-0 flex-1 flex-wrap items-center gap-x-4 gap-y-2 empty:hidden"
        />
    )
}

export function FormDialogCancel() {
    const { t } = useTranslation('common')
    const { close } = useFormDialog()

    return (
        <Button type="button" variant="outline" onClick={close}>
            {t('actions.cancel')}
        </Button>
    )
}

/** Submits the dialog's form. Not drawn for the read-only role. */
export function FormDialogSubmit() {
    const { t } = useTranslation('common')
    const { formId, isSubmitting, isEdit, isReadOnly } = useFormDialog()

    if (isReadOnly) {
        return null
    }

    return (
        <Button type="submit" form={formId} disabled={isSubmitting}>
            {isSubmitting ? t('actions.saving') : isEdit ? t('actions.save') : t('actions.create')}
        </Button>
    )
}

/** The default right region: Cancel and Save. */
export function FormDialogActions() {
    return (
        <div data-slot="form-dialog-actions" className="ml-auto flex shrink-0 items-center gap-2">
            <FormDialogCancel />
            <FormDialogSubmit />
        </div>
    )
}

type FooterOverride = ReactNode | ((ctx: FormDialogContextValue) => ReactNode)

function resolve(value: FooterOverride, ctx: FormDialogContextValue): ReactNode {
    return typeof value === 'function' ? value(ctx) : value
}

export interface FormDialogProps {
    open: boolean
    onOpenChange: (open: boolean) => void
    title: string
    description?: string
    formId: string
    isSubmitting?: boolean
    isEdit?: boolean
    className?: string
    headerExtra?: ReactNode
    /** Replaces the whole footer row. Compose it from FormDialogFooter and the default parts. */
    footer?: FooterOverride
    /** Replaces the left region only (default: the FormDialogFooterStart slot). */
    footerStart?: FooterOverride
    /** Replaces the right region only (default: Cancel and Save). */
    footerEnd?: FooterOverride
    children: ReactNode
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
    footer,
    footerStart,
    footerEnd,
    children,
}: FormDialogProps) {
    const isMobile = useIsMobile()
    const isReadOnly = useReadOnly()
    const [footerStartEl, setFooterStart] = useState<HTMLDivElement | null>(null)
    const ctx: FormDialogInternalContext = {
        formId,
        isSubmitting: !!isSubmitting,
        isEdit,
        isReadOnly,
        close: () => onOpenChange(false),
        footerStart: footerStartEl,
        setFooterStart,
    }
    const body = <div className="min-h-0 flex-1 overflow-y-auto px-6 py-4">{children}</div>
    const footerNode =
        footer !== undefined ? (
            resolve(footer, ctx)
        ) : (
            <FormDialogFooter>
                {footerStart !== undefined ? resolve(footerStart, ctx) : <FormDialogFooterStartSlot />}
                {footerEnd !== undefined ? resolve(footerEnd, ctx) : <FormDialogActions />}
            </FormDialogFooter>
        )

    return (
        <FormDialogContext.Provider value={ctx}>
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
                            {description && <SheetDescription>{description}</SheetDescription>}
                            {headerExtra}
                        </SheetHeader>
                        {body}
                        {footerNode}
                    </SheetContent>
                </Sheet>
            ) : (
                <Dialog open={open} onOpenChange={onOpenChange}>
                    <DialogContent
                        className={cn('flex max-h-[90vh] flex-col gap-0 overflow-hidden p-0 sm:max-w-lg', className)}
                    >
                        <DialogHeader className="shrink-0 space-y-1.5 border-b px-6 pb-4 pr-12 pt-6">
                            <DialogTitle>{title}</DialogTitle>
                            {description && <DialogDescription>{description}</DialogDescription>}
                            {headerExtra}
                        </DialogHeader>
                        {body}
                        {footerNode}
                    </DialogContent>
                </Dialog>
            )}
        </FormDialogContext.Provider>
    )
}
