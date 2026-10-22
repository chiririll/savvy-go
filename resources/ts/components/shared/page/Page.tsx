import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'

interface PageProps {
    title: string
    children: React.ReactNode
}

export function Page({ title, children }: PageProps) {
    const { t } = useTranslation()

    useEffect(() => {
        const appName = t('appName')
        document.title = title ? `${title} | ${appName}` : appName
        return () => {
            document.title = appName
        }
    }, [title, t])

    return <>{children}</>
}
