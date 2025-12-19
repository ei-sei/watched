import { useRef, useState } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { authApi } from '@/api/auth'
import client from '@/api/client'
import { useToast } from '@/components/ui/Toast'

export default function Settings() {
  const { user } = useAuth()
  const { show } = useToast()
  const [displayName, setDisplayName] = useState(user?.display_name ?? '')
  const [isPublic, setIsPublic] = useState(user?.is_public ?? false)
  const [pw, setPw] = useState({ current: '', next: '' })
  const [malUsername, setMalUsername] = useState('')
  const [importing, setImporting] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  const saveProfile = async () => {
    try {
      await authApi.updateMe({ display_name: displayName })
      show('Profile updated', 'success')
    } catch {
      show('Failed to update profile', 'error')
    }
  }

  const togglePublic = async (val: boolean) => {
    setIsPublic(val)
    try {
      await authApi.updateMe({ is_public: val })
      show(val ? 'Profile is now public' : 'Profile is now private', 'success')
    } catch {
      setIsPublic(!val)
      show('Failed to update privacy', 'error')
    }
  }

  const changePassword = async (e: React.FormEvent) => {
    e.preventDefault()
    try {
      await authApi.changePassword({ current_password: pw.current, new_password: pw.next })
      show('Password changed', 'success')
      setPw({ current: '', next: '' })
    } catch {
      show('Incorrect current password', 'error')
    }
  }

  const importByUsername = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!malUsername.trim()) return
    setImporting(true)
    try {
      const { data } = await client.post('/import/mal/username', { username: malUsername.trim() })
      show(`Imported ${data.imported} anime, skipped ${data.skipped}`, 'success')
      setMalUsername('')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      show(msg ?? 'Import failed', 'error')
    } finally {
      setImporting(false)
    }
  }

  const importByFile = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setImporting(true)
    try {
      const form = new FormData()
      form.append('file', file)
      const { data } = await client.post('/import/mal/file', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      show(`Imported ${data.imported} anime, skipped ${data.skipped}`, 'success')
    } catch {
      show('Invalid MAL export file', 'error')
    } finally {
      setImporting(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const inputClass = "w-full bg-[#1a1a1a] text-zinc-200 rounded-lg px-3 py-2 text-sm border border-white/[0.08] focus:outline-none focus:border-white/20"
  const sectionClass = "bg-[#161616] rounded-xl p-5 space-y-4 ring-1 ring-white/[0.06]"
  const labelClass = "block text-xs text-zinc-500 mb-1.5"
  const btnClass = "bg-white/8 hover:bg-white/12 text-zinc-200 px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-40"

  return (
    <div className="max-w-md space-y-6">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Settings</h1>

      {/* Profile */}
      <div className={sectionClass}>
        <h2 className="text-sm font-medium text-zinc-300">Profile</h2>
        <div>
          <label className={labelClass}>Username</label>
          <p className="text-zinc-400 text-sm">{user?.username}</p>
        </div>
        <div>
          <label className={labelClass}>Display name</label>
          <input
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className={inputClass}
          />
        </div>
        <button onClick={saveProfile} className={btnClass}>Save</button>
      </div>

      {/* Privacy */}
      <div className={sectionClass}>
        <h2 className="text-sm font-medium text-zinc-300">Privacy</h2>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm text-zinc-300">Public profile</p>
            <p className="text-xs text-zinc-600 mt-0.5">
              {isPublic
                ? `Anyone can view your library at /u/${user?.username}`
                : 'Only you can see your library'}
            </p>
          </div>
          <button
            onClick={() => togglePublic(!isPublic)}
            className={`relative inline-flex h-5 w-9 flex-shrink-0 rounded-full transition-colors ${
              isPublic ? 'bg-indigo-600' : 'bg-white/10'
            }`}
          >
            <span
              className={`inline-block h-4 w-4 mt-0.5 rounded-full bg-white shadow transition-transform ${
                isPublic ? 'translate-x-4' : 'translate-x-0.5'
              }`}
            />
          </button>
        </div>
        {isPublic && (
          <div className="flex items-center gap-2">
            <code className="flex-1 text-xs text-zinc-400 bg-[#111] px-3 py-1.5 rounded-md border border-white/[0.06] truncate">
              {window.location.origin}/u/{user?.username}
            </code>
            <button
              onClick={() => {
                navigator.clipboard.writeText(`${window.location.origin}/u/${user?.username}`)
                show('Link copied', 'success')
              }}
              className={btnClass}
            >
              Copy
            </button>
          </div>
        )}
      </div>

      {/* Change password */}
      <form onSubmit={changePassword} className={sectionClass}>
        <h2 className="text-sm font-medium text-zinc-300">Change password</h2>
        <div>
          <label className={labelClass}>Current password</label>
          <input type="password" value={pw.current} onChange={(e) => setPw((p) => ({ ...p, current: e.target.value }))}
            className={inputClass} required />
        </div>
        <div>
          <label className={labelClass}>New password</label>
          <input type="password" value={pw.next} onChange={(e) => setPw((p) => ({ ...p, next: e.target.value }))}
            className={inputClass} required minLength={8} />
        </div>
        <button type="submit" className={btnClass}>Change password</button>
      </form>

      {/* MAL Import */}
      <div className={sectionClass}>
        <div>
          <h2 className="text-sm font-medium text-zinc-300">Import from MyAnimeList</h2>
          <p className="text-xs text-zinc-600 mt-1">Your anime list will be imported into the Anime section.</p>
        </div>

        {/* By username */}
        <form onSubmit={importByUsername} className="space-y-3">
          <div>
            <label className={labelClass}>MAL username</label>
            <input
              value={malUsername}
              onChange={(e) => setMalUsername(e.target.value)}
              placeholder="e.g. sheikh"
              className={inputClass}
            />
            <p className="text-xs text-zinc-700 mt-1.5">Requires MAL_CLIENT_ID to be set on the server.</p>
          </div>
          <button type="submit" disabled={importing || !malUsername.trim()} className={btnClass}>
            {importing ? 'Importing…' : 'Import by username'}
          </button>
        </form>

        <div className="flex items-center gap-3">
          <div className="flex-1 border-t border-white/[0.06]" />
          <span className="text-xs text-zinc-700">or</span>
          <div className="flex-1 border-t border-white/[0.06]" />
        </div>

        {/* By XML file */}
        <div className="space-y-2">
          <label className={labelClass}>
            Upload MAL export XML
            <span className="block text-zinc-700 font-normal mt-0.5">
              MAL → Profile → Export my List → Export Anime List
            </span>
          </label>
          <input
            ref={fileRef}
            type="file"
            accept=".xml"
            onChange={importByFile}
            disabled={importing}
            className="w-full text-xs text-zinc-500 file:mr-3 file:py-1.5 file:px-3 file:rounded-md file:border-0 file:bg-white/8 file:text-zinc-300 file:text-xs file:cursor-pointer hover:file:bg-white/12 disabled:opacity-40"
          />
        </div>
      </div>
    </div>
  )
}
