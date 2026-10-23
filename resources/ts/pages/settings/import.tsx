import { useTranslation } from 'react-i18next'
import { Page, PageHeader } from '@/components/shared'
import { CsvImportWizard } from '@/components/features/import/CsvImportWizard'
import { useReadOnly } from '@/components/providers/ReadOnlyProvider'
import { Card, CardContent } from '@/components/ui/card'
import { ShieldAlert } from 'lucide-react'

export default function ImportSettingsPage() {
    const { t } = useTranslation('settings')
    const isReadOnly = useReadOnly()

    return (
        <Page title={t('import.title')}>
            <PageHeader
                title={t('import.title')}
                description={t('import.description')}
            />
            {isReadOnly ? (
                <Card>
                    <CardContent className="flex flex-col items-center justify-center py-12 text-center">
                        <ShieldAlert className="size-12 text-muted-foreground mb-4" />
                        <h2 className="text-lg font-medium mb-2">{t('import.readOnlyTitle')}</h2>
                        <p className="text-muted-foreground max-w-md">
                            {t('import.readOnlyDescription')}
                        </p>
                    </CardContent>
                </Card>
            ) : (
                <CsvImportWizard />
            )}
        </Page>
    )
}
