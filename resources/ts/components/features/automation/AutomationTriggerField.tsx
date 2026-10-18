import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { FieldHelp } from '@/components/shared'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { triggerDescription, triggerLabel } from '@/lib/labels'
import type { AutomationRuleFormData } from '@/schemas'

interface AutomationTriggerFieldProps {
    control: Control<AutomationRuleFormData>
    triggers?: AutomationRuleFormData['trigger_type'][]
}

export function AutomationTriggerField({ control, triggers }: AutomationTriggerFieldProps) {
    const { t } = useTranslation('forms')

    return (
        <FormField
            control={control}
            name="trigger_type"
            render={({ field }) => {
                const selectedTrigger = triggers?.find((tr) => tr === field.value)
                return (
                    <FormItem className="min-w-0">
                        <FormLabel className="flex items-center gap-1.5">
                            {t('automation.trigger')}
                            {selectedTrigger && <FieldHelp>{triggerDescription(t, selectedTrigger)}</FieldHelp>}
                        </FormLabel>
                        <Select onValueChange={field.onChange} value={field.value}>
                            <FormControl>
                                <SelectTrigger className="w-full">
                                    <SelectValue placeholder={t('automation.selectTrigger')} />
                                </SelectTrigger>
                            </FormControl>
                            <SelectContent>
                                {triggers?.map((trigger) => (
                                    <SelectItem key={trigger} value={trigger}>
                                        {triggerLabel(t, trigger)}
                                    </SelectItem>
                                ))}
                            </SelectContent>
                        </Select>
                        <FormMessage />
                    </FormItem>
                )
            }}
        />
    )
}
