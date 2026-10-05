import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
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
import { currencySchema, CurrencyFormData } from '@/schemas'
import { FormWrapper } from '@/components/shared/FormWrapper'
import { useCurrencyCatalog, useFormValuesChange } from '@/hooks'
import type { CurrencyCatalogItem } from '@/types'

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

function filterCatalog(catalog: CurrencyCatalogItem[], query: string, field: 'code' | 'name') {
    const q = query.trim().toLowerCase()
    if (!q) {
        return []
    }

    return catalog
        .filter((item) =>
            field === 'code'
                ? item.code.toLowerCase().startsWith(q) || item.name.toLowerCase().includes(q)
                : item.name.toLowerCase().includes(q) || item.code.toLowerCase().startsWith(q)
        )
        .slice(0, 8)
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
        <ul className="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-md border bg-popover p-1 text-popover-foreground shadow-md">
            {items.map((item) => (
                <li key={item.code}>
                    <button
                        type="button"
                        className="flex w-full items-center justify-between gap-3 rounded-sm px-2 py-1.5 text-left text-sm hover:bg-accent hover:text-accent-foreground"
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
    const [suggestField, setSuggestField] = useState<'code' | 'name' | null>(null)
    const { data: catalog = [] } = useCurrencyCatalog(!isEditing)

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

    const code = form.watch('code')
    const name = form.watch('name')

    const suggestions = useMemo(() => {
        if (isEditing || !suggestField) {
            return []
        }

        return filterCatalog(catalog, suggestField === 'code' ? code : name, suggestField)
    }, [catalog, code, isEditing, name, suggestField])

    const applyCatalogItem = (item: CurrencyCatalogItem) => {
        form.setValue('code', item.code, { shouldValidate: true, shouldDirty: true })
        form.setValue('name', item.name, { shouldValidate: true, shouldDirty: true })
        form.setValue('symbol', item.symbol, { shouldValidate: true, shouldDirty: true })
        form.setValue('decimals', item.decimals, { shouldValidate: true, shouldDirty: true })
        if (item.rate) {
            form.setValue('rate', item.rate, { shouldValidate: true, shouldDirty: true })
        }
        setSuggestField(null)
    }

    useFormValuesChange(form, onValuesChange)

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
                            <FormControl>
                                <div className="relative">
                                    <Input
                                        placeholder="USD"
                                        autoComplete="off"
                                        {...field}
                                        onChange={(e) => {
                                            field.onChange(e.target.value.toUpperCase())
                                            setSuggestField('code')
                                        }}
                                        onFocus={() => !isEditing && setSuggestField('code')}
                                        onBlur={() => setSuggestField((current) => current === 'code' ? null : current)}
                                    />
                                    {suggestField === 'code' && (
                                        <Suggestions items={suggestions} onSelect={applyCatalogItem} />
                                    )}
                                </div>
                            </FormControl>
                            <FormDescription>
                                {t('forms:currencies.codeHelp')}
                            </FormDescription>
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
                            <FormControl>
                                <div className="relative">
                                    <Input
                                        placeholder={t('forms:currencies.namePlaceholder')}
                                        autoComplete="off"
                                        {...field}
                                        onChange={(e) => {
                                            field.onChange(e.target.value)
                                            setSuggestField('name')
                                        }}
                                        onFocus={() => !isEditing && setSuggestField('name')}
                                        onBlur={() => setSuggestField((current) => current === 'name' ? null : current)}
                                    />
                                    {suggestField === 'name' && (
                                        <Suggestions items={suggestions} onSelect={applyCatalogItem} />
                                    )}
                                </div>
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="grid grid-cols-2 items-start gap-3">
                    <FormField
                        control={form.control}
                        name="symbol"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('forms:currencies.symbol')}</FormLabel>
                                <FormControl>
                                    <Input placeholder="$" {...field} />
                                </FormControl>
                                <FormDescription>
                                    {t('forms:currencies.symbolHelp')}
                                </FormDescription>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={form.control}
                        name="decimals"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('forms:currencies.decimals')}</FormLabel>
                                <FormControl>
                                    <Input type="number" min={0} max={8} {...field} />
                                </FormControl>
                                <FormDescription>
                                    {t('forms:currencies.decimalsHelp')}
                                </FormDescription>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                <FormField
                    control={form.control}
                    name="rate"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('forms:currencies.rate')}</FormLabel>
                            <FormControl>
                                <Input
                                    type="number"
                                    step="0.000001"
                                    min={0}
                                    disabled={isBase || autoUpdateEnabled}
                                    {...field}
                                />
                            </FormControl>
                            <FormDescription>
                                {isBase
                                    ? t('forms:currencies.rateBaseHelp')
                                    : autoUpdateEnabled
                                        ? t('forms:currencies.rateAutoHelp')
                                        : t('forms:currencies.rateHelp')}
                            </FormDescription>
                            <FormMessage />
                        </FormItem>
                    )}
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
