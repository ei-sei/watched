import { NavLink, useNavigate } from 'react-router-dom'
import { Film, Tv, BookOpen, Sparkles, Search, BarChart2, Settings, LogOut } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'

export const PRIMARY_NAV = [
  { to: '/films',  label: 'Movies',   icon: Film },
  { to: '/tv',     label: 'TV Shows', icon: Tv },
  { to: '/books',  label: 'Books',    icon: BookOpen },
  { to: '/anime',  label: 'Anime',    icon: Sparkles },
]

const SECONDARY_NAV = [
  { to: '/search', label: 'Search', icon: Search },
  { to: '/stats',  label: 'Stats',  icon: BarChart2 },
]

const linkClass = ({ isActive }: { isActive: boolean }) =>
  `flex items-center gap-2.5 px-3 py-1.5 rounded-md text-sm w-full transition-colors ${
    isActive
      ? 'bg-white/10 text-white font-medium'
      : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-200'
  }`

export default function Sidebar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <aside className="fixed inset-y-0 left-0 w-52 flex flex-col bg-[#111111] border-r border-white/[0.06] z-20 select-none">
      {/* Workspace */}
      <div className="px-4 h-12 flex items-center">
        <span className="text-white font-semibold text-sm tracking-tight">watched</span>
      </div>

      {/* Primary nav */}
      <nav className="flex-1 px-2 pt-1 space-y-px overflow-y-auto">
        {PRIMARY_NAV.map(({ to, label, icon: Icon }) => (
          <NavLink key={to} to={to} className={linkClass}>
            <Icon size={15} strokeWidth={1.75} />
            {label}
          </NavLink>
        ))}

        <div className="py-2">
          <div className="border-t border-white/[0.06]" />
        </div>

        {SECONDARY_NAV.map(({ to, label, icon: Icon }) => (
          <NavLink key={to} to={to} className={linkClass}>
            <Icon size={15} strokeWidth={1.75} />
            {label}
          </NavLink>
        ))}
      </nav>

      {/* Bottom */}
      <div className="px-2 pb-3 space-y-px">
        <div className="border-t border-white/[0.06] mb-2" />
        <NavLink to="/settings" className={linkClass}>
          <Settings size={15} strokeWidth={1.75} />
          Settings
        </NavLink>
        <button
          onClick={handleLogout}
          className="flex items-center gap-2.5 px-3 py-1.5 rounded-md text-sm text-zinc-500 hover:bg-white/5 hover:text-zinc-200 transition-colors w-full"
        >
          <LogOut size={15} strokeWidth={1.75} />
          Sign out
        </button>
        <div className="px-3 pt-2">
          <p className="text-xs text-zinc-600 truncate">{user?.display_name ?? user?.username}</p>
        </div>
      </div>
    </aside>
  )
}
