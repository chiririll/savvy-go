import { useTranslation } from 'react-i18next'
import { useFormValuesChange } from '@/hooks'
import { useForm } from 'react-hook-form'
import { schemaResolver } from '@/lib/form-resolver'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import {
    Form,
    FormField,
    FormItem,
    FormLabel,
    FormControl,
    FormMessage,
    FormDescription,
} from '@/components/ui/form'
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select'
import {
    createUserSchema,
    updateUserSchema,
    CreateUserFormData,
    UpdateUserFormData,
} from '@/schemas/users'
import { useUser } from '@/stores/auth'
import { FormWrapper } from '@/components/shared/FormWrapper'

interface UserFormProps {
    defaultValues?: Partial<CreateUserFormData> & Partial<UpdateUserFormData> & { id?: number }
    onSubmit: (data: CreateUserFormData | UpdateUserFormData) => void
    onValuesChange?: (data: CreateUserFormData) => void
    isSubmitting?: boolean
    submitLabel?: string
    isEdit?: boolean
    formId?: string
    hideSubmit?: boolean
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
}: UserFormProps) {
    const { t } = useTranslation(['common', 'forms'])
    const currentUser = useUser()
    const isEditingSelf = isEdit && defaultValues?.id === currentUser?.id

    const form = useForm<CreateUserFormData>({
        resolver: schemaResolver<CreateUserFormData>(isEdit ? updateUserSchema : createUserSchema),
        defaultValues: {
            name: '',
            email: '',
            password: '',
            role: 'user',
            setPassword: false,
            ...defaultValues,
        },
    })

    const setPassword = form.watch('setPassword')
    const showPassword = isEdit || setPassword

    useFormValuesChange(form, onValuesChange)

    return (
        <FormWrapper>
            <Form {...form}>
                <form
                    id={formId}
                    onSubmit={form.handleSubmit(onSubmit)}
                    className="space-y-4"
                >
                    <FormField
                        control={form.control}
                        name="name"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('fields.name')}</FormLabel>
                                <FormControl>
                                    <Input placeholder={t('forms:users.namePlaceholder')} {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    <FormField
                        control={form.control}
                        name="email"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('fields.email')}</FormLabel>
                                <FormControl>
                                    <Input type="email" placeholder={t('forms:users.emailPlaceholder')} {...field} />
                                </FormControl>
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    {!isEdit && (
                        <FormField
                            control={form.control}
                            name="setPassword"
                            render={({ field }) => (
                                <FormItem className="flex flex-row items-start space-x-3 space-y-0 rounded-md border p-4">
                                    <FormControl>
                                        <Checkbox
                                            checked={Boolean(field.value)}
                                            onCheckedChange={field.onChange}
                                        />
                                    </FormControl>
                                    <div className="space-y-1 leading-none">
                                        <FormLabel>{t('forms:users.setPassword')}</FormLabel>
                                        <FormDescription>
                                            {t('forms:users.setPasswordHelp')}
                                        </FormDescription>
                                    </div>
                                </FormItem>
                            )}
                        />
                    )}

                    {showPassword && (
                        <FormField
                            control={form.control}
                            name="password"
                            render={({ field }) => (
                                <FormItem>
                                    <FormLabel>{isEdit ? t('forms:users.newPassword') : t('fields.password')}</FormLabel>
                                    <FormControl>
                                        <Input
                                            type="password"
                                            placeholder={isEdit ? t('forms:users.passwordKeepPlaceholder') : t('forms:users.passwordPlaceholder')}
                                            {...field}
                                        />
                                    </FormControl>
                                    {isEdit && (
                                        <FormDescription>
                                            {t('forms:users.passwordKeepHelp')}
                                        </FormDescription>
                                    )}
                                    <FormMessage />
                                </FormItem>
                            )}
                        />
                    )}

                    <FormField
                        control={form.control}
                        name="role"
                        render={({ field }) => (
                            <FormItem>
                                <FormLabel>{t('fields.role')}</FormLabel>
                                <Select
                                    value={field.value}
                                    onValueChange={field.onChange}
                                    disabled={isEditingSelf}
                                >
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
                                {isEditingSelf && (
                                    <FormDescription>
                                        {t('forms:users.cannotChangeOwnRole')}
                                    </FormDescription>
                                )}
                                <FormMessage />
                            </FormItem>
                        )}
                    />

                    {!hideSubmit && (
                        <Button type="submit" disabled={isSubmitting} className="w-full">
                            {isSubmitting ? t('actions.saving') : (submitLabel ?? t('actions.save'))}
                        </Button>
                    )}
                </form>
            </Form>
        </FormWrapper>
    )
}
