import { testId } from '@/lib/test-id'
import { UserSettingsTabs } from '@/components/features/spaces/UserSettingsTabs'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ExternalLink, KeyRound, Plus } from 'lucide-react'
import { Page, PageHeader, FormWrapper } from '@/components/shared'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useApiTokens, useRevokeApiToken, useSpaces } from '@/hooks'
import { EditTokenDialog } from '@/components/features/api-tokens/EditTokenDialog'
import { CreateTokenDialog } from '@/components/features/api-tokens/CreateTokenDialog'
import { CreatedTokenDialog } from '@/components/features/api-tokens/CreatedTokenDialog'
import { TokenRow } from '@/components/features/api-tokens/TokenRow'
import type { ApiTokenSummary } from '@/types'
import { cn } from '@/lib/utils'

export default function ApiSettingsPage() {
    const { t } = useTranslation('settings')

    const { data: tokens, isLoading } = useApiTokens()
    const revokeToken = useRevokeApiToken()
    const { data: spaces = [] } = useSpaces()

    const [showCreate, setShowCreate] = useState(false)
    const [createdToken, setCreatedToken] = useState<string | null>(null)
    const [editing, setEditing] = useState<ApiTokenSummary | null>(null)

    return (
        <Page title={t('api.title')}>
            <PageHeader title={t('api.heading')} description={t('api.description')} />
            <UserSettingsTabs />

            <FormWrapper>
                <section className="rounded-xl border bg-card shadow-sm">
                    <header className="flex flex-wrap items-start justify-between gap-3 border-b px-5 py-4">
                        <div className="flex items-start gap-3">
                            <div
                                className={cn(
                                    'flex size-9 shrink-0 items-center justify-center rounded-lg border bg-muted/50',
                                    'text-muted-foreground',
                                )}
                            >
                                <KeyRound className="size-4" />
                            </div>
                            <div className="space-y-0.5">
                                <h3 className="text-sm font-semibold leading-none">{t('api.tokens.title')}</h3>
                                <p className="text-xs text-muted-foreground">{t('api.tokens.description')}</p>
                            </div>
                        </div>
                        <div className="flex shrink-0 gap-2">
                            <Button variant="outline" size="sm" asChild>
                                <a href="/api/docs" target="_blank" rel="noreferrer">
                                    <ExternalLink className="size-4" />
                                    {t('api.docs')}
                                </a>
                            </Button>
                            <Button size="sm" onClick={() => setShowCreate(true)} {...testId('create-token')}>
                                <Plus className="size-4" />
                                {t('api.tokens.add')}
                            </Button>
                        </div>
                    </header>

                    <div className="p-5">
                        {isLoading ? (
                            <Skeleton className="h-16 w-full" />
                        ) : tokens && tokens.length > 0 ? (
                            <div className="divide-y rounded-lg border">
                                {tokens.map((token) => (
                                    <TokenRow
                                        key={token.id}
                                        token={token}
                                        revoking={revokeToken.isPending}
                                        onEdit={() => setEditing(token)}
                                        onRevoke={() => revokeToken.mutate(token.id)}
                                    />
                                ))}
                            </div>
                        ) : (
                            <p
                                className={cn(
                                    'rounded-lg border border-dashed px-4 py-6 text-center text-sm',
                                    'text-muted-foreground',
                                )}
                            >
                                {t('api.tokens.empty')}
                            </p>
                        )}
                    </div>
                </section>
            </FormWrapper>

            <CreateTokenDialog
                open={showCreate}
                spaces={spaces}
                onOpenChange={setShowCreate}
                onCreated={setCreatedToken}
            />

            {editing && (
                <EditTokenDialog key={editing.id} token={editing} spaces={spaces} onClose={() => setEditing(null)} />
            )}

            <CreatedTokenDialog token={createdToken} onClose={() => setCreatedToken(null)} />
        </Page>
    )
}
