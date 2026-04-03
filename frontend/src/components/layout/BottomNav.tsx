import { NavLink } from 'react-router-dom'
import { Film, Tv, BookOpen, Sparkles, Search } from 'lucide-react'

const TABS = [
  { to: '/films',  label: 'Movies',   icon: Film },
  { to: '/tv',     label: 'TV',       icon: Tv },
  { to: '/books',  label: 'Books',    icon: BookOpen },
  { to: '/anime',  label: 'Anime',    icon: Sparkles },
  { to: '/search', label: 'Search',   icon: Search },
]

export default function BottomNav() {
  return (
    <nav className="fixed bottom-0 inset-x-0 z-20 bg-[#111111] border-t border-white/[0.06] flex md:hidden safe-area-pb">
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
    </nav>
  )
}
