import { Loader2 } from 'lucide-react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { testId } from '@/lib/test-id'

import type { AdminSetupValues } from './useAdminSetup'

type TextField = {
    name: keyof AdminSetupValues
    label: string
    placeholder: string
    type?: string
    autoComplete: string
}

interface AdminFieldsStepProps {
    form: UseFormReturn<AdminSetupValues>
    isLoading: boolean
    onBack: () => void
}

export function AdminFieldsStep({ form, isLoading, onBack }: AdminFieldsStepProps) {
    const { t } = useTranslation('auth')
    const fields: TextField[] = [
        { name: 'name', label: t('setup.name'), placeholder: t('setup.namePlaceholder'), autoComplete: 'name' },
        { name: 'email', label: t('email'), placeholder: t('emailPlaceholder'), type: 'email', autoComplete: 'email' },
        {
            name: 'password',
            label: t('password'),
            placeholder: t('setup.passwordPlaceholder'),
            type: 'password',
            autoComplete: 'new-password',
        },
        {
            name: 'password_confirmation',
            label: t('setup.confirmPassword'),
            placeholder: t('setup.confirmPlaceholder'),
            type: 'password',
            autoComplete: 'new-password',
        },
    ]

    return (
        <>
            {fields.map(({ name, label, placeholder, type, autoComplete }) => (
                <FormField
                    key={name}
                    control={form.control}
                    name={name}
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{label}</FormLabel>
                            <FormControl>
                                <Input type={type} placeholder={placeholder} autoComplete={autoComplete} {...field} />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
            ))}
            <Button type="submit" className="w-full" disabled={isLoading}>
                {isLoading && <Loader2 className="mr-2 size-4 animate-spin" />}
                {t('setup.submit')}
            </Button>
            <Button
                type="button"
                variant="ghost"
                className="w-full"
                disabled={isLoading}
                onClick={onBack}
                {...testId('setup-back')}
            >
                {t('setup.back')}
            </Button>
        </>
    )
}
