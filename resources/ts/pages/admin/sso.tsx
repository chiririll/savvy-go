import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { ProviderCard, ProvidersEmptyState, ProviderGridSkeleton } from '@/components/features/sso'
import { useIdentityProviders, useDeleteIdentityProvider, useTestIdentityProvider } from '@/hooks/use-sso'

/** Single sign-on providers. */
export default function AdminSsoPage() {
    const { t } = useTranslation('settings')
    const { data: providers, isLoading } = useIdentityProviders()
    const deleteProvider = useDeleteIdentityProvider()
    const testProvider = useTestIdentityProvider()
    const hasProviders = !!providers?.length

    return (
        <Page title={t('providers.title')}>
            <PageHeader
                title={t('providers.title')}
                description={t('providers.description')}
                createLink={hasProviders ? '/admin/providers/create' : undefined}
                createLabel={t('providers.add')}
            />

            {isLoading ? (
                <ProviderGridSkeleton />
            ) : !hasProviders ? (
                <ProvidersEmptyState />
            ) : (
                <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                    {providers.map((provider, index) => (
                        <ProviderCard
                            key={provider.id}
                            provider={provider}
                            index={index}
                            onDelete={(id) => deleteProvider.mutate(id)}
                            onTest={(id) => testProvider.mutate(id)}
                        />
                    ))}
                </div>
            )}
        </Page>
    )
}
