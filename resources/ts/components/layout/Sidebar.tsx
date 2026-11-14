import { useEffect } from 'react'
import { NavLink, useLocation } from 'react-router-dom'
import { Boxes, UserRound, Wrench } from 'lucide-react'
import { SpaceSwitcher } from '@/components/features/spaces/SpaceSwitcher'
import { useUser } from '@/stores/auth'
import { useCurrentSpace } from '@/hooks'
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
import { navLabel } from '@/lib/i18n/labels'
import { NavGroup } from './NavGroup'
import { SidebarAbout } from './SidebarAbout'
import { mainItems, spaceItems, accountItems, adminItems } from './nav-items'

export function AppSidebar() {
    const { t } = useTranslation('nav')
    const location = useLocation()
    const spaceOpen = useUiStore((state) => state.spaceOpen)
    const setSpaceOpen = useUiStore((state) => state.setSpaceOpen)
    const accountOpen = useUiStore((state) => state.accountOpen)
    const setAccountOpen = useUiStore((state) => state.setAccountOpen)
    const space = useCurrentSpace()
    const visibleSpaceItems = spaceItems.filter((item) => !item.spaceAdminOnly || space?.role === 'admin')
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

                {space && (
                    <NavGroup
                        icon={Boxes}
                        label={t('space')}
                        items={visibleSpaceItems}
                        open={spaceOpen}
                        onOpenChange={setSpaceOpen}
                        isActive={isActive}
                    />
                )}
                <NavGroup
                    icon={UserRound}
                    label={t('account')}
                    items={accountItems}
                    open={accountOpen}
                    onOpenChange={setAccountOpen}
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
