export const MEDIA_TYPE_LABELS: Record<string, string> = {
  film: 'Movies',
  tv_show: 'TV Shows',
  book: 'Books',
  anime: 'Anime',
}

export const STATUS_LABELS: Record<string, string> = {
  want_to: 'Want to',
  in_progress: 'In Progress',
  completed: 'Completed',
  dropped: 'Dropped',
  on_hold: 'On Hold',
}

export const STATUS_COLOURS: Record<string, string> = {
  want_to: 'bg-zinc-700 text-zinc-300',
  in_progress: 'bg-blue-500/20 text-blue-400',
  completed: 'bg-emerald-500/20 text-emerald-400',
  dropped: 'bg-red-500/20 text-red-400',
  on_hold: 'bg-amber-500/20 text-amber-400',
}
