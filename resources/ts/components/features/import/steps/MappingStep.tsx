import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import i18n from '@/lib/i18n'
import { Form, FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { ChoiceField, ColumnField } from './MappingFields'
import { AccountSelect } from '@/components/shared/AccountSelect'
import type { CsvParseResult, ColumnMapping, ImportOptions, DateFormat, AmountFormat } from '@/types/import'

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

const DATE_FORMATS = [
    { value: 'ISO', label: 'ISO (YYYY-MM-DD)' },
    { value: 'DD.MM.YYYY', label: 'DD.MM.YYYY' },
    { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
    { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
]

const AMOUNT_FORMATS = [
    { value: 'US', label: 'US (1,234.56)' },
    { value: 'EU', label: 'EU (1.234,56)' },
]

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

    const { control } = form
    const common = { control, disabled: isLoading }
    const required = { ...common, options: columnOptions, placeholder: tForms('import.selectColumn') }
    const optional = { ...required, noneLabel: tForms('import.none') }

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(handleSubmit)} className="space-y-6" id="mapping-form">
                <div className="space-y-4">
                    <h3 className="font-medium">{t('import.requiredFields')}</h3>

                    <div className="grid gap-4 sm:grid-cols-2">
                        <ColumnField {...required} name="date" label={tForms('import.dateColumn')} />
                        <ColumnField {...required} name="amount" label={tForms('import.amountColumn')} />
                    </div>

                    <FormField
                        control={control}
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
                                <FormDescription>{tForms('import.targetAccountHelp')}</FormDescription>
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                <div className="space-y-4">
                    <h3 className="font-medium">{t('import.optionalFields')}</h3>

                    <div className="grid gap-4 sm:grid-cols-2">
                        <ColumnField {...optional} name="description" label={tForms('import.descriptionColumn')} />
                        <ColumnField
                            {...optional}
                            name="category"
                            label={tForms('import.categoryColumn')}
                            help={tForms('import.categoryHelp')}
                        />
                        <ColumnField
                            {...optional}
                            name="tags"
                            label={tForms('import.tagsColumn')}
                            help={tForms('import.tagsHelp')}
                        />
                        <ColumnField
                            {...optional}
                            name="type"
                            label={tForms('import.typeColumn')}
                            help={tForms('import.typeHelp')}
                            noneLabel={tForms('import.autoDetect')}
                        />
                    </div>
                </div>

                <div className="space-y-4">
                    <h3 className="font-medium">{t('import.formatSettings')}</h3>

                    <div className="grid gap-4 sm:grid-cols-3">
                        <ChoiceField
                            {...common}
                            name="dateFormat"
                            label={tForms('import.dateFormat')}
                            choices={DATE_FORMATS}
                        />
                        <ChoiceField
                            {...common}
                            name="amountFormat"
                            label={tForms('import.amountFormat')}
                            choices={AMOUNT_FORMATS}
                        />
                        <ChoiceField
                            {...common}
                            name="defaultType"
                            label={tForms('import.defaultType')}
                            help={tForms('import.defaultTypeHelp')}
                            choices={[
                                { value: 'expense', label: tPages('transactions.types.expense') },
                                { value: 'income', label: tPages('transactions.types.income') },
                            ]}
                        />
                    </div>
                </div>
            </form>
        </Form>
    )
}
