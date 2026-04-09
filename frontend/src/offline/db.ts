import Dexie, { type Table } from 'dexie'

export interface MutationQueueItem {
  id?: number
  method: 'POST' | 'PATCH' | 'DELETE'
  url: string
  data?: unknown
  createdAt: number
  retries: number
}

class BrtstiDB extends Dexie {
  mutationQueue!: Table<MutationQueueItem>

  constructor() {
    super('watched')
    this.version(1).stores({
      mutationQueue: '++id, createdAt, retries',
    })
  }
}

export const db = new BrtstiDB()
