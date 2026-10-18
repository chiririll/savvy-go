import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'

import { authApi, onboardingApi } from '@/api'
import { pages } from '@/app/pages'
import { isLaravelFile } from '@/components/features/backups/LaravelRestoreOptions'
import { getApiErrorMessage } from '@/lib/api-error'
import { useAuthStore } from '@/stores/auth'
import type { ApiError } from '@/types'

export type AdminSetupValues = {
    name: string
    email: string
    password: string
    password_confirmation: string
}

export type SetupStep = 'bootstrap' | 'admin' | 'restore'

// The new admin goes on to the server setup, then 2FA.
const NEXT_PATH = pages.serverSetup.path

/** State and actions of the first-run flow: the bootstrap password, then the first admin or a restore. */
export function useAdminSetup() {
    const { t } = useTranslation('auth')
    const navigate = useNavigate()
    const queryClient = useQueryClient()
    const register = useAuthStore((state) => state.register)
    const [step, setStep] = useState<SetupStep>('bootstrap')
    const [isLoading, setIsLoading] = useState(false)
    // Not a form field: a field of the first screen would lose its value when the screen unmounts.
    const [bootstrapPassword, setBootstrapPassword] = useState('')
    const [bootstrapError, setBootstrapError] = useState<string | null>(null)
    const [backupFile, setBackupFile] = useState<File | null>(null)
    const [appKey, setAppKey] = useState('')
    const [restoreError, setRestoreError] = useState<string | null>(null)

    const { data: status } = useQuery({ queryKey: ['auth', 'status'], queryFn: authApi.status, retry: false })

    // Registering is what makes needs_registration false: that must not send the new admin to the login page.
    const registered = useRef(false)

    useEffect(() => {
        if (status && !status.needs_registration && !registered.current) {
            navigate('/login', { replace: true })
        }
    }, [status, navigate])

    const schema = useMemo(
        () =>
            z
                .object({
                    name: z.string().min(1, t('setup.validation.name')),
                    email: z.string().email(t('setup.validation.email')),
                    password: z.string().min(8, t('setup.validation.passwordMin')),
                    password_confirmation: z.string(),
                })
                .refine((data) => data.password === data.password_confirmation, {
                    message: t('setup.validation.passwordMatch'),
                    path: ['password_confirmation'],
                }),
        [t]
    )

    const form = useForm<AdminSetupValues>({
        resolver: zodResolver(schema),
        defaultValues: { name: '', email: '', password: '', password_confirmation: '' },
    })

    const goTo = async (next: 'admin' | 'restore') => {
        const password = bootstrapPassword.trim()
        if (password === '') {
            setBootstrapError(t('setup.validation.bootstrapPassword'))
            return
        }
        setIsLoading(true)
        try {
            await authApi.verifyBootstrap(password)
            setBootstrapError(null)
            setStep(next)
        } catch (error: unknown) {
            const wrong = (error as ApiError).details?.bootstrap_password?.[0]
            if (wrong) {
                setBootstrapError(wrong)
            } else {
                toast.error(getApiErrorMessage(error, t('setup.failed')))
            }
        } finally {
            setIsLoading(false)
        }
    }

    const onRestore = async () => {
        if (!backupFile) {
            setRestoreError(t('setup.validation.backupFile'))
            return
        }
        setIsLoading(true)
        setRestoreError(null)
        try {
            await onboardingApi.restore(
                bootstrapPassword.trim(),
                backupFile,
                isLaravelFile(backupFile) && appKey.trim() !== '' ? appKey.trim() : undefined
            )
            // The backup brought its own users and no session: drop the cached needs_registration=true and sign in.
            registered.current = true
            await queryClient.invalidateQueries({ queryKey: ['auth', 'status'] })
            toast.success(t('setup.restored'))
            navigate('/login', { replace: true })
        } catch (error: unknown) {
            const details = (error as ApiError).details
            if (details?.bootstrap_password?.[0]) {
                setBootstrapError(details.bootstrap_password[0])
                setStep('bootstrap')
            } else if (details?.file?.[0]) {
                setRestoreError(details.file[0])
            } else {
                toast.error(getApiErrorMessage(error, t('setup.restoreFailed')))
            }
        } finally {
            setIsLoading(false)
        }
    }

    const onSubmit = async (data: AdminSetupValues) => {
        setIsLoading(true)
        try {
            await register({
                bootstrap_password: bootstrapPassword.trim(),
                name: data.name,
                email: data.email,
                password: data.password,
            })
            registered.current = true
            // AuthProvider caches the status forever; drop the stale needs_registration=true.
            await queryClient.invalidateQueries({ queryKey: ['auth', 'status'] })
            toast.success(t('setup.created'))
            navigate(NEXT_PATH)
        } catch (error: unknown) {
            const wrongPassword = (error as ApiError).details?.bootstrap_password?.[0]
            if (wrongPassword) {
                // The password is on the first screen: take the user back to it.
                setBootstrapError(wrongPassword)
                setStep('bootstrap')
            } else {
                toast.error(getApiErrorMessage(error, t('setup.failed')))
            }
        } finally {
            setIsLoading(false)
        }
    }

    return {
        status,
        form,
        step,
        setStep,
        isLoading,
        bootstrapPassword,
        setBootstrapPassword,
        bootstrapError,
        backupFile,
        setBackupFile,
        appKey,
        setAppKey,
        restoreError,
        setRestoreError,
        goTo,
        onRestore,
        onSubmit,
    }
}
