import { STATUS_LABELS, STATUS_COLOURS } from '@/utils/constants'

interface Props { status: string }

export default function StatusBadge({ status }: Props) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${STATUS_COLOURS[status] ?? 'bg-zinc-700 text-zinc-300'}`}>
      {STATUS_LABELS[status] ?? status}
    </span>
  )
}
