import { useEffect } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Settings, Wrench } from 'lucide-react'
import { SpaceSwitcher } from '@/components/features/spaces/SpaceSwitcher'
import { useUser } from '@/stores/auth'
import { useServerChecks } from '@/hooks/use-server-checks'
import {
    Sidebar,
    SidebarContent,
    SidebarGroup,
    SidebarGroupContent,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarRail,
    useSidebar,
} from '@/components/ui/sidebar'
import { useUiStore } from '@/stores/ui'
import { useTranslation } from 'react-i18next'
import { navLabel } from '@/lib/labels'
import { NavGroup } from './NavGroup'
import { SidebarAbout } from './SidebarAbout'
import { mainItems, settingsItems, adminItems } from './nav-items'

export function AppSidebar() {
    const { t } = useTranslation('nav')
    const location = useLocation()
    const settingsOpen = useUiStore((state) => state.settingsOpen)
    const setSettingsOpen = useUiStore((state) => state.setSettingsOpen)
    const adminOpen = useUiStore((state) => state.adminOpen)
    const setAdminOpen = useUiStore((state) => state.setAdminOpen)
    const isServerAdmin = useUser()?.role === 'admin'
    const { data: checks } = useServerChecks(isServerAdmin)
    const problems = checks?.filter((c) => c.severity !== 'info') ?? []
    const { setOpenMobile } = useSidebar()

    // Close mobile sidebar when navigating to a new page
    useEffect(() => {
        setOpenMobile(false)
    }, [location.pathname, setOpenMobile])

    const isActive = (path: string, also: string[] = []) => {
        if (path === '/') return location.pathname === '/'
        return [path, ...also].some((p) => location.pathname.startsWith(p))
    }

    return (
        <Sidebar collapsible="icon">
            <SidebarHeader>
                <SpaceSwitcher />
            </SidebarHeader>

            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupLabel>{t('menu')}</SidebarGroupLabel>
                    <SidebarGroupContent>
                        <SidebarMenu>
                            {mainItems.map(({ to, icon: Icon, labelKey }) => {
                                const label = navLabel(t, labelKey)
                                return (
                                    <SidebarMenuItem key={to}>
                                        <SidebarMenuButton asChild isActive={isActive(to)} tooltip={label}>
                                            <NavLink to={to}>
                                                <Icon />
                                                <span>{label}</span>
                                            </NavLink>
                                        </SidebarMenuButton>
                                    </SidebarMenuItem>
                                )
                            })}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>

                <NavGroup
                    icon={Settings}
                    label={t('settings')}
                    items={settingsItems}
                    open={settingsOpen}
                    onOpenChange={setSettingsOpen}
                    isActive={isActive}
                />
                {isServerAdmin && (
                    <NavGroup
                        icon={Wrench}
                        label={t('administration')}
                        items={adminItems}
                        counts={{ '/admin/checks': problems.length }}
                        alert={
                            problems.length === 0
                                ? undefined
                                : problems.some((c) => c.severity === 'error')
                                  ? 'error'
                                  : 'warning'
                        }
                        open={adminOpen}
                        onOpenChange={setAdminOpen}
                        isActive={isActive}
                    />
                )}
            </SidebarContent>

            <SidebarAbout />

            <SidebarRail />
        </Sidebar>
    )
}
