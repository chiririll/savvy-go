import { testId, testIdMenu } from '@/lib/test-id'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useLocation, useNavigate } from 'react-router-dom'
import { Check, ChevronsUpDown, Plus, Settings2 } from 'lucide-react'
import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuLabel,
    DropdownMenuSeparator,
    DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from '@/components/ui/sidebar'
import { pathAfterSwitch, useCurrentSpace, useSpaces, useSwitchSpace } from '@/hooks/use-spaces'
import { useUser } from '@/stores/auth'
import { CreateSpaceDialog } from './CreateSpaceDialog'
import { spaceRoleLabel } from '@/lib/labels'

function initials(name: string) {
    return name
        .split(/\s+/)
        .filter(Boolean)
        .slice(0, 2)
        .map((part) => part[0]?.toUpperCase())
        .join('') || '?'
}

/** The current space at the top of the sidebar, and the list to switch to. */
export function SpaceSwitcher() {
    const { t } = useTranslation('settings')
    const navigate = useNavigate()
    const location = useLocation()
    const user = useUser()
    const { isMobile } = useSidebar()
    const { data: spaces = [] } = useSpaces()
    const current = useCurrentSpace()
    const switchSpace = useSwitchSpace()
    const [creating, setCreating] = useState(false)

    const choose = async (id: number) => {
        if (id !== current?.id) {
            // Stay on this page; its data comes again from the new space.
            navigate(pathAfterSwitch(location.pathname), { replace: true })
            await switchSpace(id)
        }
    }

    return (
        <>
            <SidebarMenu>
                <SidebarMenuItem>
                    <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                            <SidebarMenuButton
                                size="lg"
                                {...testIdMenu('layout-space')}
                                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                            >
                                <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-primary text-sm font-semibold text-primary-foreground">
                                    {initials(current?.name ?? '')}
                                </div>
                                <div className="grid flex-1 text-left text-sm leading-tight">
                                    <span className="truncate font-semibold">{current?.name}</span>
                                    <span className="truncate text-xs text-muted-foreground">
                                        {current ? spaceRoleLabel(t, current.role) : ''}
                                    </span>
                                </div>
                                <ChevronsUpDown className="ml-auto size-4" />
                            </SidebarMenuButton>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent
                            className="min-w-56 rounded-lg"
                            align="start"
                            side={isMobile ? 'bottom' : 'right'}
                            sideOffset={4}
                        >
                            <DropdownMenuLabel className="text-xs text-muted-foreground">{t('spaces.switcher.label')}</DropdownMenuLabel>
                            {spaces.map((space) => (
                                <DropdownMenuItem key={space.id} onClick={() => choose(space.id)} className="gap-2 p-2">
                                    <div className="flex size-6 items-center justify-center rounded-md border text-[10px] font-semibold">
                                        {initials(space.name)}
                                    </div>
                                    <div className="grid flex-1 leading-tight">
                                        <span className="truncate">{space.name}</span>
                                        <span className="truncate text-xs text-muted-foreground">{spaceRoleLabel(t, space.role)}</span>
                                    </div>
                                    {space.id === current?.id && <Check className="size-4" />}
                                </DropdownMenuItem>
                            ))}
                            <DropdownMenuSeparator />
                            <DropdownMenuItem onClick={() => navigate('/settings/space')} className="gap-2 p-2">
                                <Settings2 className="size-4" />
                                {t('spaces.switcher.settings')}
                            </DropdownMenuItem>
                            {user?.role !== 'guest' && (
                                <DropdownMenuItem onClick={() => setCreating(true)} className="gap-2 p-2" {...testId('layout-space-create')}>
                                    <Plus className="size-4" />
                                    {t('spaces.switcher.create')}
                                </DropdownMenuItem>
                            )}
                        </DropdownMenuContent>
                    </DropdownMenu>
                </SidebarMenuItem>
            </SidebarMenu>
            <CreateSpaceDialog open={creating} onOpenChange={setCreating} />
        </>
    )
}
