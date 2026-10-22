// For components built on the ShadCN form parts (FormControl and the like), which throw outside a form.
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import type { Decorator } from '@storybook/react-vite'

import { Form, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

type Options = {
    /** The label that names the field; most controls get their accessible name from it. */
    label?: string
    /** An error to show under the field, as validation would. */
    error?: string
}

export const withForm =
    ({ label, error }: Options = {}): Decorator =>
    (Story) => {
        const form = useForm({ defaultValues: { field: '' } })

        useEffect(() => {
            if (error) form.setError('field', { message: error })
        }, [form])

        return (
            <Form {...form}>
                <FormField
                    control={form.control}
                    name="field"
                    render={() => (
                        <FormItem>
                            {label && <FormLabel>{label}</FormLabel>}
                            <Story />
                            <FormMessage />
                        </FormItem>
                    )}
                />
            </Form>
        )
    }
