import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import type { RegisterInput } from './use-invite-actions'

interface Props {
    /** The address the invitation was sent to; it can't be changed. */
    email?: string | null
    busy: boolean
    onSubmit: (input: RegisterInput) => void
}

/** Creates an account through an invitation. */
export function InviteRegisterForm({ email, busy, onSubmit }: Props) {
    const { t } = useTranslation('settings')
    const [form, setForm] = useState<RegisterInput>({ name: '', email: email ?? '', password: '' })
    const patch = (changes: Partial<RegisterInput>) => setForm((f) => ({ ...f, ...changes }))

    return (
        <form
            onSubmit={(e) => {
                e.preventDefault()
                onSubmit(form)
            }}
            className="space-y-3"
        >
            <div className="space-y-1.5">
                <Label htmlFor="name">{t('invite.name')}</Label>
                <Input id="name" required value={form.name} onChange={(e) => patch({ name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
                <Label htmlFor="email">{t('invite.email')}</Label>
                <Input
                    id="email"
                    type="email"
                    required
                    readOnly={!!email}
                    value={form.email}
                    onChange={(e) => patch({ email: e.target.value })}
                />
            </div>
            <div className="space-y-1.5">
                <Label htmlFor="password">{t('invite.password')}</Label>
                <Input
                    id="password"
                    type="password"
                    minLength={8}
                    required
                    value={form.password}
                    onChange={(e) => patch({ password: e.target.value })}
                />
            </div>
            <Button type="submit" className="w-full" disabled={busy}>
                {t('invite.register')}
            </Button>
        </form>
    )
}
