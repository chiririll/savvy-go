import { useTranslation } from 'react-i18next'
import { EntityFormDialog } from '@/components/shared'
import { User } from '@/types/users'
import { CreateUserFormData, UpdateUserFormData } from '@/schemas/users'
import { UserForm } from './UserForm'

const FORM_ID = 'user-form'

interface UserFormDialogProps {
    user?: User | null
    open: boolean
    onOpenChange: (open: boolean) => void
    onSubmit: (data: CreateUserFormData | UpdateUserFormData) => void
    isSubmitting?: boolean
    mail?: { canEmail: boolean; mustEmail: boolean }
}

export function UserFormDialog({
    user,
    open,
    onOpenChange,
    onSubmit,
    isSubmitting,
    mail,
}: UserFormDialogProps) {
    const { t } = useTranslation('pages')

    return (
        <EntityFormDialog<User, CreateUserFormData | UpdateUserFormData, CreateUserFormData & { id?: number }>
            entity={user}
            open={open}
            onOpenChange={onOpenChange}
            onSubmit={onSubmit}
            isSubmitting={isSubmitting}
            formId={FORM_ID}
            title={user ? t('users.editTitle') : t('users.createTitle')}
            description={user ? t('users.editDescription') : t('users.createDescription')}
            toFormValues={(item) => ({
                id: item.id,
                name: item.name,
                email: item.email,
                role: item.role,
            })}
        >
            {({ formKey, formProps, isEdit }) => (
                <UserForm
                    key={formKey}
                    defaultValues={formProps.defaultValues}
                    onSubmit={onSubmit}
                    onValuesChange={formProps.onValuesChange}
                    isSubmitting={formProps.isSubmitting}
                    formId={formProps.formId}
                    hideSubmit
                    isEdit={isEdit}
                    mail={mail}
                />
            )}
        </EntityFormDialog>
    )
}
