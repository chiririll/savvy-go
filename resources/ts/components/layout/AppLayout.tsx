import { Outlet } from 'react-router-dom'
import { AppSidebar } from './Sidebar'
import { Header } from './Header'
import { StatusBar } from './StatusBar'
import { SidebarProvider, SidebarInset } from '@/components/ui/sidebar'
import { CreateTransactionProvider } from '@/components/features/transactions'
import { useCurrencies } from '@/hooks/use-currencies'
import { useUiStore } from '@/stores/ui'

export function AppLayout() {
    useCurrencies()
    const sidebarOpen = useUiStore((state) => state.sidebarOpen)
    const setSidebarOpen = useUiStore((state) => state.setSidebarOpen)

    return (
        <SidebarProvider open={sidebarOpen} onOpenChange={setSidebarOpen}>
            <AppSidebar />
            <SidebarInset className="min-w-0">
                <CreateTransactionProvider>
                    <div className="sticky top-0 z-50 min-w-0">
                        <StatusBar />
                        <Header />
                    </div>
                    <main className="min-w-0 flex-1 overflow-y-auto px-3 py-4 sm:p-6">
                        <Outlet />
                    </main>
                </CreateTransactionProvider>
            </SidebarInset>
        </SidebarProvider>
    )
}
