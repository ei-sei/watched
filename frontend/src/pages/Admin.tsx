import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminApi, type AdminStats, type AdminUser, type AdminUserStats, type ServiceStatus, type FeatureFlag } from '@/api/admin'
import { useAuth } from '@/hooks/useAuth'
import { useToast } from '@/components/ui/Toast'
import { formatDate } from '@/utils/formatters'
import { Copy, RefreshCw, Trash2, ChevronDown, ChevronUp, KeyRound } from 'lucide-react'

type Tab = 'invites' | 'users' | 'flags'

const FLAG_LABELS: Record<string, string> = {
  stats: 'Stats',
  trending: 'Trending',
  portal: 'Portal',
}

function FeatureFlagsTab({ flags, onToggle }: { flags: FeatureFlag[]; onToggle: (key: string, isPremium: boolean) => void }) {
  return (
    <div className="space-y-2">
      {flags.map((f) => (
        <div key={f.key} className="bg-[#1a1a1a] ring-1 ring-white/[0.06] rounded-xl px-4 py-3 flex items-center justify-between">
          <span className="text-sm text-zinc-200 font-medium">{FLAG_LABELS[f.key] ?? f.key}</span>
          <div className="flex gap-1.5">
            <button
              onClick={() => onToggle(f.key, false)}
              className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                !f.is_premium ? 'bg-emerald-600 text-white' : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
              }`}
            >
              Free
            </button>
            <button
              onClick={() => onToggle(f.key, true)}
              className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                f.is_premium ? 'bg-indigo-600 text-white' : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
              }`}
            >
              Premium
            </button>
          </div>
        </div>
      ))}
    </div>
  )
}

function randomCode() {
  return Math.random().toString(36).slice(2, 10).toUpperCase() +
         Math.random().toString(36).slice(2, 6).toUpperCase()
}

function ConfirmDeleteModal({ username, onConfirm, onCancel }: {
  username: string
  onConfirm: () => void
  onCancel: () => void
}) {
  const [input, setInput] = useState('')
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
      <div className="bg-[#1a1a1a] rounded-xl p-6 w-full max-w-sm space-y-4 ring-1 ring-white/[0.08]">
        <h2 className="text-white font-semibold">Delete user</h2>
        <p className="text-zinc-400 text-sm">
          This will permanently delete <span className="text-white font-medium">{username}</span> and all their data. This cannot be undone.
        </p>
        <div>
          <p className="text-zinc-500 text-xs mb-1.5">Type <span className="text-white font-mono">{username}</span> to confirm</p>
          <input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            className="w-full bg-[#111] text-zinc-200 rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-red-500/50 text-sm font-mono"
            placeholder={username}
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
            disabled={input !== username}
            className="px-4 py-2 text-sm bg-red-600 hover:bg-red-700 disabled:opacity-40 text-white rounded-md transition-colors"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  )
}

function ResetPasswordModal({ username, onConfirm, onCancel, isPending }: {
  username: string
  onConfirm: (password: string) => void
  onCancel: () => void
  isPending: boolean
}) {
  const [password, setPassword] = useState('')
  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 px-4">
      <div className="bg-[#1a1a1a] rounded-xl p-6 w-full max-w-sm space-y-4 ring-1 ring-white/[0.08]">
        <h2 className="text-white font-semibold">Reset password</h2>
        <p className="text-zinc-400 text-sm">Set a new password for <span className="text-white font-medium">{username}</span>.</p>
        <input
          type="password"
          value={password}
          onChange={e => setPassword(e.target.value)}
          placeholder="New password (min 8 chars)"
          className="w-full bg-[#111] text-zinc-200 rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm"
          autoFocus
        />
        <div className="flex gap-2 justify-end">
          <button onClick={onCancel} className="px-4 py-2 text-sm text-zinc-400 hover:text-white transition-colors">Cancel</button>
          <button
            onClick={() => onConfirm(password)}
            disabled={password.length < 8 || isPending}
            className="px-4 py-2 text-sm bg-indigo-600 hover:bg-indigo-700 disabled:opacity-40 text-white rounded-md transition-colors"
          >
            {isPending ? 'Saving…' : 'Reset password'}
          </button>
        </div>
      </div>
    </div>
  )
}

function fmtBytes(bytes: number) {
  if (bytes <= 0) return '0 B'
  if (bytes >= 1024 ** 3) return `${(bytes / 1024 ** 3).toFixed(2)} GB`
  if (bytes >= 1024 ** 2) return `${(bytes / 1024 ** 2).toFixed(1)} MB`
  return `${Math.round(bytes / 1024)} KB`
}

function fmtKB(kb: number) {
  return fmtBytes(kb * 1024)
}

function UsageBar({ pct, color }: { pct: number; color: string }) {
  return (
    <div className="h-1.5 w-full bg-white/[0.06] rounded-full overflow-hidden">
      <div className={`h-full rounded-full transition-all ${color}`} style={{ width: `${Math.min(pct, 100)}%` }} />
    </div>
  )
}

function barColor(pct: number) {
  return pct >= 90 ? 'bg-red-500' : pct >= 70 ? 'bg-amber-400' : 'bg-emerald-500'
}

function OverviewCard({ s }: { s: AdminStats }) {
  const items = [
    { label: 'Users',           value: s.total_users },
    { label: 'Library items',   value: s.total_items },
    { label: 'Episodes logged', value: s.total_episodes },
    { label: 'Chapters logged', value: s.total_chapters },
    { label: 'Unused invites',  value: s.unused_invites },
  ]
  return (
    <div className="bg-[#1a1a1a] rounded-lg p-4 ring-1 ring-white/[0.06] space-y-3">
      <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">Overview</p>
      <div className="grid grid-cols-3 gap-3 sm:grid-cols-5">
        {items.map(({ label, value }) => (
          <div key={label} className="space-y-0.5">
            <p className="text-xl font-semibold text-white tabular-nums">{value}</p>
            <p className="text-xs text-zinc-600">{label}</p>
          </div>
        ))}
      </div>
    </div>
  )
}

function InfraCard({ s }: { s: AdminStats }) {
  const memPct = s.mem_total_kb > 0
    ? Math.round((s.mem_used_kb / s.mem_total_kb) * 100)
    : null

  const diskPct = s.disk_total_bytes > 0
    ? Math.round((s.disk_used_bytes / s.disk_total_bytes) * 100)
    : null

  return (
    <div className="bg-[#1a1a1a] rounded-lg p-4 ring-1 ring-white/[0.06] space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">Infrastructure</p>
        <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${s.db_healthy ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400'}`}>
          DB {s.db_healthy ? `healthy · ${s.db_size_human}` : 'error'}
        </span>
      </div>

      {/* Disk usage */}
      {diskPct !== null && (
        <div className="space-y-1.5">
          <div className="flex items-center justify-between text-xs text-zinc-500">
            <span>Disk usage</span>
            <span>{fmtBytes(s.disk_used_bytes)} / {fmtBytes(s.disk_total_bytes)} ({diskPct}%)</span>
          </div>
          <UsageBar pct={diskPct} color={barColor(diskPct)} />
        </div>
      )}

      {/* Server memory */}
      {memPct !== null && (
        <div className="space-y-1.5">
          <div className="flex items-center justify-between text-xs text-zinc-500">
            <span>Server memory</span>
            <span>{fmtKB(s.mem_used_kb)} / {fmtKB(s.mem_total_kb)} ({memPct}%)</span>
          </div>
          <UsageBar pct={memPct} color={barColor(memPct)} />
        </div>
      )}
    </div>
  )
}

function ServicesCard({ services, loading }: { services: ServiceStatus[] | undefined; loading: boolean }) {
  return (
    <div className="bg-[#1a1a1a] rounded-lg p-4 ring-1 ring-white/[0.06] space-y-3">
      <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">External APIs</p>
      {loading ? (
        <p className="text-xs text-zinc-600">Checking…</p>
      ) : !services ? (
        <p className="text-xs text-zinc-600">Failed to load</p>
      ) : (
        <div className="space-y-2">
          {services.map((s) => (
            <div key={s.name} className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className={`w-2 h-2 rounded-full flex-shrink-0 ${s.ok ? 'bg-emerald-500' : 'bg-red-500'}`} />
                <span className="text-sm text-zinc-300">{s.name}</span>
                {s.error && <span className="text-xs text-zinc-600">({s.error})</span>}
              </div>
              <span className="text-xs text-zinc-600 tabular-nums">
                {s.ok ? `${s.latency_ms} ms` : '—'}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function UserStatsPanel({ stats }: { stats: AdminUserStats }) {
  const sections = [
    { label: 'Films',    data: stats.films,    extra: null },
    { label: 'TV',       data: stats.tv_shows, extra: stats.episodes_watched > 0 ? `${stats.episodes_watched} eps` : null },
    { label: 'Anime',    data: stats.anime,    extra: null },
    { label: 'Books',    data: stats.books,    extra: stats.chapters_read > 0 ? `${stats.chapters_read} ch` : null },
  ]
  const hasAny = sections.some(s => s.data.total > 0)
  if (!hasAny) return <p className="text-xs text-zinc-600 py-1">No library items yet.</p>
  return (
    <div className="flex flex-wrap gap-2 pt-1">
      {sections.filter(s => s.data.total > 0).map(s => (
        <div key={s.label} className="flex flex-col items-center bg-white/[0.04] rounded-md px-3 py-2 min-w-[60px]">
          <span className="text-sm font-semibold text-white tabular-nums">{s.data.total}</span>
          <span className="text-[10px] text-zinc-500 leading-tight">{s.label}</span>
          {s.data.completed > 0 && (
            <span className="text-[10px] text-zinc-600 leading-tight">{s.data.completed} done</span>
          )}
          {s.extra && <span className="text-[10px] text-indigo-400 leading-tight">{s.extra}</span>}
        </div>
      ))}
    </div>
  )
}

const LIBRARY_LABELS: Record<string, string> = {
  film: 'Films', tv_show: 'TV Shows', anime: 'Anime', book: 'Books',
}
const LIBRARY_ORDER = ['film', 'tv_show', 'anime', 'book']

function UserLibraryPanel({ items }: { items: import('@/types/media').MediaItem[] }) {
  const grouped = LIBRARY_ORDER.reduce<Record<string, import('@/types/media').MediaItem[]>>((acc, type) => {
    const matching = items.filter(i => i.media_type === type)
    if (matching.length) acc[type] = matching
    return acc
  }, {})

  if (!items.length) return <p className="text-xs text-zinc-600 pt-1">Empty library.</p>

  return (
    <div className="space-y-3 pt-1">
      {Object.entries(grouped).map(([type, list]) => (
        <div key={type}>
          <p className="text-[10px] text-zinc-500 uppercase tracking-wider mb-1.5">
            {LIBRARY_LABELS[type]} ({list.length})
          </p>
          <div className="space-y-0.5">
            {list.map(item => (
              <div key={item.id} className="flex items-center justify-between gap-2">
                <p className="text-xs text-zinc-300 truncate">{item.title}{item.year ? ` (${item.year})` : ''}</p>
                <span className="text-[10px] text-zinc-600 flex-shrink-0 capitalize">{item.status.replace('_', ' ')}</span>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function UserRow({ u, isSuperAdmin, updateFlags, onDelete, onResetPassword }: {
  u: AdminUser
  isSuperAdmin: boolean
  updateFlags: (args: { id: number; flags: { is_admin?: boolean; is_premium?: boolean } }) => void
  onDelete: (u: AdminUser) => void
  onResetPassword: (u: AdminUser) => void
}) {
  const [open, setOpen] = useState(false)
  const [showLibrary, setShowLibrary] = useState(false)

  const stats = useQuery({
    queryKey: ['admin', 'user-stats', u.id],
    queryFn: () => adminApi.getUserStats(u.id).then(r => r.data),
    enabled: open,
    staleTime: 1000 * 60 * 5,
  })

  const library = useQuery({
    queryKey: ['admin', 'user-library', u.id],
    queryFn: () => adminApi.getUserLibrary(u.id).then(r => r.data),
    enabled: isSuperAdmin && open && showLibrary,
    staleTime: 1000 * 60 * 5,
  })

  return (
    <div className="bg-[#1a1a1a] rounded-lg ring-1 ring-white/[0.06] overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3">
        <button
          onClick={() => setOpen(o => !o)}
          className="flex-1 min-w-0 text-left flex items-center gap-2 group"
        >
          <div className="min-w-0">
            <p className="text-sm text-zinc-200 font-medium">{u.username}</p>
            {u.display_name && u.display_name !== u.username && (
              <p className="text-xs text-zinc-600">{u.display_name}</p>
            )}
          </div>
          <span className="text-zinc-600 group-hover:text-zinc-400 transition-colors ml-1 flex-shrink-0">
            {open ? <ChevronUp size={13} /> : <ChevronDown size={13} />}
          </span>
        </button>

        <div className="flex items-center gap-3 flex-shrink-0">
          {isSuperAdmin ? (
            <label className={`flex items-center gap-1.5 text-xs select-none ${u.username === 'admin' ? 'text-zinc-600 cursor-not-allowed' : 'text-zinc-500 cursor-pointer'}`}>
              <input
                type="checkbox"
                checked={u.is_admin}
                disabled={u.username === 'admin'}
                onChange={(e) => updateFlags({ id: u.id, flags: { is_admin: e.target.checked } })}
                className="rounded"
              />
              Admin
            </label>
          ) : (
            u.is_admin && <span className="text-xs px-2 py-0.5 rounded-full bg-white/[0.06] text-zinc-400">Admin</span>
          )}

          <label className={`flex items-center gap-1.5 text-xs select-none ${u.username === 'admin' ? 'text-zinc-600 cursor-not-allowed' : 'text-zinc-500 cursor-pointer'}`}>
            <input
              type="checkbox"
              checked={u.is_premium}
              disabled={u.username === 'admin'}
              onChange={(e) => updateFlags({ id: u.id, flags: { is_premium: e.target.checked } })}
              className="rounded"
            />
            Premium
          </label>

          {isSuperAdmin && (
            <button
              onClick={() => onResetPassword(u)}
              className="p-1.5 text-zinc-600 hover:text-indigo-400 transition-colors rounded"
              title="Reset password"
            >
              <KeyRound size={14} />
            </button>
          )}
          {isSuperAdmin && u.username !== 'admin' && (
            <button
              onClick={() => onDelete(u)}
              className="p-1.5 text-zinc-600 hover:text-red-400 transition-colors rounded"
              title="Delete user"
            >
              <Trash2 size={14} />
            </button>
          )}
        </div>
      </div>

      {open && (
        <div className="px-4 pb-3 border-t border-white/[0.04]">
          <p className="text-[10px] text-zinc-600 mt-2 mb-2">
            Joined {formatDate(u.created_at)}
          </p>
          {stats.isLoading && <p className="text-xs text-zinc-600">Loading…</p>}
          {stats.isError && <p className="text-xs text-red-400">Failed to load stats</p>}
          {stats.data && <UserStatsPanel stats={stats.data} />}

          {isSuperAdmin && (
            <div className="mt-3 border-t border-white/[0.04] pt-3">
              <button
                onClick={() => setShowLibrary(v => !v)}
                className="text-[10px] text-zinc-500 hover:text-zinc-300 transition-colors uppercase tracking-wider"
              >
                {showLibrary ? 'Hide library' : 'View library'}
              </button>
              {showLibrary && (
                <>
                  {library.isLoading && <p className="text-xs text-zinc-600 mt-2">Loading…</p>}
                  {library.isError && <p className="text-xs text-red-400 mt-2">Failed to load library</p>}
                  {library.data && <UserLibraryPanel items={library.data} />}
                </>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default function Admin() {
  const [tab, setTab] = useState<Tab>('invites')
  const [newCode, setNewCode] = useState(randomCode)
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null)
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null)
  const { show } = useToast()
  const { user: currentUser } = useAuth()
  const isSuperAdmin = currentUser?.username === 'admin'
  const qc = useQueryClient()

  const stats   = useQuery({ queryKey: ['admin', 'stats'],   queryFn: () => adminApi.getStats().then(r => r.data) })
  const health  = useQuery({ queryKey: ['admin', 'health'],  queryFn: () => adminApi.getHealth().then(r => r.data), staleTime: 30_000 })
  const invites = useQuery({ queryKey: ['admin', 'invites'], queryFn: () => adminApi.listInvites().then(r => r.data) })
  const users   = useQuery({ queryKey: ['admin', 'users'],   queryFn: () => adminApi.listUsers().then(r => r.data), enabled: tab === 'users' })
  const flags   = useQuery({ queryKey: ['admin', 'flags'],   queryFn: () => adminApi.getFlags().then(r => r.data), enabled: isSuperAdmin && tab === 'flags' })

  const deleteInvite = useMutation({
    mutationFn: (code: string) => adminApi.deleteInvite(code),
    onSuccess: () => {
      show('Invite code revoked', 'success')
      qc.invalidateQueries({ queryKey: ['admin', 'invites'] })
    },
    onError: () => show('Failed to revoke invite code', 'error'),
  })

  const createInvite = useMutation({
    mutationFn: () => adminApi.createInvite(newCode),
    onSuccess: () => {
      show(`Invite code "${newCode}" created`, 'success')
      setNewCode(randomCode())
      qc.invalidateQueries({ queryKey: ['admin', 'invites'] })
    },
    onError: () => show('Code already exists', 'error'),
  })

  const updateFlags = useMutation({
    mutationFn: ({ id, flags }: { id: number; flags: { is_admin?: boolean; is_premium?: boolean } }) =>
      adminApi.updateFlags(id, flags),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'users'] }),
    onError: () => show('Failed to update user', 'error'),
  })

  const resetPassword = useMutation({
    mutationFn: ({ id, password }: { id: number; password: string }) =>
      adminApi.resetPassword(id, password),
    onSuccess: () => { show('Password reset successfully', 'success'); setResetTarget(null) },
    onError: () => show('Failed to reset password', 'error'),
  })

  const setFlag = useMutation({
    mutationFn: ({ key, isPremium }: { key: string; isPremium: boolean }) =>
      adminApi.setFlag(key, isPremium),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'flags'] })
      qc.invalidateQueries({ queryKey: ['user'] })
    },
    onError: () => show('Failed to update feature flag', 'error'),
  })

  const deleteUser = useMutation({
    mutationFn: (id: number) => adminApi.deleteUser(id),
    onSuccess: () => {
      show(`User deleted`, 'success')
      setDeleteTarget(null)
      qc.invalidateQueries({ queryKey: ['admin', 'users'] })
    },
    onError: () => show('Failed to delete user', 'error'),
  })

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    show('Copied to clipboard', 'success')
  }

  const tabClass = (t: Tab) =>
    `px-3 py-1 rounded-md text-xs font-medium transition-colors ${
      tab === t ? 'bg-white/10 text-white' : 'text-zinc-500 hover:bg-white/5 hover:text-zinc-300'
    }`

  return (
    <div className="space-y-6 max-w-2xl">
      <h1 className="text-2xl font-semibold text-white tracking-tight">Admin</h1>

      {stats.isLoading && <p className="text-zinc-600 text-sm">Loading stats…</p>}
      {stats.isError && <p className="text-red-400 text-sm">Failed to load stats — you may not have admin access yet. Try signing out and back in.</p>}
      {stats.data && <OverviewCard s={stats.data} />}
      {stats.data && <InfraCard s={stats.data} />}
      <ServicesCard services={health.data} loading={health.isLoading} />

      <div className="flex gap-1.5">
        <button className={tabClass('invites')} onClick={() => setTab('invites')}>Invite Codes</button>
        <button className={tabClass('users')}   onClick={() => setTab('users')}>Users</button>
        {isSuperAdmin && (
          <button className={tabClass('flags')} onClick={() => setTab('flags')}>Feature Flags</button>
        )}
      </div>

      {tab === 'invites' && (
        <div className="space-y-4">
          <div className="bg-[#1a1a1a] rounded-lg p-4 ring-1 ring-white/[0.06] space-y-3">
            <p className="text-xs text-zinc-500 font-medium uppercase tracking-wider">New Code</p>
            <div className="flex gap-2">
              <input
                value={newCode}
                onChange={(e) => setNewCode(e.target.value.toUpperCase())}
                className="flex-1 bg-[#111] text-zinc-200 rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-white/20 text-sm font-mono"
              />
              <button
                onClick={() => setNewCode(randomCode())}
                className="p-2 text-zinc-500 hover:text-zinc-300 hover:bg-white/5 rounded-md transition-colors"
                title="Regenerate"
              >
                <RefreshCw size={15} />
              </button>
              <button
                onClick={() => createInvite.mutate()}
                disabled={createInvite.isPending || newCode.length < 8}
                className="px-4 py-2 bg-white/10 hover:bg-white/15 disabled:opacity-40 text-white text-sm rounded-md transition-colors"
              >
                Create
              </button>
            </div>
            <p className="text-xs text-zinc-600">Share the link: brsti.uk/register?invite={newCode}</p>
          </div>

          {invites.isLoading && <p className="text-zinc-600 text-sm">Loading…</p>}
          {invites.isError && <p className="text-red-400 text-sm">Failed to load invite codes.</p>}
          {invites.data && (
            <div className="space-y-1.5">
              {invites.data.length === 0 && (
                <p className="text-zinc-600 text-sm text-center py-8">No invite codes yet</p>
              )}
              {invites.data.map((inv) => (
                <div key={inv.code} className={`flex items-center gap-3 rounded-lg px-4 py-3 ring-1 ${inv.used_at ? 'bg-[#141414] ring-white/[0.04]' : 'bg-[#1a1a1a] ring-white/[0.06]'}`}>
                  <span className={`font-mono text-sm flex-1 ${inv.used_at ? 'text-zinc-600 line-through' : 'text-zinc-200'}`}>
                    {inv.code}
                  </span>
                  {inv.used_at ? (
                    <span className="text-xs text-zinc-600">Used</span>
                  ) : (
                    <>
                      <span className="text-xs text-emerald-600">Unused</span>
                      <button
                        onClick={() => copyToClipboard(`https://brsti.uk/register?invite=${inv.code}`)}
                        className="p-1 text-zinc-600 hover:text-zinc-300 transition-colors"
                        title="Copy invite link"
                      >
                        <Copy size={13} />
                      </button>
                      <button
                        onClick={() => deleteInvite.mutate(inv.code)}
                        disabled={deleteInvite.isPending}
                        className="p-1 text-zinc-600 hover:text-red-400 transition-colors disabled:opacity-30"
                        title="Revoke invite code"
                      >
                        <Trash2 size={13} />
                      </button>
                    </>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {tab === 'users' && (
        <div className="space-y-1.5">
          {users.isLoading && <p className="text-zinc-600 text-sm">Loading…</p>}
          {users.isError && <p className="text-red-400 text-sm">Failed to load users.</p>}
          {users.data?.map((u) => (
            <UserRow
              key={u.id}
              u={u}
              isSuperAdmin={isSuperAdmin}
              updateFlags={(args) => updateFlags.mutate(args)}
              onDelete={(u) => setDeleteTarget({ id: u.id, username: u.username })}
              onResetPassword={setResetTarget}
            />
          ))}
        </div>
      )}

      {tab === 'flags' && isSuperAdmin && (
        <div className="space-y-3">
          <p className="text-xs text-zinc-500">Toggle whether a feature requires a premium subscription or is available to all users.</p>
          {flags.isLoading && <p className="text-zinc-600 text-sm">Loading…</p>}
          {flags.isError && <p className="text-red-400 text-sm">Failed to load feature flags.</p>}
          {flags.data && (
            <FeatureFlagsTab
              flags={flags.data}
              onToggle={(key, isPremium) => setFlag.mutate({ key, isPremium })}
            />
          )}
        </div>
      )}

      {resetTarget && (
        <ResetPasswordModal
          username={resetTarget.username}
          isPending={resetPassword.isPending}
          onConfirm={(password) => resetPassword.mutate({ id: resetTarget.id, password })}
          onCancel={() => setResetTarget(null)}
        />
      )}

      {deleteTarget && (
        <ConfirmDeleteModal
          username={deleteTarget.username}
          onConfirm={() => deleteUser.mutate(deleteTarget.id)}
          onCancel={() => setDeleteTarget(null)}
        />
      )}
    </div>
  )
}
