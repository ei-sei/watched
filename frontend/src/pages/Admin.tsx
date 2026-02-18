import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { adminApi, type AdminStats, type ServiceStatus } from '@/api/admin'
import { useToast } from '@/components/ui/Toast'
import { Copy, RefreshCw, Trash2 } from 'lucide-react'

type Tab = 'invites' | 'users'

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

export default function Admin() {
  const [tab, setTab] = useState<Tab>('invites')
  const [newCode, setNewCode] = useState(randomCode)
  const [deleteTarget, setDeleteTarget] = useState<{ id: number; username: string } | null>(null)
  const { show } = useToast()
  const qc = useQueryClient()

  const stats   = useQuery({ queryKey: ['admin', 'stats'],   queryFn: () => adminApi.getStats().then(r => r.data) })
  const health  = useQuery({ queryKey: ['admin', 'health'],  queryFn: () => adminApi.getHealth().then(r => r.data), staleTime: 30_000 })
  const invites = useQuery({ queryKey: ['admin', 'invites'], queryFn: () => adminApi.listInvites().then(r => r.data) })
  const users   = useQuery({ queryKey: ['admin', 'users'],   queryFn: () => adminApi.listUsers().then(r => r.data), enabled: tab === 'users' })

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
            <div key={u.id} className="flex items-center gap-3 bg-[#1a1a1a] rounded-lg px-4 py-3 ring-1 ring-white/[0.06]">
              <div className="flex-1 min-w-0">
                <p className="text-sm text-zinc-200 font-medium">{u.username}</p>
                {u.display_name && u.display_name !== u.username && (
                  <p className="text-xs text-zinc-600">{u.display_name}</p>
                )}
              </div>
              <div className="flex items-center gap-3">
                <label className={`flex items-center gap-1.5 text-xs select-none ${u.username === 'admin' ? 'text-zinc-600 cursor-not-allowed' : 'text-zinc-500 cursor-pointer'}`}>
                  <input
                    type="checkbox"
                    checked={u.is_admin}
                    disabled={u.username === 'admin'}
                    onChange={(e) => updateFlags.mutate({ id: u.id, flags: { is_admin: e.target.checked } })}
                    className="rounded"
                  />
                  Admin
                </label>
                <label className={`flex items-center gap-1.5 text-xs select-none ${u.username === 'admin' ? 'text-zinc-600 cursor-not-allowed' : 'text-zinc-500 cursor-pointer'}`}>
                  <input
                    type="checkbox"
                    checked={u.is_premium}
                    disabled={u.username === 'admin'}
                    onChange={(e) => updateFlags.mutate({ id: u.id, flags: { is_premium: e.target.checked } })}
                    className="rounded"
                  />
                  Premium
                </label>
                {u.username !== 'admin' && (
                  <button
                    onClick={() => setDeleteTarget({ id: u.id, username: u.username })}
                    className="p-1.5 text-zinc-600 hover:text-red-400 transition-colors rounded"
                    title="Delete user"
                  >
                    <Trash2 size={14} />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
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
