import { db, type MutationQueueItem } from './db'

export async function enqueue(item: Omit<MutationQueueItem, 'id' | 'createdAt' | 'retries'>): Promise<void> {
  await db.mutationQueue.add({ ...item, createdAt: Date.now(), retries: 0 })
}

export async function dequeue(id: number): Promise<void> {
  await db.mutationQueue.delete(id)
}

export async function getPending(): Promise<MutationQueueItem[]> {
  return db.mutationQueue.orderBy('createdAt').toArray()
}

export async function incrementRetry(id: number): Promise<void> {
  const item = await db.mutationQueue.get(id)
  if (item) {
    await db.mutationQueue.update(id, { retries: item.retries + 1 })
  }
}
