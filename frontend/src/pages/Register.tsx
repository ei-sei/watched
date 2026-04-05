import { useState } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { authApi } from '@/api/auth'

function validate(form: { username: string; password: string; invite_code: string }) {
  const errors: Record<string, string> = {}
  if (form.username.length < 3) errors.username = 'At least 3 characters'
  else if (form.username.length > 50) errors.username = 'Max 50 characters'
  else if (!/^[a-zA-Z0-9]+$/.test(form.username)) errors.username = 'Letters and numbers only'
  if (form.password.length < 8) errors.password = 'At least 8 characters'
  if (!form.invite_code) errors.invite_code = 'Required'
  return errors
}

export default function Register() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const [form, setForm] = useState({
    username: '', password: '', invite_code: params.get('invite') ?? '',
  })
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [serverError, setServerError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleChange = (field: string, value: string) => {
    setForm((f) => ({ ...f, [field]: value }))
    // Clear field error on change
    setErrors((e) => ({ ...e, [field]: '' }))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const errs = validate(form)
    if (Object.keys(errs).length > 0) {
      setErrors(errs)
      return
    }
    setLoading(true)
    setServerError('')
    try {
      await authApi.register(form)
      navigate('/login')
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number; data?: { detail?: string } } })?.response?.status
      const detail = (err as { response?: { data?: { detail?: string } } })?.response?.data?.detail
      if (status === 403) setServerError('Invalid or already used invite code')
      else if (status === 409) setServerError('Username already taken')
      else if (status === 422) setServerError('Invalid data — check your details')
      else setServerError(typeof detail === 'string' ? detail : 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  const fields = [
    { key: 'username',    label: 'Username',    type: 'text',     hint: '3–50 characters, letters and numbers only' },
    { key: 'password',    label: 'Password',    type: 'password', hint: 'At least 8 characters' },
    { key: 'invite_code', label: 'Invite Code', type: 'text',     hint: '' },
  ] as const

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <h1 className="text-3xl font-bold text-indigo-400 text-center mb-8">watched</h1>
        <form onSubmit={handleSubmit} className="bg-slate-900 rounded-xl p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white">Create account</h2>
          {serverError && <p className="text-red-400 text-sm">{serverError}</p>}
          {fields.map(({ key, label, type, hint }) => (
            <div key={key}>
              <label className="block text-sm text-slate-400 mb-1">{label}</label>
              <input
                type={type}
                value={form[key]}
                onChange={(e) => handleChange(key, e.target.value)}
                className={`w-full bg-slate-800 text-white rounded px-3 py-2 text-sm border focus:outline-none focus:border-indigo-500 ${
                  errors[key] ? 'border-red-500' : 'border-slate-700'
                }`}
                required
              />
              {errors[key] ? (
                <p className="text-red-400 text-xs mt-1">{errors[key]}</p>
              ) : hint ? (
                <p className="text-slate-600 text-xs mt-1">{hint}</p>
              ) : null}
            </div>
          ))}
          <button
            type="submit"
            disabled={loading}
            className="w-full bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white py-2 rounded font-medium text-sm transition-colors"
          >
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
