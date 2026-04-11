import { useEffect } from 'react'
import { Bookmark, Play, CheckCircle, PauseCircle, XCircle } from 'lucide-react'
import { STATUS_LABELS, STATUS_LABELS_BOOK, STATUS_COLOURS } from '@/utils/constants'
import type { MediaStatus } from '@/types/media'

const STATUSES: MediaStatus[] = ['in_progress', 'want_to', 'completed', 'on_hold', 'dropped']

const ICONS = {
  want_to:     Bookmark,
  in_progress: Play,
  completed:   CheckCircle,
  on_hold:     PauseCircle,
  dropped:     XCircle,
} as const

interface Props {
  open: boolean
  current: MediaStatus
  mediaType: string
  onSelect: (status: MediaStatus) => void
  onClose: () => void
}

export default function StatusSheet({ open, current, mediaType, onSelect, onClose }: Props) {
  // Lock body scroll while sheet is open
  useEffect(() => {
    if (open) document.body.style.overflow = 'hidden'
    else document.body.style.overflow = ''
    return () => { document.body.style.overflow = '' }
  }, [open])

  if (!open) return null

  const labels = mediaType === 'book' ? STATUS_LABELS_BOOK : STATUS_LABELS

  return (
    <div className="fixed inset-0 z-50 md:hidden">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Sheet */}
      <div className="absolute bottom-0 left-0 right-0 bg-[#1a1a1a] rounded-t-2xl shadow-2xl">
        {/* Handle */}
        <div className="flex justify-center pt-3 pb-1">
          <div className="w-9 h-1 rounded-full bg-white/20" />
        </div>

        <p className="text-xs text-zinc-500 uppercase tracking-wider px-5 py-3">Set status</p>

        <div className="pb-safe">
          {STATUSES.map((s) => {
            const Icon = ICONS[s]
            const active = s === current
            const colour = STATUS_COLOURS[s]
            // Extract the text colour class for the icon/label when active
            const textColour = colour.split(' ').find((c) => c.startsWith('text-')) ?? 'text-zinc-300'

            return (
              <button
                key={s}
                onClick={() => { onSelect(s); onClose() }}
                className={`w-full flex items-center gap-4 px-5 py-4 transition-colors ${
                  active ? 'bg-white/[0.06]' : 'hover:bg-white/[0.03]'
                }`}
              >
                <span className={`flex-shrink-0 ${active ? textColour : 'text-zinc-600'}`}>
                  <Icon size={20} strokeWidth={2} />
                </span>
                <span className={`text-base font-medium ${active ? 'text-white' : 'text-zinc-400'}`}>
                  {labels[s]}
                </span>
                {active && (
                  <span className="ml-auto w-2 h-2 rounded-full bg-current" style={{ color: 'inherit' }}>
                    <span className={`block w-2 h-2 rounded-full ${textColour.replace('text-', 'bg-')}`} />
                  </span>
                )}
              </button>
            )
          })}
        </div>

        {/* Extra bottom padding for home indicator */}
        <div className="h-6" />
      </div>
    </div>
  )
}
