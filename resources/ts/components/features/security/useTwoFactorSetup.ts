import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
    useConfirmTwoFactor,
    useDisableTwoFactor,
    useEnableTwoFactor,
    useRegenerateRecoveryCodes,
    useTwoFactorStatus,
} from '@/hooks'

export type SetupStep = 'qr' | 'verify' | 'recovery'

/** State and handlers shared by the two-factor row, the recovery-codes row and their dialogs. */
export function useTwoFactorSetup() {
    const { t } = useTranslation('settings')
    const { data: status, isLoading } = useTwoFactorStatus()
    const enableMutation = useEnableTwoFactor()
    const confirmMutation = useConfirmTwoFactor()
    const disableMutation = useDisableTwoFactor()
    const regenerateMutation = useRegenerateRecoveryCodes()

    const [showEnableDialog, setShowEnableDialog] = useState(false)
    const [showDisableDialog, setShowDisableDialog] = useState(false)
    const [showRegenerateDialog, setShowRegenerateDialog] = useState(false)
    const [setupStep, setSetupStep] = useState<SetupStep>('qr')
    const [qrData, setQrData] = useState<{ secret: string; qr_code_url: string } | null>(null)
    const [recoveryCodes, setRecoveryCodes] = useState<string[]>([])
    const [otpValue, setOtpValue] = useState('')

    const handleEnable = async () => {
        try {
            const data = await enableMutation.mutateAsync()
            setQrData(data)
            setSetupStep('qr')
            setShowEnableDialog(true)
        } catch {
            // toasted by the mutation, or the user cancelled the confirmation
        }
    }

    const handleConfirm = async () => {
        const data = await confirmMutation.mutateAsync(otpValue)
        setRecoveryCodes(data.recovery_codes)
        setSetupStep('recovery')
        setOtpValue('')
    }

    const handleDisable = async () => {
        await disableMutation.mutateAsync(otpValue)
        setShowDisableDialog(false)
        setOtpValue('')
    }

    const handleRegenerate = async () => {
        const data = await regenerateMutation.mutateAsync(otpValue)
        setRecoveryCodes(data.recovery_codes)
        setShowRegenerateDialog(false)
        setSetupStep('recovery')
        setShowEnableDialog(true)
        setOtpValue('')
    }

    const copySecret = () => {
        if (qrData?.secret) {
            navigator.clipboard.writeText(qrData.secret)
            toast.success(t('security.dialogs.secretCopied'))
        }
    }

    const copyRecoveryCodes = () => {
        navigator.clipboard.writeText(recoveryCodes.join('\n'))
        toast.success(t('security.dialogs.codesCopied'))
    }

    const closeEnableDialog = () => {
        setShowEnableDialog(false)
        setSetupStep('qr')
        setQrData(null)
        setRecoveryCodes([])
        setOtpValue('')
    }

    const closeWithOtpReset = (close: (open: boolean) => void) => () => {
        close(false)
        setOtpValue('')
    }

    return {
        status,
        isLoading,
        enableMutation,
        confirmMutation,
        disableMutation,
        regenerateMutation,
        showEnableDialog,
        showDisableDialog,
        setShowDisableDialog,
        showRegenerateDialog,
        setShowRegenerateDialog,
        setupStep,
        setSetupStep,
        qrData,
        recoveryCodes,
        otpValue,
        setOtpValue,
        handleEnable,
        handleConfirm,
        handleDisable,
        handleRegenerate,
        copySecret,
        copyRecoveryCodes,
        closeEnableDialog,
        closeWithOtpReset,
    }
}

export type TwoFactorSetup = ReturnType<typeof useTwoFactorSetup>
