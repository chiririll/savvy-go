import { useTranslation } from 'react-i18next'
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AccountFormValues } from '@/schemas'
import { FORM_ACCOUNT_TYPE_CONFIG } from '@/constants'
import { FORM_ACCOUNT_TYPES } from '@/types'
import { cn } from '@/lib/utils'
import { accountTypeLabel } from '@/lib/labels'
import { UseFormReturn } from 'react-hook-form'

export function AccountTypeField({ form }: { form: UseFormReturn<AccountFormValues> }) {
    const { t } = useTranslation(['common', 'forms'])

    return (
        <FormField
            control={form.control}
            name="type"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('fields.type')}</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value}>
                        <FormControl>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('forms:selectAccountType')} />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            {FORM_ACCOUNT_TYPES.map((type) => {
                                const config = FORM_ACCOUNT_TYPE_CONFIG[type]
                                const Icon = config.icon
                                return (
                                    <SelectItem key={type} value={type}>
                                        <div className="flex items-center gap-2">
                                            <Icon className={cn('size-4', config.textColor)} />
                                            {accountTypeLabel(t, type)}
                                        </div>
                                    </SelectItem>
                                )
                            })}
                        </SelectContent>
                    </Select>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
