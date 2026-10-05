import { useTranslation } from 'react-i18next'
import { FeedList, Page, PageHeader } from '@/components/shared'
import { AutomationRuleFormDialog, AutomationRuleRow } from '@/components/features/automation'
import {
    useAutomationRules,
    useCreateAutomationRule,
    useDeleteAutomationRule,
    useToggleAutomationRule,
    useUpdateAutomationRule,
    useResourceFormDialog,
} from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import type { AutomationRule } from '@/types/automation'
import type { AutomationRuleFormData } from '@/schemas'

export default function AutomationPage() {
    const { t } = useTranslation('pages')
    const { data: rules, isLoading } = useAutomationRules()
    const deleteRule = useDeleteAutomationRule()
    const toggleRule = useToggleAutomationRule()
    const createRule = useCreateAutomationRule()
    const updateRule = useUpdateAutomationRule()
    const isReadOnly = useReadOnly()
    const items = rules ?? []
    const form = useResourceFormDialog<AutomationRule, AutomationRuleFormData>({
        items,
        isLoading,
        create: createRule,
        update: updateRule,
    })

    return (
        <Page title={t('automation.title')}>
            <PageHeader
                title={t('automation.title')}
                description={t('automation.description')}
                createLabel={t('automation.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
            />

            <div className="mx-auto w-full max-w-[800px]">
                <FeedList
                    items={items}
                    isLoading={isLoading}
                    emptyTitle={t('table.emptyTitle', { ns: 'common' })}
                    emptyDescription={t('table.emptyDescription', { ns: 'common' })}
                    onCreate={form.openCreate}
                    createLabel={t('automation.create')}
                    isReadOnly={isReadOnly}
                    getKey={(rule) => rule.id}
                >
                    {(rule) => (
                        <AutomationRuleRow
                            rule={rule}
                            onEdit={form.openEdit}
                            onDelete={(id) => deleteRule.mutate(id)}
                            onToggle={(id) => toggleRule.mutate(id)}
                            isReadOnly={isReadOnly}
                        />
                    )}
                </FeedList>
            </div>

            <AutomationRuleFormDialog
                rule={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
            />
        </Page>
    )
}
