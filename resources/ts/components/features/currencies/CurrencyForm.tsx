import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Input } from '@/components/ui/input'
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from '@/components/ui/form'
import { currencySchema, CurrencyFormData } from '@/schemas'
import { FormInputField, FormSubmitButton, FormWrapper } from '@/components/shared'
import { useFormValuesChange } from '@/hooks'
import { useCurrencySuggestions } from './use-currency-suggestions'
import type { CurrencyCatalogItem } from '@/types'
import { cn } from '@/lib/utils'

interface CurrencyFormProps {
    defaultValues?: Partial<CurrencyFormData>
    onSubmit: (data: CurrencyFormData) => void
    onValuesChange?: (data: CurrencyFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    isEditing?: boolean
    autoUpdateEnabled?: boolean
    isBase?: boolean
    formId?: string
    hideSubmit?: boolean
}

function Suggestions({
    items,
    onSelect,
}: {
    items: CurrencyCatalogItem[]
    onSelect: (item: CurrencyCatalogItem) => void
}) {
    if (items.length === 0) {
        return null
    }

    return (
        <ul
            className={cn(
                'absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-md border bg-popover',
                'p-1 text-popover-foreground shadow-md',
            )}
        >
            {items.map((item) => (
                <li key={item.code}>
                    <button
                        type="button"
                        className={cn(
                            'flex w-full items-center justify-between gap-3 rounded-sm px-2 py-1.5 text-left',
                            'text-sm hover:bg-accent hover:text-accent-foreground',
                        )}
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => onSelect(item)}
                    >
                        <span className="min-w-0 truncate">
                            <span className="font-medium">{item.code}</span>
                            <span className="text-muted-foreground"> · {item.name}</span>
                        </span>
                        <span className="shrink-0 text-muted-foreground">{item.symbol}</span>
                    </button>
                </li>
            ))}
        </ul>
    )
}

export function CurrencyForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    isEditing = false,
    autoUpdateEnabled = false,
    isBase = false,
    formId,
    hideSubmit,
}: CurrencyFormProps) {
    const { t } = useTranslation(['common', 'forms'])

    const form = useForm<CurrencyFormData>({
        resolver: schemaResolver<CurrencyFormData>(currencySchema),
        defaultValues: {
            code: '',
            name: '',
            symbol: '',
            decimals: 2,
            rate: 1,
            ...defaultValues,
        },
    })

    const { suggestField, setSuggestField, suggestions, applyCatalogItem } = useCurrencySuggestions(form, isEditing)

    useFormValuesChange(form, onValuesChange)

    const rateHelp = isBase
        ? t('forms:currencies.rateBaseHelp')
        : autoUpdateEnabled
          ? t('forms:currencies.rateAutoHelp')
          : t('forms:currencies.rateHelp')

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                    <FormField
                        control={form.control}
                        name="code"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('forms:currencies.code')}</FormLabel>
                                <div className="relative">
                                    <FormControl>
                                        <Input
                                            placeholder="USD"
                                            autoComplete="off"
                                            {...field}
                                            onChange={(e) => {
                                                field.onChange(e.target.value.toUpperCase())
                                                setSuggestField('code')
                                            }}
                                            onFocus={() => !isEditing && setSuggestField('code')}
                                            onBlur={() =>
                                                setSuggestField((current) => (current === 'code' ? null : current))
                                            }
                                        />
                                    </FormControl>
                                    {suggestField === 'code' && (
                                        <Suggestions items={suggestions} onSelect={applyCatalogItem} />
                                    )}
                                </div>
                                <FormDescription>{t('forms:currencies.codeHelp')}</FormDescription>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('fields.name')}</FormLabel>
                                <div className="relative">
                                    <FormControl>
                                        <Input
                                            placeholder={t('forms:currencies.namePlaceholder')}
                                            autoComplete="off"
                                            {...field}
                                            onChange={(e) => {
                                                field.onChange(e.target.value)
                                                setSuggestField('name')
                                            }}
                                            onFocus={() => !isEditing && setSuggestField('name')}
                                            onBlur={() =>
                                                setSuggestField((current) => (current === 'name' ? null : current))
                                            }
                                        />
                                    </FormControl>
                                    {suggestField === 'name' && (
                                        <Suggestions items={suggestions} onSelect={applyCatalogItem} />
                                    )}
                                </div>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <div className="grid grid-cols-2 items-start gap-3">
                        <FormInputField
                            control={form.control}
                            name="symbol"
                            label={t('forms:currencies.symbol')}
                            description={t('forms:currencies.symbolHelp')}
                            inputProps={{ placeholder: '$' }}
                        />
                        <FormInputField
                            control={form.control}
                            name="decimals"
                            label={t('forms:currencies.decimals')}
                            description={t('forms:currencies.decimalsHelp')}
                            inputProps={{ type: 'number', min: 0, max: 8 }}
                        />
                    </div>

                    <FormInputField
                        control={form.control}
                        name="rate"
                        label={t('forms:currencies.rate')}
                        description={rateHelp}
                        inputProps={{
                            type: 'number',
                            step: '0.000001',
                            min: 0,
                            disabled: isBase || autoUpdateEnabled,
                        }}
                    />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
        </FormWrapper>
    )
}
