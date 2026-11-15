import { useFormContext } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { FormControl, FormDescription, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import type { TransactionFormValues } from '@/schemas/transactions'
import type { LinkedSpace } from '@/types/spaces'

interface SpaceTransferTargetFieldProps {
    links: LinkedSpace[]
    /** The chosen space has no accounts to receive the money. */
    noAccounts: boolean
}

/** The linked space a transfer_out goes to. */
export function SpaceTransferTargetField({ links, noAccounts }: SpaceTransferTargetFieldProps) {
    const { t } = useTranslation('forms')
    const form = useFormContext<TransactionFormValues>()

    return (
        <FormField
            control={form.control}
            name="to_space_id"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('transactions.toSpace')}</FormLabel>
                    <Select
                        value={field.value ? String(field.value) : undefined}
                        onValueChange={(value) => {
                            field.onChange(Number(value))
                            // Account ids are per space.
                            form.setValue('to_account_id', null)
                        }}
                    >
                        <FormControl>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('selectSpace')} />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            {links.map((link) => (
                                <SelectItem key={link.id} value={String(link.id)}>
                                    {link.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    {noAccounts && <FormDescription>{t('transactions.toSpaceNoAccounts')}</FormDescription>}
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}
