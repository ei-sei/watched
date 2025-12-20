import { useState } from 'react'
import { useUpdateMedia } from '@/hooks/useMedia'
import type { MediaItem, MediaType } from '@/types/media'

interface Props {
  item: MediaItem
}

const UNIT: Record<MediaType, string> = {
  book:    'page',
  tv_show: 'episode',
  anime:   'episode',
  film:    'minute',
}

function getDefaultTotal(item: MediaItem): number | '' {
  if (item.total_progress) return item.total_progress
  // Pre-fill total episodes for anime from MAL metadata
  if (item.media_type === 'anime' && item.metadata?.episodes) {
    return item.metadata.episodes as number
  }
  return ''
}

export default function ProgressTracker({ item }: Props) {
  const update = useUpdateMedia()
  const unit = UNIT[item.media_type] ?? 'item'

  const [current, setCurrent] = useState<number | ''>(item.current_progress ?? '')
  const [total, setTotal] = useState<number | ''>(getDefaultTotal(item))

  const save = (newCurrent: number | '', newTotal: number | '') => {
    update.mutate({
      id: item.id,
      data: {
        current_progress: newCurrent !== '' ? newCurrent : null,
        total_progress: newTotal !== '' ? newTotal : null,
      },
    })
  }

  const pct = current !== '' && total !== '' && total > 0
    ? Math.min(100, Math.round((current / total) * 100))
    : null

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">Progress</h3>

      <div className="flex items-center gap-2 flex-wrap">
        <div className="flex items-center gap-1 bg-[#1a1a1a] rounded-md border border-white/[0.08] px-1">
          <button
            onClick={() => { const v = Math.max(0, (current === '' ? 0 : current) - 1); setCurrent(v); save(v, total) }}
            className="w-7 h-7 flex items-center justify-center text-zinc-500 hover:text-zinc-200 transition-colors text-base"
          >−</button>
          <input
            type="number"
            min={0}
            max={total !== '' ? total : undefined}
            value={current}
            onChange={(e) => setCurrent(e.target.value === '' ? '' : +e.target.value)}
            onBlur={() => save(current, total)}
            placeholder="0"
            className="w-12 bg-transparent text-zinc-200 text-sm py-1.5 focus:outline-none text-center [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
          />
          <button
            onClick={() => { const v = (current === '' ? 0 : current) + 1; setCurrent(v); save(v, total) }}
            className="w-7 h-7 flex items-center justify-center text-zinc-500 hover:text-zinc-200 transition-colors text-base"
          >+</button>
        </div>
        <span className="text-zinc-600 text-sm">/</span>
        <input
          type="number"
          min={1}
          value={total}
          onChange={(e) => setTotal(e.target.value === '' ? '' : +e.target.value)}
          onBlur={() => save(current, total)}
          placeholder={`total ${unit}s`}
          className="w-28 bg-[#1a1a1a] text-zinc-200 text-sm rounded-md px-3 py-1.5 border border-white/[0.08] focus:outline-none focus:border-white/20 [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
        />
        <span className="text-zinc-600 text-xs">{unit}s</span>
      </div>

      {pct !== null && (
        <div className="space-y-1">
          <div className="h-1 bg-white/[0.06] rounded-full overflow-hidden">
            <div
              className="h-full bg-indigo-500 rounded-full transition-all"
              style={{ width: `${pct}%` }}
            />
          </div>
          <p className="text-xs text-zinc-600">{pct}% — {current} of {total} {unit}s</p>
        </div>
      )}
    </div>
  )
}
