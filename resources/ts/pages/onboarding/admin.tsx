import { Loader2 } from 'lucide-react'
import type { FormEvent } from 'react'
import { useTranslation } from 'react-i18next'

import { AdminFieldsStep } from '@/components/features/onboarding/admin-setup/AdminFieldsStep'
import { BootstrapStep } from '@/components/features/onboarding/admin-setup/BootstrapStep'
import { RestoreStep } from '@/components/features/onboarding/admin-setup/RestoreStep'
import { useAdminSetup } from '@/components/features/onboarding/admin-setup/useAdminSetup'
import { OnboardingLayout } from '@/components/features/onboarding/OnboardingLayout'
import { Form } from '@/components/ui/form'

/** Flow 1: the bootstrap password, then the first admin. */
export default function AdminSetupPage() {
    const { t } = useTranslation('auth')
    const setup = useAdminSetup()
    const { step, setStep, form, status } = setup

    if (!status?.needs_registration) {
        return (
            <div className="min-h-screen flex items-center justify-center bg-background">
                <Loader2 className="size-8 animate-spin text-muted-foreground" />
            </div>
        )
    }

    const titles = {
        bootstrap: { title: t('setup.title'), description: t('setup.description') },
        restore: { title: t('setup.restoreTitle'), description: t('setup.restoreDescription') },
        admin: { title: t('setup.adminTitle'), description: t('setup.adminDescription') },
    }

    const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
        // Enter in the password field submits the form: on the first screen that means "next",
        // not validating admin fields the user has not seen yet.
        if (step === 'bootstrap') {
            e.preventDefault()
            void setup.goTo('admin')
        } else if (step === 'restore') {
            e.preventDefault()
            void setup.onRestore()
        } else {
            void form.handleSubmit(setup.onSubmit)(e)
        }
    }

    return (
        <OnboardingLayout {...titles[step]}>
            <Form {...form}>
                <form onSubmit={handleSubmit} className="space-y-4">
                    {step === 'bootstrap' && (
                        <BootstrapStep
                            hint={status.bootstrap_hint}
                            password={setup.bootstrapPassword}
                            onPasswordChange={setup.setBootstrapPassword}
                            error={setup.bootstrapError}
                            isLoading={setup.isLoading}
                            onNext={(next) => void setup.goTo(next)}
                        />
                    )}
                    {step === 'restore' && (
                        <RestoreStep
                            backupFile={setup.backupFile}
                            onFileChange={(file) => {
                                setup.setBackupFile(file)
                                setup.setRestoreError(null)
                            }}
                            appKey={setup.appKey}
                            onAppKeyChange={setup.setAppKey}
                            error={setup.restoreError}
                            isLoading={setup.isLoading}
                            onBack={() => setStep('bootstrap')}
                        />
                    )}
                    {step === 'admin' && (
                        <AdminFieldsStep form={form} isLoading={setup.isLoading} onBack={() => setStep('bootstrap')} />
                    )}
                </form>
            </Form>
        </OnboardingLayout>
    )
}
