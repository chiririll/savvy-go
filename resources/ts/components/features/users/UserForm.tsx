import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm, type UseFormReturn } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Form, FormField, FormItem, FormLabel, FormControl, FormMessage, FormDescription } from '@/components/ui/form'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { createUserSchema, updateUserSchema, CreateUserFormData, UpdateUserFormData } from '@/schemas/users'
import { useUser } from '@/stores/auth'
import {
    EmailLanguageSelect,
    FormCheckboxCard,
    FormInputField,
    FormSubmitButton,
    FormWrapper,
    useUiLocale,
} from '@/components/shared'
import type { AppLocale } from '@/lib/i18n'

interface UserFormProps {
    defaultValues?: Partial<CreateUserFormData> & Partial<UpdateUserFormData> & { id?: number }
    onSubmit: (data: CreateUserFormData | UpdateUserFormData) => void
    onValuesChange?: (data: CreateUserFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    isEdit?: boolean
    formId?: string
    hideSubmit?: boolean
    /** Whether the set-password link can be emailed, and whether it must be (verification is required). */
    mail?: { canEmail: boolean; mustEmail: boolean }
}

function UserRoleField({ form, disabled }: { form: UseFormReturn<CreateUserFormData>; disabled: boolean }) {
    const { t } = useTranslation(['common', 'forms'])
    return (
        <FormField
            control={form.control}
            name="role"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('fields.role')}</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange} disabled={disabled}>
                        <FormControl>
                            <SelectTrigger className="w-full">
                                <SelectValue placeholder={t('forms:selectRole')} />
                            </SelectTrigger>
                        </FormControl>
                        <SelectContent>
                            <SelectItem value="admin">{t('roles.admin')}</SelectItem>
                            <SelectItem value="user">{t('roles.user')}</SelectItem>
                            <SelectItem value="guest">{t('roles.guest')}</SelectItem>
                        </SelectContent>
                    </Select>
                    {disabled && <FormDescription>{t('forms:users.cannotChangeOwnRole')}</FormDescription>}
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

/** The language of the email with the set-password link; starts as the language of the interface. */
function UserLocaleField({ form }: { form: UseFormReturn<CreateUserFormData> }) {
    const { t } = useTranslation('common')
    return (
        <FormField
            control={form.control}
            name="locale"
            render={({ field }) => (
                <FormItem>
                    <FormLabel>{t('language.email')}</FormLabel>
                    <EmailLanguageSelect
                        className="w-full"
                        value={field.value as AppLocale}
                        onChange={field.onChange}
                    />
                    <FormMessage />
                </FormItem>
            )}
        />
    )
}

/** Why the set-password link may or may not be emailed. */
const sendEmailHelp = (t: TFunction, mail: { canEmail: boolean; mustEmail: boolean }) => {
    if (!mail.mustEmail) return t('forms:users.sendEmailHelp')
    return mail.canEmail ? t('forms:users.sendEmailRequired') : t('forms:users.sendEmailUnavailable')
}

export function UserForm({
    defaultValues,
    onSubmit,
    onValuesChange,
    isSubmitting,
    submitLabel,
    isEdit = false,
    formId,
    hideSubmit = false,
    mail = { canEmail: false, mustEmail: false },
}: UserFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const currentUser = useUser()
    const isEditingSelf = isEdit && defaultValues?.id === currentUser?.id
    const uiLocale = useUiLocale()

    const form = useForm<CreateUserFormData>({
        resolver: schemaResolver<CreateUserFormData>(isEdit ? updateUserSchema : createUserSchema),
        defaultValues: {
            name: '',
            email: '',
            password: '',
            role: 'user',
            setPassword: false,
            sendEmail: mail.canEmail,
            locale: uiLocale,
            ...defaultValues,
        },
    })

    const setPassword = form.watch('setPassword')
    const sendEmail = form.watch('sendEmail')
    const showPassword = isEdit || setPassword

    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
            <Form {...form}>
                <form id={formId} onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                    <FormInputField
                        control={form.control}
                        name="name"
                        label={t('fields.name')}
                        inputProps={{ placeholder: t('forms:users.namePlaceholder') }}
                    />

                    <FormInputField
                        control={form.control}
                        name="email"
                        label={t('fields.email')}
                        inputProps={{ type: 'email', placeholder: t('forms:users.emailPlaceholder') }}
                    />

                    {!isEdit && (
                        <FormCheckboxCard
                            control={form.control}
                            name="setPassword"
                            label={t('forms:users.setPassword')}
                            description={t('forms:users.setPasswordHelp')}
                        />
                    )}

                    {!isEdit && !setPassword && (mail.canEmail || mail.mustEmail) && (
                        <FormCheckboxCard
                            control={form.control}
                            name="sendEmail"
                            label={t('forms:users.sendEmail')}
                            description={sendEmailHelp(t, mail)}
                            forceChecked={mail.mustEmail}
                            disabled={mail.mustEmail || !mail.canEmail}
                        />
                    )}

                    {!isEdit && !setPassword && mail.canEmail && (mail.mustEmail || sendEmail) && (
                        <UserLocaleField form={form} />
                    )}

                    {showPassword && (
                        <FormInputField
                            control={form.control}
                            name="password"
                            label={isEdit ? t('forms:users.newPassword') : t('fields.password')}
                            description={isEdit ? t('forms:users.passwordKeepHelp') : undefined}
                            inputProps={{
                                type: 'password',
                                placeholder: isEdit
                                    ? t('forms:users.passwordKeepPlaceholder')
                                    : t('forms:users.passwordPlaceholder'),
                            }}
                        />
                    )}

                    <UserRoleField form={form} disabled={isEditingSelf} />

                    <FormSubmitButton hidden={hideSubmit} isSubmitting={isSubmitting} label={submitLabel} />
                </form>
            </Form>
        </FormWrapper>
    )
}
