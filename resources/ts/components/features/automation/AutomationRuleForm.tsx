import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Switch } from '@/components/ui/switch'
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { automationRuleSchema, type AutomationRuleFormData } from '@/schemas'
import type { Action, ConditionGroup } from '@/types/automation'
import { useAutomationTriggers } from '@/hooks/use-automation'
import { AutomationTriggerField } from './AutomationTriggerField'
import { ConditionBuilder } from './ConditionBuilder'
import { ActionBuilder } from './ActionBuilder'
import { FieldHelp, FormActiveField, FormInputField, FormSubmitButton, FormWrapper } from '@/components/shared'

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

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                    <FormInputField
                        control={form.control}
                        name="name"
                        label={t('fields.name')}
                        inputProps={{ placeholder: t('forms:automation.namePlaceholder') }}
                    />

                    <div className="grid grid-cols-2 gap-4">
                        <FormInputField
                            control={form.control}
                            name="priority"
                            className="min-w-0"
                            labelClassName="flex items-center gap-1.5"
                            label={
                                <>
                                    {t('forms:automation.priority')}
                                    <FieldHelp>{t('forms:automation.priorityHelp')}</FieldHelp>
                                </>
                            }
                            inputProps={{ type: 'number', min: 1, max: 100 }}
                        />

                        <AutomationTriggerField control={form.control} triggers={triggers} />
                    </div>

                    <FormInputField
                        control={form.control}
                        name="description"
                        label={t('fields.description')}
                        multiline
                        textareaProps={{
                            placeholder: t('forms:automation.descriptionPlaceholder'),
                            className: 'resize-none h-20',
                        }}
                    />

                    <div className="space-y-3 pt-4 border-t">
                        <h3 className="font-medium">{t('forms:automation.conditions')}</h3>
                        <FormField
                            control={form.control}
                            name="conditions"
                            render={({ field }) => (
                                <FormItem>
                                    <ConditionBuilder value={field.value as ConditionGroup} onChange={field.onChange} />
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
                                    <ActionBuilder value={field.value as Action[]} onChange={field.onChange} />
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    </div>

                    <FormActiveField control={form.control} help={t('forms:automation.activeHelp')} />

                    <div className="flex items-center gap-6 pt-2">
                        <FormField
                            control={form.control}
                            name="stop_processing"
                            render={({ field }) => (
                                <FormItem className="flex items-center gap-2 space-y-0">
                                    <FormControl>
                                        <Switch checked={field.value} onCheckedChange={field.onChange} />
                                    </FormControl>
                                    <FormLabel className="cursor-pointer">
                                        {t('forms:automation.stopProcessing')}
                                    </FormLabel>
                                </FormItem>
                            )}
                        />
                    </div>

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
        </FormWrapper>
    )
}
