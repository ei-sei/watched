import { useState } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { authApi } from '@/api/auth'

export default function Register() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const [form, setForm] = useState({
    username: '', password: '', invite_code: params.get('invite') ?? '',
  })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await authApi.register(form)
      navigate('/login')
    } catch (err: unknown) {
      const detail = (err as { response?: { data?: { detail?: string } } })?.response?.data?.detail
      setError(typeof detail === 'string' ? detail : 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-indigo-400 text-center mb-8">watched</h1>
        <form onSubmit={handleSubmit} className="bg-slate-900 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">Create account</h2>
          {error && <p className="text-red-400 text-sm">{error}</p>}
          {(['username', 'password', 'invite_code'] as const).map((field) => (
            <div key={field}>
              <label className="block text-sm text-slate-400 mb-1 capitalize">{field.replace('_', ' ')}</label>
              <input type={field === 'password' ? 'password' : 'text'}
                value={form[field]} onChange={(e) => setForm((f) => ({ ...f, [field]: e.target.value }))}
                className="w-full bg-slate-800 text-white rounded px-3 py-2 text-sm border border-slate-700 focus:outline-none focus:border-indigo-500"
                required />
            </div>
          ))}
          <button type="submit" disabled={loading}
            className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white py-2 rounded font-medium text-sm transition-colors">
            {loading ? 'Creating account…' : 'Create account'}
          </button>
          <p className="text-center text-sm text-slate-500">
            Already registered? <Link to="/login" className="text-indigo-400 hover:underline">Sign in</Link>
          </p>
        </form>
      </div>
    </div>
  )
}
