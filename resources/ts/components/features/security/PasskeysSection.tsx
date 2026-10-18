import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Fingerprint, Plus, Trash2 } from 'lucide-react'
import { testId } from '@/lib/test-id'
import { intlLocale } from '@/lib/i18n'
import { ResponsiveDialog } from '@/components/shared'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { isPasskeyDomainSupported, useDeletePasskey, useRegisterPasskey, useWebauthnCredentials } from '@/hooks'
import { RowSkeleton, Section } from './SecurityLayout'
import { cn } from '@/lib/utils'

const formatDate = (value: string) => new Date(value).toLocaleDateString(intlLocale())

interface Passkey {
    id: number | string
    name?: string | null
    createdAt: string
    lastUsedAt?: string | null
}

function PasskeyRow({ passkey, disabled, onDelete }: { passkey: Passkey; disabled: boolean; onDelete: () => void }) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')

    return (
        <div className="flex items-center justify-between gap-4 px-4 py-3.5">
            <div className="min-w-0 space-y-1">
                <div className="truncate text-sm font-medium">{passkey.name || t('security.passkeys.unnamed')}</div>
                <p className="text-sm leading-relaxed text-muted-foreground">
                    {t('security.passkeys.added', { date: formatDate(passkey.createdAt) })}
                    {' · '}
                    {passkey.lastUsedAt
                        ? t('security.passkeys.lastUsed', { date: formatDate(passkey.lastUsedAt) })
                        : t('security.passkeys.neverUsed')}
                </p>
            </div>
            <AlertDialog>
                <AlertDialogTrigger asChild>
                    <Button
                        variant="ghost"
                        size="icon"
                        disabled={disabled}
                        aria-label={t('security.passkeys.removeAria')}
                    >
                        <Trash2 className="size-4 text-destructive" />
                    </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>{t('security.passkeys.deleteTitle')}</AlertDialogTitle>
                        <AlertDialogDescription>{t('security.passkeys.deleteDescription')}</AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>{tCommon('actions.cancel')}</AlertDialogCancel>
                        <AlertDialogAction variant="destructive" onClick={onDelete}>
                            {tCommon('actions.delete')}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    )
}

export function PasskeysSection() {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const isReadOnly = useReadOnly()
    const passkeyDomainValid = isPasskeyDomainSupported()
    const { data: passkeys, isLoading } = useWebauthnCredentials()
    const registerPasskey = useRegisterPasskey()
    const deletePasskey = useDeletePasskey()

    const [addOpen, setAddOpen] = useState(false)
    const [name, setName] = useState('')

    const closeAdd = () => {
        setAddOpen(false)
        setName('')
    }

    const handleAdd = async () => {
        try {
            await registerPasskey.mutateAsync(name.trim() || null)
            closeAdd()
        } catch {
            // toasted by the mutation, or the user cancelled the confirmation
        }
    }

    return (
        <>
            <Section
                icon={Fingerprint}
                title={t('security.passkeys.title')}
                description={t('security.passkeys.description')}
            >
                <div className="space-y-4">
                    {isLoading ? (
                        <div className="rounded-lg border">
                            <RowSkeleton />
                        </div>
                    ) : passkeys && passkeys.length > 0 ? (
                        <div className="divide-y rounded-lg border">
                            {passkeys.map((passkey) => (
                                <PasskeyRow
                                    key={passkey.id}
                                    passkey={passkey}
                                    disabled={isReadOnly || deletePasskey.isPending}
                                    onDelete={() => deletePasskey.mutate(passkey.id)}
                                />
                            ))}
                        </div>
                    ) : (
                        <div
                            className={cn(
                                'rounded-lg border border-dashed px-4 py-6 text-center text-sm',
                                'text-muted-foreground',
                            )}
                        >
                            {t('security.passkeys.empty')}
                        </div>
                    )}
                    {!passkeyDomainValid && (
                        <p className="text-sm text-muted-foreground">{t('security.passkeys.domainHint')}</p>
                    )}
                    <Button
                        onClick={() => setAddOpen(true)}
                        {...testId('add-passkey')}
                        disabled={isReadOnly || registerPasskey.isPending || !passkeyDomainValid}
                    >
                        <Plus className="mr-2 size-4" />
                        {t('security.passkeys.add')}
                    </Button>
                </div>
            </Section>

            <ResponsiveDialog
                open={addOpen}
                onOpenChange={(open) => (open ? setAddOpen(true) : closeAdd())}
                title={t('security.passkeys.addTitle')}
                description={t('security.passkeys.addDescription')}
                footer={
                    <>
                        <Button variant="outline" onClick={closeAdd}>
                            {tCommon('actions.cancel')}
                        </Button>
                        <Button onClick={handleAdd} disabled={registerPasskey.isPending}>
                            {registerPasskey.isPending ? t('security.passkeys.waiting') : tCommon('actions.continue')}
                        </Button>
                    </>
                }
            >
                <Input
                    placeholder={t('security.passkeys.namePlaceholder')}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    maxLength={100}
                    autoFocus
                    onKeyDown={(e) => {
                        if (e.key === 'Enter' && !registerPasskey.isPending) handleAdd()
                    }}
                />
            </ResponsiveDialog>
        </>
    )
}
