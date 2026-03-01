import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { Film, Tv, BookOpen, Sparkles, Search, BarChart2, TrendingUp, Settings, LogOut, ShieldCheck, Users, MoreHorizontal, X, Lock } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { VERSION } from '@/version'

const TABS = [
  { to: '/films',  label: 'Movies', icon: Film },
  { to: '/tv',     label: 'TV',     icon: Tv },
  { to: '/books',  label: 'Books',  icon: BookOpen },
  { to: '/anime',  label: 'Anime',  icon: Sparkles },
  { to: '/search', label: 'Search', icon: Search },
]

export default function BottomNav() {
  const [open, setOpen] = useState(false)
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    setOpen(false)
    await logout()
    navigate('/login')
  }

  return (
    <>
      {/* More sheet backdrop */}
      {open && (
        <div
          className="fixed inset-0 z-30 bg-black/50"
          onClick={() => setOpen(false)}
        />
      )}

      {/* More sheet */}
      {open && (
        <div className="fixed bottom-16 inset-x-0 z-40 mx-3 mb-1 bg-[#1a1a1a] rounded-2xl ring-1 ring-white/[0.08] overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-white/[0.06]">
            <p className="text-xs text-zinc-500 font-medium">{user?.display_name ?? user?.username}</p>
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-zinc-700">v{VERSION}</span>
              <button onClick={() => setOpen(false)} className="text-zinc-600 hover:text-zinc-300">
                <X size={16} />
              </button>
            </div>
          </div>
          <div className="p-2 space-y-0.5">
            {(user?.is_premium || user?.is_admin) ? (
              <>
                <SheetLink to="/stats" icon={BarChart2} label="Stats" onClick={() => setOpen(false)} />
                <SheetLink to="/trending" icon={TrendingUp} label="Trending" onClick={() => setOpen(false)} />
              </>
            ) : (
              <>
                <div className="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-zinc-700 select-none">
                  <BarChart2 size={16} />
                  Stats
                  <Lock size={12} className="ml-auto" />
                </div>
                <div className="flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm text-zinc-700 select-none">
                  <TrendingUp size={16} />
                  Trending
                  <Lock size={12} className="ml-auto" />
                </div>
              </>
            )}
            <SheetLink to="/settings" icon={Settings} label="Settings" onClick={() => setOpen(false)} />
            {user?.username === 'admin' && (
              <SheetLink to="/users" icon={Users} label="Users" onClick={() => setOpen(false)} />
            )}
            {user?.is_admin && (
              <SheetLink to="/admin" icon={ShieldCheck} label="Admin" onClick={() => setOpen(false)} />
            )}
            <button
              onClick={handleLogout}
              className="flex items-center gap-3 w-full px-3 py-2.5 rounded-xl text-sm text-red-400 hover:bg-white/[0.04] transition-colors"
            >
              <LogOut size={16} />
              Sign out
            </button>
          </div>
        </div>
      )}

      {/* Bottom nav bar */}
      <nav className="fixed bottom-0 inset-x-0 z-20 bg-[#111111] border-t border-white/[0.06] flex md:hidden" style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}>
        {TABS.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              `flex-1 flex flex-col items-center justify-center py-2.5 gap-1 text-[10px] transition-colors ${
                isActive ? 'text-white' : 'text-zinc-600'
              }`
            }
          >
            <Icon size={20} strokeWidth={1.75} />
            {label}
          </NavLink>
        ))}
        <button
          onClick={() => setOpen((v) => !v)}
          className={`flex-1 flex flex-col items-center justify-center py-2.5 gap-1 text-[10px] transition-colors ${open ? 'text-white' : 'text-zinc-600'}`}
        >
          <MoreHorizontal size={20} strokeWidth={1.75} />
          More
        </button>
      </nav>
    </>
  )
}

function SheetLink({ to, icon: Icon, label, onClick }: { to: string; icon: React.ElementType; label: string; onClick: () => void }) {
  return (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) =>
        `flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-colors ${
          isActive ? 'bg-white/[0.08] text-white' : 'text-zinc-400 hover:bg-white/[0.04]'
        }`
      }
    >
      <Icon size={16} />
      {label}
    </NavLink>
  )
}
