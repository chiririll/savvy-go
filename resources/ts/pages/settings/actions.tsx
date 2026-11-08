import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Trash2 } from 'lucide-react'
import { Page, PageHeader, FormWrapper, Confirm } from '@/components/shared'
import { ActionsList } from '@/components/features/audit/ActionsList'
import { ActionsFilters } from '@/components/features/audit/ActionsFilters'
import type { ActionFilters } from '@/components/features/audit/actions'
import { Button } from '@/components/ui/button'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { useClearSpaceActions, useCurrentSpace, useSpaceActions, useSpaceMembers } from '@/hooks'

/** What was changed in the current space: a member sees their own changes, a space admin everyone's and may clear the log. */
export default function SpaceActionsPage() {
    const { t } = useTranslation('settings')
    const space = useCurrentSpace()
    const isReadOnly = useReadOnly()
    const [filters, setFilters] = useState<ActionFilters>({})
    const actions = useSpaceActions(space?.id ?? null, filters)
    const { data: members } = useSpaceMembers(space?.id ?? null)
    const clear = useClearSpaceActions(space?.id ?? 0)

    if (!space) return <></>
    const isAdmin = space.role === 'admin'
    const rows = actions.data?.pages.flat() ?? []

    return (
        <Page title={t('spaces.actions.title')}>
            <PageHeader
                title={t('spaces.actions.heading', { name: space.name })}
                description={isAdmin ? t('spaces.actions.descriptionAdmin') : t('spaces.actions.description')}
                actions={
                    isAdmin &&
                    !isReadOnly && (
                        <Confirm
                            destructive
                            trigger={
                                <Button variant="outline" disabled={rows.length === 0 && !(filters.op || filters.entityType || filters.source)}>
                                    <Trash2 className="mr-2 size-4" />
                                    {t('spaces.actions.clear')}
                                </Button>
                            }
                            title={t('spaces.actions.clearTitle')}
                            description={t('spaces.actions.clearDescription')}
                            action={t('spaces.actions.clear')}
                            onConfirm={() => clear.mutate()}
                        />
                    )
                }
            />
            <FormWrapper>
                <ActionsFilters value={filters} onChange={setFilters} />
                <ActionsList
                    actions={rows}
                    members={members}
                    showWho={isAdmin}
                    filtered={!!(filters.op || filters.entityType || filters.source)}
                    isLoading={actions.isLoading}
                    hasMore={actions.hasNextPage}
                    isLoadingMore={actions.isFetchingNextPage}
                    onLoadMore={() => actions.fetchNextPage()}
                />
            </FormWrapper>
        </Page>
    )
}
