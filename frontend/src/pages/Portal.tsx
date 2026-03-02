import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { portalApi, type PortalLink } from '@/api/portal'
import { useAuth } from '@/hooks/useAuth'
import { useToast } from '@/components/ui/Toast'
import { Navigate } from 'react-router-dom'
import { Plus, Pencil, Trash2, X } from 'lucide-react'

function favicon(url: string) {
  try {
    const domain = new URL(url).hostname
    return `https://www.google.com/s2/favicons?domain=${domain}&sz=64`
  } catch {
    return null
  }
}

function StatusDot({ ok }: { ok: boolean | undefined }) {
  if (ok === undefined) return <span className="w-2 h-2 rounded-full bg-zinc-700 animate-pulse" />
  return <span className={`w-2 h-2 rounded-full ${ok ? 'bg-emerald-500' : 'bg-red-500'}`} />
}

function LinkCard({
  link,
  ok,
  onEdit,
  onDelete,
}: {
  link: PortalLink
  ok: boolean | undefined
  onEdit: (l: PortalLink) => void
  onDelete: (l: PortalLink) => void
}) {
  const icon = favicon(link.url)
  let displayUrl = link.url
  try { displayUrl = new URL(link.url).hostname } catch { /* keep original */ }

  return (
    <div className="bg-[#1a1a1a] ring-1 ring-white/[0.06] rounded-xl p-4 flex items-center gap-4 group">
      <a
        href={link.url}
        target="_blank"
        rel="noopener noreferrer"
        className="flex items-center gap-4 flex-1 min-w-0"
      >
        <div className="w-10 h-10 rounded-lg bg-white/[0.04] flex-shrink-0 flex items-center justify-center overflow-hidden">
          {icon ? (
            <img src={icon} alt="" className="w-8 h-8 object-contain" onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }} />
          ) : (
            <span className="text-zinc-600 text-xs">?</span>
          )}
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium text-zinc-200">{link.name}</p>
            <StatusDot ok={ok} />
          </div>
          <p className="text-xs text-zinc-600 truncate">{displayUrl}</p>
        </div>
      </a>
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
        <button
          onClick={() => onEdit(link)}
          className="p-1.5 text-zinc-600 hover:text-zinc-300 transition-colors rounded"
        >
          <Pencil size={13} />
        </button>
        <button
          onClick={() => onDelete(link)}
          className="p-1.5 text-zinc-600 hover:text-red-400 transition-colors rounded"
        >
          <Trash2 size={13} />
        </button>
      </div>
    </div>
  )
}

function LinkFormModal({
  initial,
  onClose,
  onSave,
  saving,
}: {
  initial?: PortalLink
  onClose: () => void
  onSave: (data: { name: string; url: string; category: string }) => void
  saving: boolean
}) {
  const [name, setName] = useState(initial?.name ?? '')
  const [url, setUrl] = useState(initial?.url ?? '')
  const [category, setCategory] = useState<string>(initial?.category ?? 'movies_tv')

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim() || !url.trim()) return
    onSave({ name: name.trim(), url: url.trim(), category })
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4" onClick={onClose}>
      <div className="bg-[#1a1a1a] rounded-xl p-6 w-full max-w-sm space-y-4 ring-1 ring-white/[0.08]" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <h2 className="text-white font-semibold">{initial ? 'Edit link' : 'Add link'}</h2>
          <button onClick={onClose} className="text-zinc-600 hover:text-zinc-400"><X size={16} /></button>
        </div>
        <form onSubmit={submit} className="space-y-3">
          <div>
            <label className="block text-xs text-zinc-600 mb-1">Name</label>
            <input
              value={name}
              onChange={e => setName(e.target.value)}
              placeholder="e.g. Crunchyroll"
              className="w-full bg-[#111] text-zinc-200 rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm"
              autoFocus
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-600 mb-1">URL</label>
            <input
              value={url}
              onChange={e => setUrl(e.target.value)}
              placeholder="https://..."
              className="w-full bg-[#111] text-zinc-200 rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm"
            />
          </div>
          <div>
            <label className="block text-xs text-zinc-600 mb-1">Category</label>
            <div className="flex gap-2">
              {(['movies_tv', 'anime'] as const).map(c => (
                <button
                  key={c}
                  type="button"
                  onClick={() => setCategory(c)}
                  className={`flex-1 py-1.5 text-xs rounded-md border transition-colors ${
                    category === c
                      ? 'bg-indigo-600 border-indigo-500 text-white'
                      : 'bg-transparent border-white/[0.08] text-zinc-500 hover:text-zinc-300'
                  }`}
                >
                  {c === 'movies_tv' ? 'Movies & TV' : 'Anime'}
                </button>
              ))}
            </div>
          </div>
          <button
            type="submit"
            disabled={saving || !name.trim() || !url.trim()}
            className="w-full py-2 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-40 text-white text-sm font-medium rounded-md transition-colors"
          >
            {saving ? 'Saving…' : initial ? 'Save changes' : 'Add link'}
          </button>
        </form>
      </div>
    </div>
  )
}

const CATEGORIES: { key: 'movies_tv' | 'anime'; label: string }[] = [
  { key: 'movies_tv', label: 'Movies & TV Shows' },
  { key: 'anime',     label: 'Anime' },
]

export default function Portal() {
  const { user } = useAuth()
  const { show } = useToast()
  const qc = useQueryClient()
  const [editing, setEditing] = useState<PortalLink | null | 'new'>(null)
  const [deleteTarget, setDeleteTarget] = useState<PortalLink | null>(null)

  const links = useQuery({
    queryKey: ['portal', 'links'],
    queryFn: () => portalApi.list().then(r => r.data),
    enabled: !!user?.is_admin,
  })

  const statuses = useQuery({
    queryKey: ['portal', 'status'],
    queryFn: () => portalApi.status().then(r => r.data),
    enabled: !!user?.is_admin && (links.data?.length ?? 0) > 0,
    staleTime: 0,
    refetchOnWindowFocus: false,
  })

  const statusMap = new Map((statuses.data ?? []).map(s => [s.id, s.ok]))

  const create = useMutation({
    mutationFn: (data: { name: string; url: string; category: string }) => portalApi.create(data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['portal'] }); setEditing(null); show('Link added', 'success') },
    onError: () => show('Failed to add link', 'error'),
  })

  const update = useMutation({
    mutationFn: ({ id, data }: { id: number; data: { name: string; url: string; category: string } }) =>
      portalApi.update(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['portal'] }); setEditing(null); show('Link updated', 'success') },
    onError: () => show('Failed to update link', 'error'),
  })

  const remove = useMutation({
    mutationFn: (id: number) => portalApi.delete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['portal'] }); setDeleteTarget(null); show('Link removed', 'success') },
    onError: () => show('Failed to remove link', 'error'),
  })

  if (!user?.is_admin) return <Navigate to="/" replace />

  const handleSave = (data: { name: string; url: string; category: string }) => {
    if (editing === 'new') create.mutate(data)
    else if (editing) update.mutate({ id: editing.id, data })
  }

  return (
    <div className="space-y-8 max-w-2xl">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-white tracking-tight">Portal</h1>
        <button
          onClick={() => setEditing('new')}
          className="flex items-center gap-1.5 px-3 py-1.5 bg-white/[0.06] hover:bg-white/[0.10] text-zinc-300 text-xs rounded-lg transition-colors"
        >
          <Plus size={13} />
          Add link
        </button>
      </div>

      {links.isLoading && <p className="text-zinc-600 text-sm">Loading…</p>}
      {links.isError && <p className="text-red-400 text-sm">Failed to load links.</p>}

      {links.data && CATEGORIES.map(({ key, label }) => {
        const items = links.data.filter(l => l.category === key)
        if (!items.length) return null
        return (
          <div key={key} className="space-y-3">
            <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">{label}</p>
            <div className="space-y-2">
              {items.map(l => (
                <LinkCard
                  key={l.id}
                  link={l}
                  ok={statusMap.get(l.id)}
                  onEdit={setEditing}
                  onDelete={setDeleteTarget}
                />
              ))}
            </div>
          </div>
        )
      })}

      {links.data?.length === 0 && (
        <p className="text-zinc-600 text-sm text-center py-12">No links yet. Add one to get started.</p>
      )}

      {editing !== null && (
        <LinkFormModal
          initial={editing === 'new' ? undefined : editing}
          onClose={() => setEditing(null)}
          onSave={handleSave}
          saving={create.isPending || update.isPending}
        />
      )}

      {deleteTarget && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
          <div className="bg-[#1a1a1a] rounded-xl p-6 w-full max-w-sm space-y-4 ring-1 ring-white/[0.08]">
            <h2 className="text-white font-semibold">Remove link</h2>
            <p className="text-zinc-400 text-sm">Remove <span className="text-white font-medium">{deleteTarget.name}</span> from Portal?</p>
            <div className="flex gap-2 justify-end">
              <button onClick={() => setDeleteTarget(null)} className="px-4 py-2 text-sm text-zinc-400 hover:text-white transition-colors">Cancel</button>
              <button
                onClick={() => remove.mutate(deleteTarget.id)}
                disabled={remove.isPending}
                className="px-4 py-2 text-sm bg-red-600 hover:bg-red-700 disabled:opacity-40 text-white rounded-md transition-colors"
              >
                Remove
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
