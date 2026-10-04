import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
} from '@/components/ui/form'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import { automationRuleSchema, type AutomationRuleFormData } from '@/schemas'
import type { Action, ConditionGroup } from '@/types/automation'
import { useAutomationTriggers } from '@/hooks/use-automation'
import { ConditionBuilder } from './ConditionBuilder'
import { ActionBuilder } from './ActionBuilder'
import { FieldHelp, FormActiveField, FormWrapper } from '@/components/shared'

interface AutomationRuleFormProps {
    defaultValues?: Partial<AutomationRuleFormData>
    onSubmit: (data: AutomationRuleFormData) => void
    onValuesChange?: (data: AutomationRuleFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    formId?: string
    hideSubmit?: boolean
}

export function AutomationRuleForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    formId,
    hideSubmit,
}: AutomationRuleFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const { data: triggers } = useAutomationTriggers()

    const form = useForm<AutomationRuleFormData>({
        resolver: schemaResolver<AutomationRuleFormData>(automationRuleSchema),
        defaultValues: {
            name: '',
            description: null,
            trigger_type: 'on_transaction_create',
            priority: 50,
            conditions: { match: 'all', conditions: [] },
            actions: [],
            is_active: true,
            stop_processing: false,
            ...defaultValues,
        },
    })

    useFormValuesChange(form, onValuesChange)

    const handleSubmit = (data: AutomationRuleFormData) => {
        onSubmit(data)
    }

    return (
        <FormWrapper>
        <Form {...form}>
            <form id={formId} onSubmit={form.handleSubmit(handleSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.name')}</FormLabel>
                            <FormControl>
                                <Input placeholder={t('forms:automation.namePlaceholder')} {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="grid grid-cols-2 gap-4">
                    <FormField
                        control={form.control}
                        name="priority"
                        render={({ field }) => (
                            <FormItem className="min-w-0">
                                <FormLabel className="flex items-center gap-1.5">
                                    {t('forms:automation.priority')}
                                    <FieldHelp>{t('forms:automation.priorityHelp')}</FieldHelp>
                                </FormLabel>
                                <FormControl>
                                    <Input type="number" min={1} max={100} {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={form.control}
                        name="trigger_type"
                        render={({ field }) => {
                            const selectedTrigger = triggers?.find(tr => tr === field.value)
                            return (
                                <FormItem className="min-w-0">
                                    <FormLabel className="flex items-center gap-1.5">
                                        {t('forms:automation.trigger')}
                                        {selectedTrigger && (
                                            <FieldHelp>
                                                {t(`forms:automation.triggerDescriptions.${selectedTrigger}`)}
                                            </FieldHelp>
                                        )}
                                    </FormLabel>
                                    <Select onValueChange={field.onChange} value={field.value}>
                                        <FormControl>
                                            <SelectTrigger className="w-full">
                                                <SelectValue placeholder={t('forms:automation.selectTrigger')} />
                                            </SelectTrigger>
                                        </FormControl>
                                        <SelectContent>
                                            {triggers?.map((trigger) => (
                                                <SelectItem key={trigger} value={trigger}>
                                                    {t(`forms:automation.triggers.${trigger}`)}
                                                </SelectItem>
                                            ))}
                                        </SelectContent>
                                    </Select>
                                    <FormMessage />
                                </FormItem>
                            )
                        }}
                    />
                </div>

                <FormField
                    control={form.control}
                    name="description"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('fields.description')}</FormLabel>
                            <FormControl>
                                <Textarea
                                    placeholder={t('forms:automation.descriptionPlaceholder')}
                                    className="resize-none h-20"
                                    {...field}
                                    value={field.value ?? ''}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />

                <div className="space-y-3 pt-4 border-t">
                    <h3 className="font-medium">{t('forms:automation.conditions')}</h3>
                    <FormField
                        control={form.control}
                        name="conditions"
                        render={({ field }) => (
                            <FormItem>
                                <ConditionBuilder
                                    value={field.value as ConditionGroup}
                                    onChange={field.onChange}
                                />
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                <div className="space-y-3 pt-4 border-t">
                    <h3 className="font-medium">{t('forms:automation.actions')}</h3>
                    <FormField
                        control={form.control}
                        name="actions"
                        render={({ field }) => (
                            <FormItem>
                                <ActionBuilder
                                    value={field.value as Action[]}
                                    onChange={field.onChange}
                                />
                                <FormMessage />
                            </FormItem>
                        )}
                    />
                </div>

                <FormActiveField
                    control={form.control}
                    help={t('forms:automation.activeHelp')}
                />

                <div className="flex items-center gap-6 pt-2">
                    <FormField
                        control={form.control}
                        name="stop_processing"
                        render={({ field }) => (
                            <FormItem className="flex items-center gap-2 space-y-0">
                                <FormControl>
                                    <Switch
                                        checked={field.value}
                                        onCheckedChange={field.onChange}
                                    />
                                </FormControl>
                                <FormLabel className="cursor-pointer">{t('forms:automation.stopProcessing')}</FormLabel>
                            </FormItem>
                        )}
                    />
                </div>

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
