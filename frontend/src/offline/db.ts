import Dexie, { type Table } from 'dexie'
import type { MediaItem, EpisodeLog, ChapterLog } from '@/types/media'

export interface MutationQueueItem {
  id?: number
  method: 'POST' | 'PATCH' | 'DELETE'
  url: string
  data?: unknown
  createdAt: number
  retries: number
}

class BrtstiDB extends Dexie {
  mediaItems!: Table<MediaItem>
  episodeLogs!: Table<EpisodeLog>
  chapterLogs!: Table<ChapterLog>
  mutationQueue!: Table<MutationQueueItem>

  constructor() {
    super('watched')
    this.version(1).stores({
      mediaItems: 'id, user_id, media_type, status, updated_at',
      episodeLogs: 'id, media_item_id, season_number, episode_number',
      chapterLogs: 'id, media_item_id, chapter_number, status',
      mutationQueue: '++id, createdAt, retries',
    })
  }
}

export const db = new BrtstiDB()
