import type { Control } from 'react-hook-form'
import { FormField } from '@/components/ui/form'
import { TagSelect } from '@/components/shared'
import type { TransactionFormValues } from '@/schemas/transactions'

export function TransactionTagsField({ control }: { control: Control<TransactionFormValues> }) {
    return (
        <FormField
            control={control}
            name="tag_ids"
            render={({ field }) => <TagSelect value={field.value ?? []} onChange={field.onChange} asFormItem allowCreate />}
        />
    )
}
