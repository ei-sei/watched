import { useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { useAuth } from '@/hooks/useAuth'

export default function Login() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [form, setForm] = useState({ username: '', password: '' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await login(form.username, form.password)
      navigate('/')
    } catch {
      setError('Invalid username or password')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-indigo-400 text-center mb-8">watched</h1>
        <form onSubmit={handleSubmit} className="bg-slate-900 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">Sign in</h2>
          {error && <p className="text-red-400 text-sm">{error}</p>}
          <div>
            <label className="block text-sm text-slate-400 mb-1">Username</label>
            <input value={form.username} onChange={(e) => setForm((f) => ({ ...f, username: e.target.value }))}
              className="w-full bg-slate-800 text-white rounded px-3 py-2 text-sm border border-slate-700 focus:outline-none focus:border-indigo-500" required />
          </div>
          <div>
            <label className="block text-sm text-slate-400 mb-1">Password</label>
            <input type="password" value={form.password} onChange={(e) => setForm((f) => ({ ...f, password: e.target.value }))}
              className="w-full bg-slate-800 text-white rounded px-3 py-2 text-sm border border-slate-700 focus:outline-none focus:border-indigo-500" required />
          </div>
          <button type="submit" disabled={loading}
            className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white py-2 rounded font-medium text-sm transition-colors">
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
          <p className="text-center text-sm text-slate-500">
            Have an invite? <Link to="/register" className="text-indigo-400 hover:underline">Register</Link>
          </p>
        </form>
      </div>
    </div>
  )
}
