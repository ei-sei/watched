import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { listsApi } from '@/api/lists'
import type { UserList } from '@/types/media'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import { Copy, Check } from 'lucide-react'


export default function SharedList() {
  const { id } = useParams<{ id: string }>()
  const [list, setList] = useState<UserList | null>(null)
  const [loading, setLoading] = useState(true)
  const [notFound, setNotFound] = useState(false)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    if (!id) return
    listsApi.getPublic(Number(id))
      .then((r) => setList(r.data))
      .catch(() => setNotFound(true))
      .finally(() => setLoading(false))
  }, [id])

  const copyLink = () => {
    navigator.clipboard.writeText(window.location.href)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  if (loading) return (
    <div className="min-h-screen bg-[#0d0d0d] flex items-center justify-center">
      <LoadingSpinner />
    </div>
  )

  if (notFound || !list) return (
    <div className="min-h-screen bg-[#0d0d0d] flex items-center justify-center">
      <div className="text-center space-y-3">
        <p className="text-zinc-400 text-sm">This list doesn't exist or isn't public.</p>
        <Link to="/" className="text-indigo-400 text-sm hover:underline">Go to Watched</Link>
      </div>
    </div>
  )

  return (
    <div className="min-h-screen bg-[#0d0d0d] text-zinc-200">
      <div className="max-w-2xl mx-auto px-4 py-12 space-y-8">
        {/* Header */}
        <div className="space-y-2">
          <div className="flex items-start justify-between gap-4">
            <h1 className="text-2xl font-semibold text-white tracking-tight">{list.name}</h1>
            <button
              onClick={copyLink}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-white/5 hover:bg-white/10 text-zinc-400 hover:text-zinc-200 text-xs transition-colors flex-shrink-0"
            >
              {copied ? <Check size={13} /> : <Copy size={13} />}
              {copied ? 'Copied!' : 'Copy link'}
            </button>
          </div>
          {list.description && (
            <p className="text-zinc-500 text-sm">{list.description}</p>
          )}
          <p className="text-zinc-600 text-xs">{list.items.length} items</p>
        </div>

        {/* Items */}
        {list.items.length === 0 && (
          <p className="text-zinc-600 text-sm">This list is empty.</p>
        )}
        <div className="space-y-2">
          {list.items.map((li, idx) => {
            const item = li.media_item
            if (!item) return null
            return (
              <div key={li.id} className="flex items-center gap-4 bg-[#1a1a1a] rounded-lg p-3 ring-1 ring-white/[0.06]">
                <span className="text-zinc-600 text-sm w-5 text-right flex-shrink-0">{idx + 1}</span>
                {item.poster_url ? (
                  <img src={item.poster_url} alt={item.title} className="w-9 h-12 object-cover rounded flex-shrink-0" />
                ) : (
                  <div className="w-9 h-12 bg-[#222] rounded flex-shrink-0" />
                )}
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium text-zinc-100 truncate">{item.title}</p>
                  <p className="text-xs text-zinc-600 mt-0.5">
                    {item.year && `${item.year} · `}{item.media_type.replace('_', ' ')}
                  </p>
                </div>
                {item.rating && (
                  <span className="text-xs text-zinc-500 flex-shrink-0">{item.rating}/10</span>
                )}
              </div>
            )
          })}
        </div>

        {/* Footer */}
        <div className="pt-4 border-t border-white/[0.06]">
          <Link to="/" className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors">
            Made with Watched
          </Link>
        </div>
      </div>
    </div>
  )
}
