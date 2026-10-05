import { NavLink } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { userTabLabel, type UserTab } from '@/lib/labels'

const tabs: { to: string; key: UserTab }[] = [
    { to: '/settings/user', key: 'preferences' },
    { to: '/settings/security', key: 'security' },
    { to: '/settings/api', key: 'api' },
]

/** Tabs across the user's own settings pages. */
export function UserSettingsTabs() {
    const { t } = useTranslation('settings')
    return (
        <nav className="mb-6 inline-flex rounded-lg border bg-muted/40 p-1 text-sm">
            {tabs.map((tab) => (
                <NavLink
                    key={tab.to}
                    to={tab.to}
                    end
                    className={({ isActive }) =>
                        cn(
                            'rounded-md px-3 py-1.5 font-medium text-muted-foreground transition-colors hover:text-foreground',
                            isActive && 'bg-background text-foreground shadow-sm',
                        )
                    }
                >
                    {userTabLabel(t, tab.key)}
                </NavLink>
            ))}
        </nav>
    )
}
