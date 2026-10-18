import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { CardContent, CardFooter } from '@/components/ui/card'
import { PermissionPicker } from '@/components/features/api-tokens/PermissionPicker'
import { SpacePicker, hasSpaces, type SpaceChoice } from '@/components/features/api-tokens/SpacePicker'
import { testId } from '@/lib/test-id'
import { useUser } from '@/stores/auth'
import type { ApiPermission, ApiResource, KindOptions } from '@/types'

interface ConsentFormProps {
    spaces: ReadonlyArray<{ id: number; name: string }>
    options: KindOptions | null
    permissions: ApiPermission[]
    onPermissionsChange: (permissions: ApiPermission[]) => void
    spaceChoice: SpaceChoice
    onSpaceChoiceChange: (choice: SpaceChoice) => void
    /** Ticked permissions that change data: allowing them asks once more. */
    changes: ApiResource[]
    host: string | null
    error: string | null
    busy: 'allow' | 'deny' | null
    onDecide: (approve: boolean) => void
    onAsk: () => void
}

/** What the app asks for, the spaces it may see and the allow/deny buttons. */
export function ConsentForm({
    spaces,
    options,
    permissions,
    onPermissionsChange,
    spaceChoice,
    onSpaceChoiceChange,
    changes,
    host,
    error,
    busy,
    onDecide,
    onAsk,
}: ConsentFormProps) {
    const { t } = useTranslation('auth')
    const user = useUser()

    return (
        <>
            <CardContent className="space-y-5 text-sm">
                <div className="space-y-2">
                    <p className="font-medium">{t('oauth.canTitle')}</p>
                    {options && (
                        <PermissionPicker value={permissions} onChange={onPermissionsChange} options={options} />
                    )}
                    <p className="text-xs text-muted-foreground">
                        {changes.length === 0 && `${t('oauth.readOnly')} `}
                        {t('oauth.cannot')}
                    </p>
                </div>
                <div>
                    <p className="font-medium mb-2">{t('oauth.spacesTitle')}</p>
                    <SpacePicker
                        idPrefix="oauth-space"
                        spaces={spaces}
                        value={spaceChoice}
                        onChange={onSpaceChoiceChange}
                    />
                    {!hasSpaces(spaceChoice) && (
                        <p className="mt-1 text-xs text-muted-foreground">{t('oauth.needSpace')}</p>
                    )}
                </div>
                <p className="text-muted-foreground">{t('oauth.revoke')}</p>
                {host && <p className="font-medium">{t('oauth.returnsTo', { host })}</p>}
                {user && (
                    <p className="text-xs text-muted-foreground">{t('oauth.signedInAs', { email: user.email })}</p>
                )}
                {error && (
                    <Alert variant="destructive">
                        <AlertDescription>{error}</AlertDescription>
                    </Alert>
                )}
            </CardContent>
            <CardFooter className="gap-2">
                <Button
                    variant="outline"
                    className="flex-1"
                    disabled={busy !== null}
                    onClick={() => onDecide(false)}
                    {...testId('oauth-deny')}
                >
                    {busy === 'deny' && <Loader2 className="size-4 animate-spin" />}
                    {t('oauth.deny')}
                </Button>
                <Button
                    className="flex-1"
                    disabled={busy !== null || !hasSpaces(spaceChoice) || permissions.length === 0}
                    onClick={() => (changes.length > 0 ? onAsk() : onDecide(true))}
                    {...testId('oauth-allow')}
                >
                    {busy === 'allow' && <Loader2 className="size-4 animate-spin" />}
                    {t('oauth.allow')}
                </Button>
            </CardFooter>
        </>
    )
}
