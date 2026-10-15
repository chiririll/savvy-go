import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ResponsiveDialog } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { useApiTokenCapabilities, useUpdateApiToken } from '@/hooks'
import { kindOptions, permissionLevel, type ApiPermission, type ApiTokenSummary } from '@/types'
import { PermissionPicker } from './PermissionPicker'
import { SpacePicker, hasSpaces, type SpaceChoice } from './SpacePicker'
import { SyncWarning } from './SyncWarning'

/**
 * Change what an existing token may do, and in which spaces. The app that holds
 * it keeps working with the new rights from its next request, so nothing has
 * to be connected again. Only the parts of a space the token's kind can hold
 * are offered (a connected AI app: the ones its tools use).
 */
export function EditTokenDialog({
    token,
    spaces,
    onClose,
}: {
    token: ApiTokenSummary
    spaces: ReadonlyArray<{ id: number; name: string }>
    onClose: () => void
}) {
    const { t } = useTranslation('settings')
    const { t: tCommon } = useTranslation('common')
    const update = useUpdateApiToken()
    const { data: capabilities } = useApiTokenCapabilities()
    const [permissions, setPermissions] = useState<ApiPermission[]>(token.permissions)
    const [spaceChoice, setSpaceChoice] = useState<SpaceChoice>({ all: token.all_spaces, ids: token.space_ids })

    const changesSynced = (['categories', 'tags'] as const).some((r) =>
        ['write', 'delete'].includes(permissionLevel(permissions, r) ?? ''),
    )

    const save = async () => {
        try {
            await update.mutateAsync({
                id: token.id,
                input: { permissions, all_spaces: spaceChoice.all, space_ids: spaceChoice.ids },
            })
            onClose()
        } catch {
            // toasted by the mutation
        }
    }

    return (
        <ResponsiveDialog
            open
            onOpenChange={(open) => !open && onClose()}
            title={t('api.edit.title', { name: token.name })}
            description={t('api.edit.description')}
            footer={
                <>
                    <Button variant="outline" onClick={onClose}>
                        {tCommon('actions.cancel')}
                    </Button>
                    <Button onClick={save} disabled={!hasSpaces(spaceChoice) || permissions.length === 0 || update.isPending}>
                        {tCommon('actions.save')}
                    </Button>
                </>
            }
        >
            <div className="space-y-4">
                <div className="space-y-2">
                    <Label>{t('api.create.spaces')}</Label>
                    <SpacePicker idPrefix="edit-token-space" spaces={spaces} value={spaceChoice} onChange={setSpaceChoice} />
                    {!hasSpaces(spaceChoice) && <p className="text-xs text-muted-foreground">{t('api.spaces.none')}</p>}
                </div>
                {changesSynced && <SyncWarning spaces={spaces} choice={spaceChoice} />}
                <div className="space-y-2">
                    <Label>{t('api.create.permissions')}</Label>
                    {capabilities && (
                        <PermissionPicker
                            value={permissions}
                            onChange={setPermissions}
                            options={kindOptions(capabilities, token.kind)}
                        />
                    )}
                    <p className="text-xs text-muted-foreground">{t('api.edit.hint')}</p>
                </div>
            </div>
        </ResponsiveDialog>
    )
}
