import type { Control, Path } from 'react-hook-form'
import { FormField } from '@/components/ui/form'
import { TagSelect } from '@/components/shared'

export function TransactionTagsField<T extends { tag_ids?: number[] }>({ control }: { control: Control<T> }) {
    return (
        <FormField
            control={control}
            name={'tag_ids' as Path<T>}
            render={({ field }) => (
                <TagSelect value={(field.value as number[] | undefined) ?? []} onChange={field.onChange} asFormItem allowCreate />
            )}
        />
    )
}
