import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { FeedList, Page, PageHeader } from '@/components/shared'
import { CurrencyFormDialog, CurrencyRow } from '@/components/features/currencies'
import { Button } from '@/components/ui/button'
import { useCreateCurrency, useCurrencies, useDeleteCurrency, useSetBaseCurrency, useUpdateCurrency, useResourceFormDialog, useSettings } from '@/hooks'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { CurrencyFormData } from '@/schemas'
import type { Currency } from '@/types'

export default function CurrenciesPage() {
    const { t } = useTranslation('pages')
    const { data: currencies, isLoading } = useCurrencies()
    const deleteCurrency = useDeleteCurrency()
    const setBaseCurrency = useSetBaseCurrency()
    const createCurrency = useCreateCurrency()
    const updateCurrency = useUpdateCurrency()
    const { data: settings } = useSettings()
    const isReadOnly = useReadOnly()
    const items = currencies ?? []
    const form = useResourceFormDialog<Currency, CurrencyFormData>({
        items,
        isLoading,
        create: createCurrency,
        update: updateCurrency,
    })
    const isLast = items.length <= 1

    return (
        <Page title={t('currencies.title')}>
            <PageHeader
                title={t('currencies.title')}
                description={t('currencies.description')}
                createLabel={t('currencies.create')}
                onCreateClick={isReadOnly ? undefined : form.openCreate}
            />

            <div className="mx-auto w-full max-w-[800px]">
                <FeedList
                    items={items}
                    isLoading={isLoading}
                    emptyTitle={t('currencies.emptyTitle')}
                    emptyDescription={t('currencies.emptyDescription')}
                    emptyAction={
                        !isReadOnly ? (
                            <Button onClick={form.openCreate}>
                                <Plus className="size-4" />
                                {t('currencies.create')}
                            </Button>
                        ) : undefined
                    }
                    getKey={(currency) => currency.id}
                >
                    {(currency) => (
                        <CurrencyRow
                            currency={currency}
                            onEdit={form.openEdit}
                            onDelete={(id) => deleteCurrency.mutate(id)}
                            onSetBase={(id) => setBaseCurrency.mutate(id)}
                            isSettingBase={setBaseCurrency.isPending}
                            autoUpdated={settings?.auto_update_currencies ?? false}
                            isReadOnly={isReadOnly}
                            deleteDisabled={currency.isBase || isLast}
                            deleteDisabledLabel={isLast ? t('common:actions.cannotDeleteLast') : t('common:actions.delete')}
                        />
                    )}
                </FeedList>
            </div>

            <CurrencyFormDialog
                currency={form.entity}
                open={form.open}
                onOpenChange={form.setOpen}
                onSubmit={form.submit}
                isSubmitting={form.isSubmitting}
            />
        </Page>
    )
}
