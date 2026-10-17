import { useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ResponsiveDialog } from '@/components/shared/ResponsiveDialog'
import { cancelConfirmation, submitConfirmation, useConfirmStore, type ConfirmMode } from '@/stores/confirm'

function Ask({ mode }: { mode: ConfirmMode }) {
    const { t } = useTranslation('auth')
    const { t: tCommon } = useTranslation('common')
    const [value, setValue] = useState('')
    const [error, setError] = useState<string | null>(null)
    const [busy, setBusy] = useState(false)

    const submit = async (event: FormEvent) => {
        event.preventDefault()
        if (!value.trim() || busy) return
        setBusy(true)
        setError(null)
        const fields = mode === 'code' ? { code: value.trim() } : { current_password: value }
        const message = await submitConfirmation(fields)
        setBusy(false)
        if (message) {
            setError(message)
            setValue('')
        }
    }

    return (
        <ResponsiveDialog
            open
            onOpenChange={(open) => !open && cancelConfirmation()}
            title={t('confirm.title')}
            description={mode === 'code' ? t('confirm.descriptionCode') : t('confirm.descriptionPassword')}
            footer={
                <>
                    <Button type="button" variant="outline" onClick={cancelConfirmation}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button type="submit" form="confirm-credentials" disabled={!value.trim() || busy}>
                        {busy && <Loader2 className="mr-2 size-4 animate-spin" />}
                        {tCommon('actions.confirm')}
                    </Button>
                </>
            }
        >
            <form id="confirm-credentials" onSubmit={submit} className="space-y-2">
                <Label htmlFor="confirm-value">
                    {mode === 'code' ? t('confirm.codeLabel') : t('confirm.passwordLabel')}
                </Label>
                <Input
                    id="confirm-value"
                    autoFocus
                    type={mode === 'code' ? 'text' : 'password'}
                    autoComplete={mode === 'code' ? 'one-time-code' : 'current-password'}
                    value={value}
                    onChange={(event) => setValue(event.target.value)}
                    aria-invalid={error !== null}
                />
                {error && (
                    <p role="alert" className="text-destructive text-sm">
                        {error}
                    </p>
                )}
            </form>
        </ResponsiveDialog>
    )
}

/** Asks for the password or code before a change to the account's credentials; see `withConfirmation`. */
export default function ConfirmDialog() {
    const pending = useConfirmStore((state) => state.pending)
    return pending ? <Ask mode={pending.mode} /> : null
}
