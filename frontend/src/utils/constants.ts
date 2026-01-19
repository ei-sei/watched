export const MEDIA_TYPE_LABELS: Record<string, string> = {
  film: 'Movies',
  tv_show: 'TV Shows',
  book: 'Books',
  anime: 'Anime',
}

// Status labels — media-type-aware variants for Watch vs Read contexts.
export const STATUS_LABELS: Record<string, string> = {
  want_to:     'Plan to Watch',
  in_progress: 'Watching',
  completed:   'Completed',
  on_hold:     'On Hold',
  dropped:     'Dropped',
}

export const STATUS_LABELS_BOOK: Record<string, string> = {
  want_to:     'Plan to Read',
  in_progress: 'Reading',
  completed:   'Completed',
  on_hold:     'On Hold',
  dropped:     'Dropped',
}

export const STATUS_COLOURS: Record<string, string> = {
  want_to:     'bg-zinc-700/60 text-zinc-400',
  in_progress: 'bg-blue-500/20 text-blue-400',
  completed:   'bg-emerald-500/20 text-emerald-400',
  on_hold:     'bg-amber-500/20 text-amber-400',
  dropped:     'bg-red-500/20 text-red-400',
}

// Lucide icon names per status — consumed by StatusBadge.
export const STATUS_ICONS: Record<string, string> = {
  want_to:     'Bookmark',
  in_progress: 'Play',
  completed:   'CheckCircle',
  on_hold:     'PauseCircle',
  dropped:     'XCircle',
}
