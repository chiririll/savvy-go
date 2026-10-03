import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
    FormDescription,
} from '@/components/ui/form'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    accountSchema,
    AccountFormValues,
    AccountFormData,
    decodeAccountCurrency,
    encodeAccountCurrency,
} from '@/schemas'
import { REGULAR_ACCOUNT_TYPE_CONFIG, REGULAR_ACCOUNT_TYPES } from '@/constants'
import { cn } from '@/lib/utils'
import { CurrencySelect, type CurrencySelectValue } from '@/components/shared/CurrencySelect'
import { FormActiveField } from '@/components/shared/FormActiveField'
import { FormWrapper } from '@/components/shared/FormWrapper'

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

function toSelectValue(currency: string): CurrencySelectValue | null {
    const decoded = decodeAccountCurrency(currency)
    if (decoded.currency_code) {
        return { source: 'catalog', code: decoded.currency_code }
    }
    if (decoded.currency_id) {
        return { source: 'existing', id: decoded.currency_id }
    }
    return null
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

    const currency = form.watch('currency')

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

                <FormField
                    control={form.control}
                    name="type"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.type')}</FormLabel>
                            <Select onValueChange={field.onChange} value={field.value}>
                                <FormControl>
                                    <SelectTrigger>
                                        <SelectValue placeholder={t('forms:selectAccountType')} />
                                    </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                    {REGULAR_ACCOUNT_TYPES.map((type) => {
                                        const config = REGULAR_ACCOUNT_TYPE_CONFIG[type]
                                        const Icon = config.icon
                                        return (
                                            <SelectItem key={type} value={type}>
                                                <div className="flex items-center gap-2">
                                                    <Icon className={cn('size-4', config.textColor)} />
                                                    {t(`pages:accounts.types.${type}`)}
                                                </div>
                                            </SelectItem>
                                        )
                                    })}
                                </SelectContent>
                            </Select>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="currency"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.currency')}</FormLabel>
                            <CurrencySelect
                                value={toSelectValue(field.value || currency)}
                                onChange={(next) => {
                                    field.onChange(encodeAccountCurrency(
                                        next.source === 'existing'
                                            ? { id: next.id }
                                            : { code: next.code }
                                    ))
                                }}
                                placeholder={t('forms:selectCurrency')}
                                disabled={isEdit}
                            />
                            <FormDescription>
                                {isEdit
                                    ? t('forms:accounts.currencyLockedHelp')
                                    : t('forms:accounts.currencyCatalogHelp')}
                            </FormDescription>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormField
                    control={form.control}
                    name="initial_balance"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('forms:accounts.initialBalance')}</FormLabel>
                            <FormControl>
                                <Input
                                    type="number"
                                    step="0.01"
                                    min={0}
                                    placeholder="0.00"
                                    {...field}
                                />
                            </FormControl>
                            <FormDescription>
                                {t('forms:accounts.initialBalanceHelp')}
                            </FormDescription>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <FormActiveField
                    control={form.control}
                    help={t('forms:accounts.activeHelp')}
                />

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
