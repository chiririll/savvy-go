import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { FormField } from '@/components/ui/form'
import { FieldHelp, FormDialogFooterStart } from '@/components/shared'
import type { DebtFormData } from '@/schemas'

export function DebtOriginField({ control }: { control: Control<DebtFormData> }) {
    const { t } = useTranslation('forms')

    return (
        <FormDialogFooterStart>
            <FormField
                control={control}
                name="origin"
                render={({ field }) => (
                    <label className="flex items-center gap-2 text-sm font-normal leading-none">
                        <Checkbox
                            checked={field.value === 'new'}
                            onCheckedChange={(checked) => field.onChange(checked === true ? 'new' : 'existing')}
                        />
                        <span>{t('debts.createTransaction')}</span>
                        <FieldHelp>
                            {field.value === 'existing'
                                ? t('debts.originExistingHelp')
                                : t('debts.originNewHelp')}
                        </FieldHelp>
                    </label>
                )}
            />
        </FormDialogFooterStart>
    )
}
