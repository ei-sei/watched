import { useRef, useState, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useAuth } from '@/hooks/useAuth'
import { authApi } from '@/api/auth'
import client from '@/api/client'
import { useToast } from '@/components/ui/Toast'
import { db } from '@/offline/db'
import type { MediaItem } from '@/types/media'

// ── Import card ────────────────────────────────────────────────────────────────

type ImportService = {
  id: string
  name: string
  subtitle: string
  accept: string
  hint: string
  endpoint: string
}

const IMPORT_SERVICES: ImportService[] = [
  {
    id: 'mal',
    name: 'MyAnimeList',
    subtitle: 'Import your anime list',
    accept: '.xml',
    hint: 'MAL → Profile → Export My List → Export Anime List',
    endpoint: '/import/mal',
  },
  {
    id: 'letterboxd',
    name: 'Letterboxd',
    subtitle: 'Import films you have watched or want to watch',
    accept: '.csv',
    hint: 'Letterboxd → Profile → Import & Export → Export Your Data',
    endpoint: '/import/letterboxd',
  },
  {
    id: 'goodreads',
    name: 'Goodreads',
    subtitle: 'Import your book library',
    accept: '.csv',
    hint: 'Goodreads → My Books → Import and Export → Export Library',
    endpoint: '/import/goodreads',
  },
]

function useImportProgress(importing: boolean) {
  const [progress, setProgress] = useState(0)
  const startRef = useRef<number | null>(null)

  useEffect(() => {
    if (!importing) return
    startRef.current = Date.now()
    // Asymptotically approach 90% so it never appears stuck
    const id = setInterval(() => {
      const elapsed = Date.now() - (startRef.current ?? Date.now())
      setProgress(90 * (1 - Math.exp(-elapsed / 5000)))
    }, 80)
    return () => {
      clearInterval(id)
      setProgress(0)
    }
  }, [importing])

  return progress
}

function ImportCard() {
  const { show } = useToast()
  const fileRefs = useRef<Record<string, HTMLInputElement | null>>({})
  const [importing, setImporting] = useState(false)
  const [activeService, setActiveService] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<{ service: string; imported: number; skipped: number } | null>(null)
  const [missingCount, setMissingCount] = useState<number | null>(null)
  const [refetching, setRefetching] = useState(false)
  const [posterProgress, setPosterProgress] = useState<{ current: number; total: number } | null>(null)
  const [allCaughtUp, setAllCaughtUp] = useState(false)
  const rawProgress = useImportProgress(importing)
  const displayProgress = importing ? rawProgress : lastResult ? 100 : 0

  useEffect(() => {
    client.get('/import/posters/missing-count')
      .then(({ data }) => setMissingCount(data.count))
      .catch(() => {})
  }, [])

  const refetchPosters = async () => {
    setRefetching(true)
    setPosterProgress(null)
    setAllCaughtUp(false)
    try {
      const { data } = await client.post('/import/posters/refetch')
      if (data.queued === 0) {
        setMissingCount(0)
        setAllCaughtUp(true)
        setRefetching(false)
        return
      }
      const total: number = data.queued
      // Backend processes at 700ms/item — match the timer so UI doesn't
      // finish before the goroutine does, causing a stale count re-check.
      const msPerItem = 700
      const startedAt = Date.now()
      setMissingCount(0) // optimistically clear while fetch runs
      setPosterProgress({ current: 0, total })
      const id = setInterval(async () => {
        const elapsed = Date.now() - startedAt
        const estimated = Math.min(Math.floor(elapsed / msPerItem), total)
        setPosterProgress({ current: estimated, total })
        if (estimated >= total) {
          clearInterval(id)
          // Add a small buffer then verify with server how many are still missing
          await new Promise(r => setTimeout(r, 1500))
          try {
            const { data: check } = await client.get('/import/posters/missing-count')
            setMissingCount(check.count)
            if (check.count === 0) setAllCaughtUp(true)
          } catch { /* ignore */ }
          setRefetching(false)
        }
      }, msPerItem)
    } catch {
      show('Failed to start poster fetch', 'error')
      setRefetching(false)
    }
  }

  const handleFile = async (service: ImportService, file: File) => {
    setImporting(true)
    setActiveService(service.id)
    setLastResult(null)

    const form = new FormData()
    form.append('file', file)

    try {
      const { data } = await client.post(service.endpoint, form, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setLastResult({ service: service.name, imported: data.imported, skipped: data.skipped })
      show(`Imported ${data.imported}, skipped ${data.skipped}`, 'success')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })?.response?.data?.error
      show(msg ?? 'Import failed', 'error')
      setLastResult(null)
    } finally {
      setImporting(false)
      setActiveService(null)
      const ref = fileRefs.current[service.id]
      if (ref) ref.value = ''
    }
  }

  const sectionClass = 'bg-[#161616] rounded-xl p-5 space-y-4 ring-1 ring-white/[0.06]'

  return (
    <div className={sectionClass}>
      <div>
        <h2 className="text-sm font-medium text-zinc-300">Import library</h2>
        <p className="text-xs text-zinc-600 mt-1">Upload an export file to add entries to your library.</p>
      </div>

      <div className="space-y-4">
        {IMPORT_SERVICES.map((svc, i) => (
          <div key={svc.id}>
            {i > 0 && <div className="border-t border-white/[0.05] mb-4" />}
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-sm text-zinc-200 font-medium">{svc.name}</p>
                <p className="text-xs text-zinc-500 mt-0.5">{svc.subtitle}</p>
                <p className="text-xs text-zinc-700 mt-1">{svc.hint}</p>
              </div>
              <label
                className={`flex-shrink-0 cursor-pointer px-3 py-1.5 text-xs rounded-md text-zinc-300 bg-white/8 hover:bg-white/12 transition-colors ${importing ? 'opacity-40 pointer-events-none' : ''}`}
              >
                {activeService === svc.id ? 'Uploading…' : `Upload ${svc.accept.replace('.', '').toUpperCase()}`}
                <input
                  ref={(el) => { fileRefs.current[svc.id] = el }}
                  type="file"
                  accept={svc.accept}
                  className="sr-only"
                  disabled={importing}
                  onChange={(e) => {
                    const file = e.target.files?.[0]
                    if (file) handleFile(svc, file)
                  }}
                />
              </label>
            </div>
          </div>
        ))}
      </div>

      <div className="border-t border-white/[0.05] pt-3 space-y-2">
        <div className="flex items-center justify-between gap-3">
          <div>
            <p className="text-sm text-zinc-300">Fetch missing posters</p>
            <p className="text-xs text-zinc-600 mt-0.5">
              {allCaughtUp
                ? 'All posters are up to date.'
                : missingCount === null
                  ? 'Re-runs poster lookup for anime imported without one.'
                  : missingCount === 0
                    ? 'All posters are up to date.'
                    : `${missingCount} anime ${missingCount === 1 ? 'is' : 'are'} missing a poster.`}
            </p>
          </div>
          <button
            onClick={refetchPosters}
            disabled={refetching || importing || allCaughtUp || missingCount === 0}
            className="flex-shrink-0 px-3 py-1.5 text-xs rounded-md text-zinc-300 bg-white/8 hover:bg-white/12 transition-colors disabled:opacity-40"
          >
            {refetching ? 'Fetching…' : 'Fetch'}
          </button>
        </div>
        {posterProgress && (
          <div className="space-y-1">
            <div className="w-full bg-white/[0.06] rounded-full h-1.5 overflow-hidden">
              <div
                className="h-full bg-indigo-500 rounded-full transition-all duration-300"
                style={{ width: `${(posterProgress.current / posterProgress.total) * 100}%` }}
              />
            </div>
            <p className="text-xs text-zinc-500">
              {posterProgress.current < posterProgress.total
                ? `${posterProgress.current} / ${posterProgress.total} posters fetched…`
                : `Done — fetched ${posterProgress.total} posters`}
            </p>
          </div>
        )}
      </div>

      {/* Progress bar — shown while importing or after completion */}
      {(importing || lastResult) && (
        <div className="space-y-1.5 pt-1">
          <div className="w-full bg-white/[0.06] rounded-full h-1.5 overflow-hidden">
            <div
              className="h-full bg-indigo-500 rounded-full transition-all duration-200"
              style={{ width: `${displayProgress}%` }}
            />
          </div>
          <div className="flex justify-between items-center">
            <span className="text-xs text-zinc-500">
              {importing
                ? 'Importing…'
                : lastResult
                  ? `${lastResult.imported} imported, ${lastResult.skipped} skipped`
                  : ''}
            </span>
            {!importing && (
              <span className="text-xs text-zinc-600">{lastResult?.service}</span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

// ── Danger zone ────────────────────────────────────────────────────────────────

type ClearTarget = {
  type: MediaItem['media_type']
  label: string
}

const CLEAR_TARGETS: ClearTarget[] = [
  { type: 'film',    label: 'Films' },
  { type: 'tv_show', label: 'TV Shows' },
  { type: 'book',    label: 'Books' },
  { type: 'anime',   label: 'Anime' },
]

function ConfirmClearModal({ label, onConfirm, onCancel }: {
  label: string
  onConfirm: () => void
  onCancel: () => void
}) {
  const [input, setInput] = useState('')
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
      <div className="bg-[#1a1a1a] rounded-xl p-6 w-full max-w-sm space-y-4 ring-1 ring-white/[0.08]">
        <h2 className="text-white font-semibold">Clear {label}</h2>
        <p className="text-zinc-400 text-sm">
          This will permanently delete all{' '}
          <span className="text-white font-medium">{label}</span> from your library. This cannot be undone.
        </p>
        <div>
          <p className="text-zinc-500 text-xs mb-1.5">
            Type <span className="text-white font-mono">Delete</span> to confirm
          </p>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            className="w-full bg-[#111] text-zinc-200 rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-red-500/50 text-sm font-mono"
            placeholder="Delete"
            autoFocus
          />
        </div>
        <div className="flex gap-2 justify-end">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm text-zinc-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={input !== 'Delete'}
            className="px-4 py-2 text-sm bg-red-600 hover:bg-red-700 disabled:opacity-40 text-white rounded-md transition-colors"
          >
            Delete all
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Main page ──────────────────────────────────────────────────────────────────

export default function Settings() {
  const { user } = useAuth()
  const { show } = useToast()
  const qc = useQueryClient()
  const [displayName, setDisplayName] = useState(user?.display_name ?? '')
  const [isPublic, setIsPublic] = useState(user?.is_public ?? false)
  const [pw, setPw] = useState({ current: '', next: '' })
  const [clearing, setClearing] = useState<MediaItem['media_type'] | null>(null)
  const [confirmTarget, setConfirmTarget] = useState<ClearTarget | null>(null)

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

  const clearList = async (target: ClearTarget) => {
    setClearing(target.type)
    setConfirmTarget(null)
    try {
      const { data } = await client.delete(`/media?type=${target.type}`)
      const ids = await db.mediaItems.where('media_type').equals(target.type).primaryKeys()
      await db.mediaItems.bulkDelete(ids as number[])
      qc.invalidateQueries({ queryKey: ['stats'] })
      show(`Deleted ${data.deleted} ${target.label.toLowerCase()}`, 'success')
    } catch {
      show(`Failed to clear ${target.label.toLowerCase()}`, 'error')
    } finally {
      setClearing(null)
    }
  }

  const inputClass = 'w-full bg-[#1a1a1a] text-zinc-200 rounded-lg px-3 py-2 text-sm border border-white/[0.08] focus:outline-none focus:border-white/20'
  const sectionClass = 'bg-[#161616] rounded-xl p-5 space-y-4 ring-1 ring-white/[0.06]'
  const labelClass = 'block text-xs text-zinc-500 mb-1.5'
  const btnClass = 'bg-white/8 hover:bg-white/12 text-zinc-200 px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-40'

  return (
    <>
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

        {/* Import */}
        <ImportCard />

        {/* Danger zone */}
        <div className="relative flex items-center gap-3 pt-2">
          <div className="flex-1 border-t border-red-900/40" />
          <span className="text-xs text-red-900 font-medium tracking-wide uppercase">Danger zone</span>
          <div className="flex-1 border-t border-red-900/40" />
        </div>

        <div className={sectionClass}>
          <div>
            <h2 className="text-sm font-medium text-zinc-300">Clear library</h2>
            <p className="text-xs text-zinc-600 mt-1">Permanently delete all entries from a category.</p>
          </div>
          <div className="grid grid-cols-2 gap-2">
            {CLEAR_TARGETS.map((target) => (
              <button
                key={target.type}
                onClick={() => setConfirmTarget(target)}
                disabled={clearing === target.type}
                className="flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-sm text-red-400 bg-red-500/[0.08] hover:bg-red-500/[0.14] border border-red-500/[0.15] transition-colors disabled:opacity-40"
              >
                {clearing === target.type ? 'Clearing…' : `Clear ${target.label}`}
              </button>
            ))}
          </div>
        </div>
      </div>

      {confirmTarget && (
        <ConfirmClearModal
          label={confirmTarget.label}
          onConfirm={() => clearList(confirmTarget)}
          onCancel={() => setConfirmTarget(null)}
        />
      )}
    </>
  )
}
