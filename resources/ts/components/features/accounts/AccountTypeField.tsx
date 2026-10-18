import { useTranslation } from 'react-i18next'
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { AccountFormValues } from '@/schemas'
import { REGULAR_ACCOUNT_TYPE_CONFIG, REGULAR_ACCOUNT_TYPES } from '@/constants'
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
                            {REGULAR_ACCOUNT_TYPES.map((type) => {
                                const config = REGULAR_ACCOUNT_TYPE_CONFIG[type]
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
