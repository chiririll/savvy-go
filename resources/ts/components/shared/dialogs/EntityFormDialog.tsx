import { useMemo, type ReactNode } from 'react'
import { FormDialog, type FormDialogProps } from './FormDialog'
import { useCreateFormDraft } from '@/hooks/use-create-form-draft'

interface EntityFormFields<TForm, TValues> {
    defaultValues?: Partial<TValues>
    onValuesChange?: (values: TValues) => void
    formId: string
    onSubmit: (data: TForm) => void
    isSubmitting?: boolean
    hideSubmit: true
}

interface EntityFormDialogProps<TEntity extends { id?: string | number }, TForm, TValues = TForm> {
    entity?: TEntity | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: TForm) => void
    isSubmitting?: boolean
    formId: string
    title: string
    description?: string
    className?: string
    headerExtra?: ReactNode
    footer?: FormDialogProps['footer']
    footerStart?: FormDialogProps['footerStart']
    footerEnd?: FormDialogProps['footerEnd']
    fallbackValues?: Partial<TValues>
    toFormValues: (entity: TEntity) => Partial<TValues>
    children: (props: {
        formKey: string | number
        isEdit: boolean
        formProps: EntityFormFields<TForm, TValues>
    }) => ReactNode
}

export function EntityFormDialog<TEntity extends { id?: string | number }, TForm, TValues = TForm>({
    entity,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
    formId,
    title,
    description,
    className,
    headerExtra,
    footer,
    footerStart,
    footerEnd,
    fallbackValues,
    toFormValues,
    children,
}: EntityFormDialogProps<TEntity, TForm, TValues>) {
    const isEdit = !!entity
    const { draft, onValuesChange, formKey } = useCreateFormDraft<TValues>({
        enabled: !isEdit,
        open,
        isSubmitting,
        entityKey: entity?.id,
    })
    const defaultValues = useMemo(
        // Values the opener asks for (a button preselecting an account, a type) beat the stale draft.
        () => (entity ? toFormValues(entity) : draft || fallbackValues ? { ...draft, ...fallbackValues } : undefined),
        [entity, toFormValues, draft, fallbackValues],
    )

    return (
        <FormDialog
            open={open}
            onOpenChange={onOpenChange}
            title={title}
            description={description}
            formId={formId}
            isSubmitting={isSubmitting}
            isEdit={isEdit}
            className={className}
            headerExtra={headerExtra}
            footer={footer}
            footerStart={footerStart}
            footerEnd={footerEnd}
        >
            {children({
                formKey,
                isEdit,
                formProps: {
                    defaultValues,
                    onValuesChange,
                    formId,
                    onSubmit,
                    isSubmitting,
                    hideSubmit: true,
                },
            })}
        </FormDialog>
    )
}
