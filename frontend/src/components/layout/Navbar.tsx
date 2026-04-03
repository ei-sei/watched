import { Link, NavLink, useNavigate } from 'react-router-dom'
import { Film, Tv, BookOpen, Search, BarChart2, LogOut, Settings } from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'

const NAV = [
  { to: '/films', label: 'Films', icon: Film },
  { to: '/tv', label: 'TV', icon: Tv },
  { to: '/books', label: 'Books', icon: BookOpen },
  { to: '/search', label: 'Search', icon: Search },
  { to: '/stats', label: 'Stats', icon: BarChart2 },
]

export default function Navbar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <nav className="bg-slate-900 border-b border-slate-800 px-4 h-14 flex items-center gap-4">
      <Link to="/" className="font-bold text-indigo-400 text-lg tracking-tight mr-4">watched</Link>
      <div className="flex items-center gap-1 flex-1">
        {NAV.map(({ to, label, icon: Icon }) => (
          <NavLink key={to} to={to}
            className={({ isActive }) =>
              `flex items-center gap-1.5 px-3 py-1.5 rounded text-sm transition-colors ${isActive ? 'bg-slate-700 text-white' : 'text-slate-400 hover:text-white hover:bg-slate-800'}`
            }>
            <Icon size={15} />
            <span className="hidden sm:inline">{label}</span>
          </NavLink>
        ))}
      </div>
      <div className="flex items-center gap-2">
        <span className="text-slate-400 text-sm hidden md:inline">{user?.display_name ?? user?.username}</span>
        <NavLink to="/settings" className="p-2 text-slate-400 hover:text-white rounded hover:bg-slate-800 transition-colors">
          <Settings size={16} />
        </NavLink>
        <button onClick={handleLogout} className="p-2 text-slate-400 hover:text-red-400 rounded hover:bg-slate-800 transition-colors">
          <LogOut size={16} />
        </button>
      </div>
    </nav>
  )
}
