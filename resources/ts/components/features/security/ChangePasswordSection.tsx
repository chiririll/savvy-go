import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useForm, type Control } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Lock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { useChangePassword } from '@/hooks'
import { useUser } from '@/stores/auth'
import { SettingsSection } from '@/components/shared'


type PasswordFormValues = {
    current_password: string
    password: string
    password_confirmation: string
}


function PasswordField({
    control,
    name,
    label,
    autoComplete,
}: {
    control: Control<PasswordFormValues>
    name: keyof PasswordFormValues
    label: string
    autoComplete: string
}) {
    return (
        <FormField
            control={control}
            name={name}
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{label}</FormLabel>
                    <FormControl>
                        <Input type="password" autoComplete={autoComplete} {...field} />
                    </FormControl>
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

export function ChangePasswordSection() {
    const { t } = useTranslation('settings')
    const user = useUser()
    const changePassword = useChangePassword()

    const schema = useMemo(() => z.object({
        current_password: z.string().min(1, t('security.password.validation.currentRequired')),
        password: z.string().min(8, t('security.password.validation.passwordMin')),
        password_confirmation: z.string(),
    }).refine((data) => data.password === data.password_confirmation, {
        message: t('security.password.validation.passwordMatch'),
        path: ['password_confirmation'],
    }), [t])

    const form = useForm<PasswordFormValues>({
        resolver: zodResolver(schema),
        defaultValues: {
            current_password: '',
            password: '',
            password_confirmation: '',
        },
    })

    const onSubmit = async (data: PasswordFormValues) => {
        try {
            await changePassword.mutateAsync(data)
            form.reset()
        } catch {
            // toasted by the mutation
        }
    }

    return (
        <SettingsSection
            icon={Lock}
            title={t('security.password.title')}
            description={t('security.password.description')}
        >
            {user?.isSsoOnly ? (
                <p className="text-sm text-muted-foreground">{t('security.password.ssoOnly')}</p>
            ) : (
                <Form {...form}>
                    <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                        <PasswordField
                            control={form.control}
                            name="current_password"
                            label={t('security.password.current')}
                            autoComplete="current-password"
                        />
                        <PasswordField
                            control={form.control}
                            name="password"
                            label={t('security.password.new')}
                            autoComplete="new-password"
                        />
                        <PasswordField
                            control={form.control}
                            name="password_confirmation"
                            label={t('security.password.confirm')}
                            autoComplete="new-password"
                        />

                        <Button type="submit" disabled={changePassword.isPending}>
                            {t('security.password.submit')}
                        </Button>
                    </form>
                </Form>
            )}
        </SettingsSection>
    )
}
