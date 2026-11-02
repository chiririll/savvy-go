import { SidebarTrigger } from '@/components/ui/sidebar'
import { Separator } from '@/components/ui/separator'
import { LanguageSwitcher } from '@/components/shared/appearance/LanguageSwitcher'
import { ThemeSwitcher } from '@/components/shared/appearance/ThemeSwitcher'
import { CreateTransactionMenu } from './header/CreateTransactionMenu'
import { HeaderBalance } from './header/HeaderBalance'
import { SearchButton } from './header/SearchButton'
import { UserMenu } from './header/UserMenu'

export function Header() {
    return (
        <header className="h-14 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
            <div className="flex h-full items-center justify-between px-4">
                <div className="flex items-center gap-2">
                    <SidebarTrigger />
                    <Separator orientation="vertical" className="h-4" />
                </div>

                {/* Five controls share 390px: tighter gaps below sm keep a long balance (Russian groups digits with spaces) in the window. */}
                <div className="flex min-w-0 items-center gap-1 sm:gap-3">
                    <SearchButton />
                    <CreateTransactionMenu />
                    <HeaderBalance />
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                    <UserMenu />
                </div>
            </div>
        </header>
    )
}
