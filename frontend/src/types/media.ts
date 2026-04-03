export type MediaType = 'film' | 'tv_show' | 'book' | 'anime'
export type MediaStatus = 'want_to' | 'in_progress' | 'completed' | 'dropped' | 'on_hold'
export type ChapterStatus = 'unread' | 'in_progress' | 'completed'

export interface MediaItem {
  id: number
  user_id: number
  media_type: MediaType
  external_id: string | null
  title: string
  year: number | null
  poster_url: string | null
  metadata: Record<string, unknown>
  status: MediaStatus
  rating: number | null
  review_text: string | null
  started_at: string | null
  completed_at: string | null
  current_progress: number | null
  total_progress: number | null
  created_at: string
  updated_at: string
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  per_page: number
  pages: number
}

export interface EpisodeLog {
  id: number
  media_item_id: number
  season_number: number
  episode_number: number
  watched_at: string | null
  rating: number | null
  note: string | null
}

export interface ChapterLog {
  id: number
  media_item_id: number
  chapter_number: number
  chapter_title: string | null
  start_page: number | null
  end_page: number | null
  status: ChapterStatus
  note: string | null
  started_at: string | null
  completed_at: string | null
}

export interface SearchResult {
  source: string
  media_type: string
  external_id: string
  title: string
  year: number | null
  poster_url: string | null
  description: string | null
  extra: Record<string, unknown>
}

export interface SearchResponse {
  results: SearchResult[]
  total: number
}

export interface StatsSummary {
  films: { total: number; this_month: number; avg_rating: number | null }
  tv_shows: { total: number; in_progress: number; episodes_this_month: number }
  books: { total: number; in_progress: number; chapters_this_month: number; pages_this_month: number }
  current_streak_days: number
  recently_completed: MediaItem[]
}

export interface UserList {
  id: number
  user_id: number
  name: string
  description: string | null
  is_public: boolean
  created_at: string
  updated_at: string
  items: ListItem[]
}

export interface ListItem {
  id: number
  list_id: number
  media_item_id: number
  position: number
  added_at: string
  media_item?: MediaItem
}
