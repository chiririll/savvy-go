import type { Control } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { Checkbox } from '@/components/ui/checkbox'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { localizeDefaultName } from '@/lib/localized-name'
import type { BudgetFormData } from '@/schemas'
import type { Category } from '@/types'

interface BudgetCategoriesFieldProps {
    control: Control<BudgetFormData>
    categories: Category[]
}

export function BudgetCategoriesField({ control, categories }: BudgetCategoriesFieldProps) {
    const { t } = useTranslation('forms')

    return (
        <FormField
            control={control}
            name="category_ids"
            render={({ field }) => {
                const selected = field.value ?? []
                return (
                    <FormItem>
                        <FormLabel>{t('budgets.categories')}</FormLabel>
                        <div className="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto rounded-md border p-3">
                            {categories.map((category) => (
                                <label
                                    key={category.id}
                                    className="flex items-center gap-2 text-sm font-normal cursor-pointer"
                                >
                                    <FormControl>
                                        <Checkbox
                                            checked={selected.includes(category.id)}
                                            onCheckedChange={(checked) => field.onChange(
                                                checked
                                                    ? [...selected, category.id]
                                                    : selected.filter((id) => id !== category.id),
                                            )}
                                        />
                                    </FormControl>
                                    <span
                                        className="w-5 h-5 rounded flex items-center justify-center text-xs text-white"
                                        style={{ backgroundColor: category.color ?? undefined }}
                                    >
                                        {category.icon}
                                    </span>
                                    {localizeDefaultName(category.name)}
                                </label>
                            ))}
                        </div>
                        <FormMessage />
                    </FormItem>
                )
            }}
        />
    )
}
