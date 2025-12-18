import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { chaptersApi } from '@/api/chapters'
import type { ChapterStatus } from '@/types/media'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

interface Props { mediaId: number }

const STATUS_OPTIONS: ChapterStatus[] = ['unread', 'in_progress', 'completed']

export default function ChapterTracker({ mediaId }: Props) {
  const qc = useQueryClient()
  const { data: chapters, isLoading } = useQuery({
    queryKey: ['chapters', mediaId],
    queryFn: () => chaptersApi.list(mediaId).then((r) => r.data),
  })

  const update = useMutation({
    mutationFn: ({ chId, status }: { chId: number; status: ChapterStatus }) =>
      chaptersApi.update(mediaId, chId, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chapters', mediaId] }),
  })

  const add = useMutation({
    mutationFn: (data: { chapter_number: number; chapter_title?: string; start_page?: number; end_page?: number }) =>
      chaptersApi.log(mediaId, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['chapters', mediaId] }); setForm({ chapter_number: '', chapter_title: '', start_page: '', end_page: '' }) },
  })

  const [form, setForm] = useState({ chapter_number: '', chapter_title: '', start_page: '', end_page: '' })

  if (isLoading) return <LoadingSpinner />

  return (
    <div className="space-y-4">
      <h3 className="font-semibold text-white">Chapter Tracker</h3>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 items-end">
        {(['chapter_number', 'chapter_title', 'start_page', 'end_page'] as const).map((field) => (
          <div key={field}>
            <label className="block text-xs text-slate-400 mb-1 capitalize">{field.replace(/_/g, ' ')}</label>
            <input value={form[field]} onChange={(e) => setForm((f) => ({ ...f, [field]: e.target.value }))}
              type={field.includes('page') || field === 'chapter_number' ? 'number' : 'text'}
              className="w-full bg-slate-700 text-white rounded px-2 py-1 text-sm" />
          </div>
        ))}
        <button onClick={() => add.mutate({
          chapter_number: +form.chapter_number,
          chapter_title: form.chapter_title || undefined,
          start_page: form.start_page ? +form.start_page : undefined,
          end_page: form.end_page ? +form.end_page : undefined,
        })} className="col-span-2 sm:col-span-1 bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-1 rounded text-sm">
          Add Chapter
        </button>
      </div>

      <div className="space-y-2">
        {(chapters ?? []).map((ch) => (
          <div key={ch.id} className="flex items-center gap-3 bg-slate-800 rounded px-3 py-2">
            <span className="text-slate-400 text-sm w-6">#{ch.chapter_number}</span>
            <span className="flex-1 text-white text-sm truncate">{ch.chapter_title ?? `Chapter ${ch.chapter_number}`}</span>
            {ch.start_page && ch.end_page && (
              <span className="text-xs text-slate-500">pp.{ch.start_page}–{ch.end_page}</span>
            )}
            <select value={ch.status} onChange={(e) => update.mutate({ chId: ch.id, status: e.target.value as ChapterStatus })}
              className="bg-slate-700 text-white text-xs rounded px-1 py-0.5">
              {STATUS_OPTIONS.map((s) => <option key={s} value={s}>{s}</option>)}
            </select>
          </div>
        ))}
      </div>
    </div>
  )
}
