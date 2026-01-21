import { Bookmark, Play, CheckCircle, PauseCircle, XCircle } from 'lucide-react'
import { STATUS_LABELS, STATUS_LABELS_BOOK, STATUS_COLOURS } from '@/utils/constants'

const ICONS = {
  want_to:     Bookmark,
  in_progress: Play,
  completed:   CheckCircle,
  on_hold:     PauseCircle,
  dropped:     XCircle,
} as const

interface Props {
  status: string
  mediaType?: string
}

export default function StatusBadge({ status, mediaType }: Props) {
  const labels = mediaType === 'book' ? STATUS_LABELS_BOOK : STATUS_LABELS
  const label = labels[status] ?? status
  const colour = STATUS_COLOURS[status] ?? 'bg-zinc-700/60 text-zinc-400'
  const Icon = ICONS[status as keyof typeof ICONS]

  return (
    <span className={`inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium leading-none ${colour}`}>
      {Icon && <Icon size={9} strokeWidth={2.5} />}
      {label}
    </span>
  )
}
