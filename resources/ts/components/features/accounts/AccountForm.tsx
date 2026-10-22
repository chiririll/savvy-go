import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from '@/components/ui/form'
import { FormActiveField } from '@/components/shared/form/FormActiveField'
import { accountSchema, AccountFormValues, AccountFormData, decodeAccountCurrency } from '@/schemas'
import { FormWrapper } from '@/components/shared/form/FormWrapper'
import { AccountCurrencyField } from './AccountCurrencyField'
import { AccountTypeField } from './AccountTypeField'

interface AccountFormProps {
    defaultValues?: Partial<AccountFormValues>
    onSubmit: (data: AccountFormData) => void
    onValuesChange?: (data: AccountFormValues) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
    isEdit?: boolean
}

function toPayload(data: AccountFormValues): AccountFormData {
    const { currency, ...rest } = data
    return { ...rest, ...decodeAccountCurrency(currency) }
}

export function AccountForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
    isEdit,
}: AccountFormProps) {
    const { t } = useTranslation(['common', 'forms', 'pages'])

    const form = useForm<AccountFormValues>({
        resolver: schemaResolver<AccountFormValues>(accountSchema),
        defaultValues: {
            name: '',
            type: 'bank',
            currency: '',
            initial_balance: 0,
            is_active: true,
            ...defaultValues,
        },
    })

    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
            <Form {...form}>
                <form
                    id={formId}
                    onSubmit={form.handleSubmit((data) => onSubmit(toPayload(data)))}
                    className="space-y-4"
                >
                    <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('fields.name')}</FormLabel>
                                <FormControl>
                                    <Input placeholder={t('forms:namePlaceholder')} {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <AccountCurrencyField form={form} isEdit={isEdit} />

                    <div className="grid grid-cols-2 items-start gap-3">
                        <AccountTypeField form={form} />
                        <FormField
                            control={form.control}
                            name="initial_balance"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{t('forms:accounts.initialBalance')}</FormLabel>
                                    <FormControl>
                                        <Input type="number" step="0.01" min={0} placeholder="0.00" {...field} />
                                    </FormControl>
                                    <FormDescription>{t('forms:accounts.initialBalanceHelp')}</FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>

                    <FormActiveField control={form.control} help={t('forms:accounts.activeHelp')} />

                    {!hideSubmit && (
                        <Button type="submit" disabled={isSubmitting} className="w-full">
                            {isSubmitting ? t('actions.saving') : (submitLabel ?? t('actions.save'))}
                        </Button>
                    )}
                </form>
            </Form>
        </FormWrapper>
    )
}
