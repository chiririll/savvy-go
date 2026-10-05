import { testId } from '@/lib/test-id'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Copy, KeyRound, RefreshCw, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { Page, PageHeader, SettingsSection } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
import { useRotateKey, useSigningKeys, useTrustKey, useUntrustKey } from '@/hooks/use-spaces'

function SigningKeys() {
    const { t } = useTranslation('settings')
    const { data: keys } = useSigningKeys()
    const trust = useTrustKey()
    const untrust = useUntrustKey()
    const rotate = useRotateKey()
    const [name, setName] = useState('')
    const [publicKey, setPublicKey] = useState('')

    const copy = async (value: string) => {
        await navigator.clipboard.writeText(value)
        toast.success(t('admin.keys.copied'))
    }

    return (
        <SettingsSection
            icon={KeyRound}
            title={t('admin.keys.title')}
            description={t('admin.keys.description')}
            actions={
                <AlertDialog>
                    <AlertDialogTrigger asChild {...testId('confirm')}>
                        <Button variant="outline" size="sm" disabled={rotate.isPending}>
                            <RefreshCw className="mr-2 size-4" />
                            {t('admin.keys.rotate')}
                        </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                        <AlertDialogHeader>
                            <AlertDialogTitle>{t('admin.keys.rotateTitle')}</AlertDialogTitle>
                            <AlertDialogDescription>{t('admin.keys.rotateDescription')}</AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                            <AlertDialogCancel>{t('actions.cancel', { ns: 'common' })}</AlertDialogCancel>
                            <AlertDialogAction onClick={() => rotate.mutate()}>{t('admin.keys.rotate')}</AlertDialogAction>
                        </AlertDialogFooter>
                    </AlertDialogContent>
                </AlertDialog>
            }
        >
            <div className="space-y-5">
                <div className="space-y-1.5">
                    <Label>{t('admin.keys.own')}</Label>
                    <div className="flex gap-2">
                        <Input readOnly value={keys?.own.publicKey ?? ''} className="font-mono text-xs" />
                        <Button variant="outline" size="icon" onClick={() => keys && copy(keys.own.publicKey)} title={t('admin.keys.copy')}>
                            <Copy className="size-4" />
                        </Button>
                    </div>
                    <p className="text-xs text-muted-foreground">{t('admin.keys.ownHelp', { kid: keys?.own.kid ?? '' })}</p>
                </div>

                <div className="space-y-2">
                    <Label>{t('admin.keys.trusted')}</Label>
                    {keys?.trusted.length ? (
                        <ul className="divide-y rounded-lg border">
                            {keys.trusted.map((k) => (
                                <li key={k.kid} className="flex items-center justify-between gap-3 px-3 py-2">
                                    <div className="min-w-0">
                                        <p className="truncate text-sm font-medium">{k.name}</p>
                                        <p className="truncate font-mono text-xs text-muted-foreground">{k.kid}</p>
                                    </div>
                                    <Button variant="ghost" size="icon" onClick={() => untrust.mutate(k.kid)} title={t('admin.keys.untrust')}>
                                        <Trash2 className="size-4" />
                                    </Button>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <p className="text-sm text-muted-foreground">{t('admin.keys.noneTrusted')}</p>
                    )}
                </div>

                <form
                    className="grid gap-2 sm:grid-cols-[1fr_2fr_auto]"
                    onSubmit={async (e) => {
                        e.preventDefault()
                        await trust.mutateAsync({ name: name.trim(), publicKey: publicKey.trim() })
                        setName('')
                        setPublicKey('')
                    }}
                >
                    <Input placeholder={t('admin.keys.namePlaceholder')} value={name} onChange={(e) => setName(e.target.value)} />
                    <Input placeholder={t('admin.keys.keyPlaceholder')} value={publicKey} onChange={(e) => setPublicKey(e.target.value)} className="font-mono text-xs" />
                    <Button type="submit" disabled={!name.trim() || !publicKey.trim() || trust.isPending}>
                        {t('admin.keys.trust')}
                    </Button>
                </form>
            </div>
        </SettingsSection>
    )
}

/** How this server signs backups and transfers, and whose signatures it trusts. */
export default function AdminSecurityPage() {
    const { t } = useTranslation('settings')
    return (
        <Page title={t('admin.security.title')}>
            <PageHeader title={t('admin.security.title')} description={t('admin.security.description')} />
            <div className="max-w-3xl">
                <SigningKeys />
            </div>
        </Page>
    )
}
