import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FieldHelp, FormInputField } from '@/components/shared'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { budgetPeriodLabel } from '@/lib/labels'
import type { BudgetFormData } from '@/schemas'

const PERIOD_OPTIONS = ['weekly', 'monthly', 'yearly', 'one_time'] as const

interface BudgetPeriodFieldsProps {
    control: Control<BudgetFormData>
    isOneTime: boolean
}

export function BudgetPeriodFields({ control, isOneTime }: BudgetPeriodFieldsProps) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <>
            <div className="grid grid-cols-2 gap-4">
                <FormField
                    control={control}
                    name="period"
                    render={({ field }) => (
                        <FormItem className="min-w-0">
                            <FormLabel className="flex min-h-4 items-center">{t('forms:budgets.period')}</FormLabel>
                            <Select onValueChange={field.onChange} value={field.value}>
                                <FormControl>
                                    <SelectTrigger className="w-full">
                                        <SelectValue placeholder={t('forms:selectPeriod')} />
                                    </SelectTrigger>
                                </FormControl>
                                <SelectContent>
                                    {PERIOD_OPTIONS.map((option) => (
                                        <SelectItem key={option} value={option}>
                                            {budgetPeriodLabel(t, option)}
                                        </SelectItem>
                                    ))}
                                </SelectContent>
                            </Select>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormInputField
                    control={control}
                    name="notify_at_percent"
                    className="min-w-0"
                    labelClassName="flex min-h-4 items-center gap-1.5"
                    label={(
                        <>
                            {t('forms:budgets.notifyAt')}
                            <FieldHelp>{t('forms:budgets.notifyHelp')}</FieldHelp>
                        </>
                    )}
                    nullable
                    inputProps={{
                        type: 'number',
                        min: '1',
                        max: '100',
                        placeholder: t('forms:budgets.notifyPlaceholder'),
                    }}
                />
            </div>

            {isOneTime && (
                <div className="grid grid-cols-2 gap-4">
                    <FormInputField
                        control={control}
                        name="start_date"
                        label={t('forms:budgets.startDate')}
                        nullable
                        inputProps={{ type: 'date' }}
                    />
                    <FormInputField
                        control={control}
                        name="end_date"
                        label={t('forms:budgets.endDate')}
                        nullable
                        inputProps={{ type: 'date' }}
                    />
                </div>
            )}
        </>
    )
}
