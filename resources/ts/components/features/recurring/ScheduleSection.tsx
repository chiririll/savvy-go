import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { frequencyLabel, weekdayLabel } from '@/lib/labels'
import type { RecurringFormData } from '@/schemas'

type FormControlOf = Control<RecurringFormData>

const frequencyOptions = ['daily', 'weekly', 'monthly', 'yearly'] as const
const weekdayValues = [0, 1, 2, 3, 4, 5, 6] as const

function FrequencyFields({ control, frequency }: { control: FormControlOf; frequency: string }) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <div className="grid grid-cols-2 gap-4">
            <FormField
                control={control}
                name="frequency"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:recurring.frequency')}</FormLabel>
                        <Select onValueChange={field.onChange} value={field.value}>
                            <FormControl>
                                <SelectTrigger className="w-full">
                                    <SelectValue />
                                </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                                {frequencyOptions.map((option) => (
                                    <SelectItem key={option} value={option}>
                                        {frequencyLabel(t, option)}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <FormMessage />
                    </FormItem>
                )}
            />

            <FormField
                control={control}
                name="interval"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:recurring.every')}</FormLabel>
                        <FormControl>
                            <Input type="number" min="1" max="365" {...field} />
                        </FormControl>
                        <FormDescription>
                            {frequency === 'daily' && t('forms:recurring.days')}
                            {frequency === 'weekly' && t('forms:recurring.weeks')}
                            {frequency === 'monthly' && t('forms:recurring.months')}
                            {frequency === 'yearly' && t('forms:recurring.years')}
                        </FormDescription>
                        <FormMessage />
                    </FormItem>
                )}
            />
        </div>
    )
}

function DayOfWeekField({ control }: { control: FormControlOf }) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={control}
            name="day_of_week"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('forms:recurring.dayOfWeek')}</FormLabel>
                    <Select onValueChange={(val) => field.onChange(Number(val))} value={field.value?.toString() ?? ''}>
                        <FormControl>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('forms:selectDay')} />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            {weekdayValues.map((value) => (
                                <SelectItem key={value} value={value.toString()}>
                                    {weekdayLabel(t, value)}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

function DayOfMonthField({ control }: { control: FormControlOf }) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={control}
            name="day_of_month"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('forms:recurring.dayOfMonth')}</FormLabel>
                    <FormControl>
                        <Input
                            type="number"
                            min="1"
                            max="31"
                            {...field}
                            value={field.value ?? ''}
                            onChange={(e) => field.onChange(e.target.value ? Number(e.target.value) : null)}
                        />
                    </FormControl>
                    <FormDescription>{t('forms:recurring.dayOfMonthHelp')}</FormDescription>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

function DateRangeFields({ control }: { control: FormControlOf }) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <div className="grid grid-cols-2 gap-4">
            <FormField
                control={control}
                name="start_date"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:recurring.startDate')}</FormLabel>
                        <FormControl>
                            <Input type="date" {...field} />
                        </FormControl>
                        <FormMessage />
                    </FormItem>
                )}
            />

            <FormField
                control={control}
                name="end_date"
                render={({ field }) => (
                    <FormItem>
                        <FormLabel>{t('forms:recurring.endDate')}</FormLabel>
                        <FormControl>
                            <Input
                                type="date"
                                {...field}
                                value={field.value ?? ''}
                                onChange={(e) => field.onChange(e.target.value || null)}
                            />
                        </FormControl>
                        <FormDescription>{t('forms:recurring.endDateHelp')}</FormDescription>
                        <FormMessage />
                    </FormItem>
                )}
            />
        </div>
    )
}

export function ScheduleSection({ control, frequency }: { control: FormControlOf; frequency: string }) {
    const { t } = useTranslation('forms')

    return (
        <div className="space-y-4 pt-4 border-t">
            <h3 className="font-medium">{t('recurring.schedule')}</h3>
            <FrequencyFields control={control} frequency={frequency} />
            {frequency === 'weekly' && <DayOfWeekField control={control} />}
            {frequency === 'monthly' && <DayOfMonthField control={control} />}
            <DateRangeFields control={control} />
        </div>
    )
}
