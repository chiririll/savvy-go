// A real react-hook-form around a component that takes `control`, for the Form* fields.
import { useEffect, type ReactNode } from 'react'
import { useForm, type Control, type FieldValues } from 'react-hook-form'

import { Form } from '@/components/ui/form'

type Props<T extends FieldValues> = {
    defaultValues: T
    children: (control: Control<T>) => ReactNode
    /** Fields to mark invalid with a message, as a failed submit would. */
    errors?: Partial<Record<keyof T & string, string>>
}

export function FormHarness<T extends FieldValues>({ defaultValues, children, errors }: Props<T>) {
    const form = useForm<T>({ defaultValues: defaultValues as never })

    // As validation after a failed submit would.
    useEffect(() => {
        for (const [name, message] of Object.entries(errors ?? {})) {
            form.setError(name as never, { message: message as string })
        }
    }, [form, errors])

    return (
        <Form {...form}>
            <form onSubmit={(event) => event.preventDefault()} className="space-y-4">
                {children(form.control)}
            </form>
        </Form>
    )
}
