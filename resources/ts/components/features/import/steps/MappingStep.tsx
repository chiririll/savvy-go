import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import i18n from '@/lib/i18n'
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
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
import { AccountSelect } from '@/components/shared/AccountSelect'
import type { CsvParseResult, ColumnMapping, ImportOptions, DateFormat, AmountFormat } from '@/types/import'
import { useEffect } from 'react'

const mappingSchema = z.object({
    date: z.number({ error: i18n.t('import.dateRequired', { ns: 'forms' }) }),
    amount: z.number({ error: i18n.t('import.amountRequired', { ns: 'forms' }) }),
    description: z.number().nullable(),
    type: z.number().nullable(),
    category: z.number().nullable(),
    tags: z.number().nullable(),
    currency: z.number().nullable(),
    dateFormat: z.string(),
    amountFormat: z.string(),
    defaultAccountId: z.number({ error: i18n.t('import.accountRequired', { ns: 'forms' }) }),
    defaultType: z.enum(['income', 'expense']),
})

type MappingFormValues = z.infer<typeof mappingSchema>

interface MappingStepProps {
    parseResult: CsvParseResult
    onSubmit: (mapping: ColumnMapping, options: ImportOptions) => void
    isLoading: boolean
}

const NONE_VALUE = '__none__'

export function MappingStep({ parseResult, onSubmit, isLoading }: MappingStepProps) {
    const { t } = useTranslation('settings')
    const { t: tForms } = useTranslation('forms')
    const { t: tPages } = useTranslation('pages')
    const form = useForm<MappingFormValues>({
        resolver: zodResolver(mappingSchema),
        defaultValues: {
            date: parseResult.suggestedMapping.date ?? undefined,
            amount: parseResult.suggestedMapping.amount ?? undefined,
            description: parseResult.suggestedMapping.description ?? null,
            type: parseResult.suggestedMapping.type ?? null,
            category: parseResult.suggestedMapping.category ?? null,
            tags: parseResult.suggestedMapping.tags ?? null,
            currency: parseResult.suggestedMapping.currency ?? null,
            dateFormat: parseResult.detectedFormats.dateFormat,
            amountFormat: parseResult.detectedFormats.amountFormat,
            defaultAccountId: undefined,
            defaultType: 'expense',
        },
    })

    // Auto-submit when form is valid and user changes values
    useEffect(() => {
        const subscription = form.watch(() => {
            // Don't auto-submit, just track changes
        })
        return () => subscription.unsubscribe()
    }, [form])

    const handleSubmit = (values: MappingFormValues) => {
        const mapping: ColumnMapping = {
            date: values.date,
            amount: values.amount,
            description: values.description,
            type: values.type,
            category: values.category,
            tags: values.tags,
            currency: values.currency,
        }

        const options: ImportOptions = {
            dateFormat: values.dateFormat as DateFormat,
            amountFormat: values.amountFormat as AmountFormat,
            defaultAccountId: values.defaultAccountId,
            defaultType: values.defaultType,
            skipFirstRow: parseResult.detectedFormats.hasHeader,
            createMissingCurrencies: true,
            createMissingTags: true,
            createMissingCategories: true,
        }

        onSubmit(mapping, options)
    }

    const columnOptions = parseResult.headers.map((header, index) => ({
        value: index.toString(),
        label: `${index + 1}: ${header}`,
    }))

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6" id="mapping-form">
                {/* Required Mappings */}
                <div className="space-y-4">
                    <h3 className="font-medium">{t('import.requiredFields')}</h3>

                    <div className="grid gap-4 sm:grid-cols-2">
                        <FormField
                            control={form.control}
                            name="date"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.dateColumn')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(Number(val))}
                                        value={field.value?.toString() ?? ''}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={tForms('import.selectColumn')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {columnOptions.map((opt) => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="amount"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.amountColumn')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(Number(val))}
                                        value={field.value?.toString() ?? ''}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={tForms('import.selectColumn')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {columnOptions.map((opt) => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>

                    <FormField
                        control={form.control}
                        name="defaultAccountId"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{tForms('import.targetAccount')}</FormLabel>
                                <AccountSelect
                                    value={field.value}
                                    onChange={field.onChange}
                                    placeholder={tForms('import.targetAccountPlaceholder')}
                                    disabled={isLoading}
                                    allowCreate
                                />
                                <FormDescription>
                                    {tForms('import.targetAccountHelp')}
                                </FormDescription>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                {/* Optional Mappings */}
                <div className="space-y-4">
                    <h3 className="font-medium">{t('import.optionalFields')}</h3>

                    <div className="grid gap-4 sm:grid-cols-2">
                        <FormField
                            control={form.control}
                            name="description"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.descriptionColumn')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(val === NONE_VALUE ? null : Number(val))}
                                        value={field.value?.toString() ?? NONE_VALUE}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={tForms('import.selectColumn')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value={NONE_VALUE}>{tForms('import.none')}</SelectItem>
                                            {columnOptions.map((opt) => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="category"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.categoryColumn')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(val === NONE_VALUE ? null : Number(val))}
                                        value={field.value?.toString() ?? NONE_VALUE}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={tForms('import.selectColumn')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value={NONE_VALUE}>{tForms('import.none')}</SelectItem>
                                            {columnOptions.map((opt) => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormDescription>
                                        {tForms('import.categoryHelp')}
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="tags"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.tagsColumn')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(val === NONE_VALUE ? null : Number(val))}
                                        value={field.value?.toString() ?? NONE_VALUE}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={tForms('import.selectColumn')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value={NONE_VALUE}>{tForms('import.none')}</SelectItem>
                                            {columnOptions.map((opt) => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormDescription>
                                        {tForms('import.tagsHelp')}
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="type"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.typeColumn')}</FormLabel>
                                    <Select
                                        onValueChange={(val) => field.onChange(val === NONE_VALUE ? null : Number(val))}
                                        value={field.value?.toString() ?? NONE_VALUE}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={tForms('import.selectColumn')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value={NONE_VALUE}>{tForms('import.autoDetect')}</SelectItem>
                                            {columnOptions.map((opt) => (
                                                <SelectItem key={opt.value} value={opt.value}>
                                                    {opt.label}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormDescription>
                                        {tForms('import.typeHelp')}
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                </div>

                {/* Format Options */}
                <div className="space-y-4">
                    <h3 className="font-medium">{t('import.formatSettings')}</h3>

                    <div className="grid gap-4 sm:grid-cols-3">
                        <FormField
                            control={form.control}
                            name="dateFormat"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.dateFormat')}</FormLabel>
                                    <Select
                                        onValueChange={field.onChange}
                                        value={field.value}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value="ISO">ISO (YYYY-MM-DD)</SelectItem>
                                            <SelectItem value="DD.MM.YYYY">DD.MM.YYYY</SelectItem>
                                            <SelectItem value="DD/MM/YYYY">DD/MM/YYYY</SelectItem>
                                            <SelectItem value="MM/DD/YYYY">MM/DD/YYYY</SelectItem>
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="amountFormat"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.amountFormat')}</FormLabel>
                                    <Select
                                        onValueChange={field.onChange}
                                        value={field.value}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value="US">US (1,234.56)</SelectItem>
                                            <SelectItem value="EU">EU (1.234,56)</SelectItem>
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />

                        <FormField
                            control={form.control}
                            name="defaultType"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{tForms('import.defaultType')}</FormLabel>
                                    <Select
                                        onValueChange={field.onChange}
                                        value={field.value}
                                        disabled={isLoading}
                                    >
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            <SelectItem value="expense">{tPages('transactions.types.expense')}</SelectItem>
                                            <SelectItem value="income">{tPages('transactions.types.income')}</SelectItem>
                                        </SelectContent>
                                    </Select>
                                    <FormDescription>
                                        {tForms('import.defaultTypeHelp')}
                                    </FormDescription>
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>
                </div>
            </form>
        </Form>
    )
}
