import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import BottomNav from './BottomNav'
import SyncIndicator from '@/components/ui/SyncIndicator'
import { VERSION } from '@/version'

export default function Layout() {
  return (
    <div className="flex min-h-screen bg-[#0d0d0d] text-zinc-200">
      {/* Desktop sidebar */}
      <div className="hidden md:block">
        <Sidebar />
      </div>

      {/* Main content */}
      <main className="flex-1 md:ml-52 min-h-screen pb-20 md:pb-0">
        <div className="max-w-5xl mx-auto px-4 md:px-10 py-6 md:py-10">
          <Outlet />
        </div>
      </main>

      {/* Mobile bottom nav */}
      <BottomNav />

      {/* Version + sync status */}
      <div className="fixed bottom-[4.5rem] md:bottom-2 right-2 flex items-center gap-2 pointer-events-none">
        <SyncIndicator />
        <span className="text-[10px] text-zinc-700 select-none">
          v{VERSION}
        </span>
      </div>
    </div>
  )
}
