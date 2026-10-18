import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

export type SetPasswordValues = {
    password: string
    password_confirmation: string
}

interface SetPasswordFormProps {
    submitting: boolean
    onSubmit: (values: SetPasswordValues) => void
}

export function SetPasswordForm({ submitting, onSubmit }: SetPasswordFormProps) {
    const { t } = useTranslation('auth')

    const schema = useMemo(() => z.object({
        password: z.string().min(8, t('setPassword.validation.passwordMin')),
        password_confirmation: z.string(),
    }).refine((data) => data.password === data.password_confirmation, {
        message: t('setPassword.validation.passwordMatch'),
        path: ['password_confirmation'],
    }), [t])

    const form = useForm<SetPasswordValues>({
        resolver: zodResolver(schema),
        defaultValues: { password: '', password_confirmation: '' },
    })

    return (
        <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                    control={form.control}
                    name="password"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('password')}</FormLabel>
                            <FormControl>
                                <Input
                                    type="password"
                                    placeholder={t('setPassword.passwordPlaceholder')}
                                    autoComplete="new-password"
                                    {...field}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <FormField
                    control={form.control}
                    name="password_confirmation"
                    render={({ field }) => (
                        <FormItem>
                            <FormLabel>{t('setup.confirmPassword')}</FormLabel>
                            <FormControl>
                                <Input
                                    type="password"
                                    placeholder={t('setup.confirmPlaceholder')}
                                    autoComplete="new-password"
                                    {...field}
                                />
                            </FormControl>
                            <FormMessage />
                        </FormItem>
                    )}
                />
                <Button type="submit" className="w-full" disabled={submitting}>
                    {submitting && <Loader2 className="mr-2 size-4 animate-spin" />}
                    {t('setPassword.submit')}
                </Button>
            </form>
        </Form>
    )
}
