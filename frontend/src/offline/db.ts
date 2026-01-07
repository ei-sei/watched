import Dexie, { type Table } from 'dexie'
import type { MediaItem, EpisodeLog, ChapterLog } from '@/types/media'

export interface MutationQueueItem {
  id?: number
  method: 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  url: string
  data?: unknown
  createdAt: number
  retries: number
}

class WatchedDB extends Dexie {
  mediaItems!: Table<MediaItem>
  episodeLogs!: Table<EpisodeLog>
  chapterLogs!: Table<ChapterLog>
  mutationQueue!: Table<MutationQueueItem>

  constructor() {
    super('watched')
    this.version(1).stores({
      mutationQueue: '++id, createdAt, retries',
    })
    this.version(2).stores({
      mediaItems: 'id, media_type, status, title, updated_at',
      episodeLogs: 'id, media_item_id, [media_item_id+season_number+episode_number]',
      chapterLogs: 'id, media_item_id, [media_item_id+chapter_number]',
    })
  }
}

export const db = new WatchedDB()
