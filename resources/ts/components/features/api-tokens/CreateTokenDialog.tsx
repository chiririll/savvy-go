import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useApiTokenCapabilities, useCreateApiToken } from '@/hooks'
import { kindOptions, permissionLevel, type ApiPermission } from '@/types'
import { SyncWarning } from './SyncWarning'
import { PermissionPicker } from './PermissionPicker'
import { SpacePicker, hasSpaces, type SpaceChoice } from './SpacePicker'
import { expiryToIso, type Expiry } from './token-grants'

interface CreateTokenDialogProps {
    open: boolean
    spaces: ReadonlyArray<{ id: number; name: string }>
    onOpenChange: (open: boolean) => void
    /** Gets the plain token, which is shown once and never again. */
    onCreated: (token: string) => void
}

export function CreateTokenDialog({ open, spaces, onOpenChange, onCreated }: CreateTokenDialogProps) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const createToken = useCreateApiToken()
    const { data: capabilities } = useApiTokenCapabilities()
    const personal = capabilities ? kindOptions(capabilities, 'personal') : null

    const [name, setName] = useState('')
    // Until the user changes it, a new token gets the lowest preset.
    const [chosen, setChosen] = useState<ApiPermission[] | null>(null)
    const permissions = chosen ?? personal?.presets[0]?.permissions ?? []
    const [spaceChoice, setSpaceChoice] = useState<SpaceChoice>({ all: false, ids: [] })
    const [expiry, setExpiry] = useState<Expiry>('never')

    // Categories and tags are mirrored between spaces of a sync group, so only a
    // token that can change them needs the warning.
    const changesSynced = (['categories', 'tags'] as const).some((r) =>
        ['write', 'delete'].includes(permissionLevel(permissions, r) ?? ''),
    )

    const resetForm = () => {
        setName('')
        setChosen(null)
        setSpaceChoice({ all: false, ids: [] })
        setExpiry('never')
    }

    const handleCreate = async () => {
        try {
            const created = await createToken.mutateAsync({
                name: name.trim(),
                permissions,
                all_spaces: spaceChoice.all,
                space_ids: spaceChoice.ids,
                expires_at: expiryToIso(expiry),
            })
            onOpenChange(false)
            resetForm()
            onCreated(created.token)
        } catch {
            // toasted by the mutation
        }
    }

    return (
        <ResponsiveDialog
            open={open}
            onOpenChange={(next) => {
                onOpenChange(next)
                if (!next) resetForm()
            }}
            title={t('api.create.title')}
            description={t('api.create.description')}
            footer={
                <>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button
                        onClick={handleCreate}
                        disabled={!name.trim() || !hasSpaces(spaceChoice) || createToken.isPending}
                    >
                        {tCommon('actions.create')}
                    </Button>
                </>
            }
        >
            <div className="space-y-4">
                <div className="space-y-2">
                    <Label htmlFor="api-token-name">{t('api.create.name')}</Label>
                    <Input
                        id="api-token-name"
                        placeholder={t('api.create.namePlaceholder')}
                        value={name}
                        maxLength={100}
                        autoFocus
                        onChange={(e) => setName(e.target.value)}
                    />
                </div>
                <div className="space-y-2">
                    <Label>{t('api.create.spaces')}</Label>
                    <SpacePicker idPrefix="token-space" spaces={spaces} value={spaceChoice} onChange={setSpaceChoice} />
                    <p className="text-xs text-muted-foreground">
                        {hasSpaces(spaceChoice) ? t('api.create.spacesHint') : t('api.spaces.none')}
                    </p>
                </div>
                {changesSynced && <SyncWarning spaces={spaces} choice={spaceChoice} />}
                <div className="space-y-2">
                    <Label>{t('api.create.permissions')}</Label>
                    {personal && <PermissionPicker value={permissions} onChange={setChosen} options={personal} />}
                    <p className="text-xs text-muted-foreground">{t('api.create.permissionsHint')}</p>
                </div>
                <div className="space-y-2">
                    <Label htmlFor="api-token-expiry">{t('api.create.expiry')}</Label>
                    <Select value={expiry} onValueChange={(v) => setExpiry(v as Expiry)}>
                        <SelectTrigger id="api-token-expiry" className="w-full">
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            <SelectItem value="never">{t('api.create.expiryNever')}</SelectItem>
                            <SelectItem value="30">{t('api.create.expiryDays', { count: 30 })}</SelectItem>
                            <SelectItem value="90">{t('api.create.expiryDays', { count: 90 })}</SelectItem>
                            <SelectItem value="365">{t('api.create.expiryDays', { count: 365 })}</SelectItem>
                        </SelectContent>
                    </Select>
                </div>
            </div>
        </ResponsiveDialog>
    )
}
