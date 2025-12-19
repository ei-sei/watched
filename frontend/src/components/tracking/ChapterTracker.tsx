import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { chaptersApi } from '@/api/chapters'
import type { ChapterLog } from '@/types/media'
import LoadingSpinner from '@/components/ui/LoadingSpinner'
import type { ImportResult } from '@/api/chapters'

interface Props { mediaId: number }

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      className={`w-3.5 h-3.5 text-zinc-500 transition-transform ${open ? 'rotate-180' : ''}`}
      fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}
    >
      <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
    </svg>
  )
}

function DotIndicator({ hasNote }: { hasNote: boolean }) {
  return hasNote ? (
    <span className="w-2 h-2 rounded-full bg-indigo-400 flex-shrink-0" />
  ) : (
    <span className="w-2 h-2 rounded-full border border-zinc-600 flex-shrink-0" />
  )
}

interface ChapterRowProps {
  ch: ChapterLog
  mediaId: number
}

function ChapterRow({ ch, mediaId }: ChapterRowProps) {
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(ch.note ?? '')
  const [titleDraft, setTitleDraft] = useState(ch.chapter_title ?? '')
  const [editingTitle, setEditingTitle] = useState(!ch.chapter_title)

  const upsert = (fields: Partial<{ note: string; chapter_title: string }>) =>
    chaptersApi.upsert(mediaId, {
      chapter_number: ch.chapter_number,
      chapter_title: fields.chapter_title ?? ch.chapter_title ?? undefined,
      start_page: ch.start_page ?? undefined,
      end_page: ch.end_page ?? undefined,
      status: ch.status,
      note: fields.note ?? ch.note ?? undefined,
    })

  const saveNote = useMutation({
    mutationFn: (note: string) => upsert({ note }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chapters', mediaId] })
      setEditing(false)
    },
  })

  const saveTitle = useMutation({
    mutationFn: (title: string) => upsert({ chapter_title: title }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['chapters', mediaId] }),
  })

  const pageRange = ch.start_page && ch.end_page
    ? `pp. ${ch.start_page}–${ch.end_page}`
    : ch.start_page
    ? `p. ${ch.start_page}+`
    : null

  return (
    <div className="bg-[#1a1a1a] rounded-lg ring-1 ring-white/[0.06] overflow-hidden">
      {/* Collapsed row */}
      <button
        onClick={() => setOpen((o) => !o)}
        className="w-full flex items-center gap-3 px-3 py-2.5 text-left hover:bg-white/[0.03] transition-colors"
      >
        <DotIndicator hasNote={!!ch.note} />
        <span className="text-xs text-zinc-500 flex-shrink-0">Ch. {ch.chapter_number}</span>
        <span className="flex-1 text-sm text-zinc-200 truncate">
          {ch.chapter_title ?? ''}
        </span>
        {pageRange && (
          <span className="text-xs text-zinc-600 flex-shrink-0">{pageRange}</span>
        )}
        <ChevronIcon open={open} />
      </button>

      {/* Expanded state */}
      {open && (
        <div className="px-3 pb-3 border-t border-white/[0.04]">
          <div className="pt-3 space-y-3">
            {editingTitle ? (
              <input
                type="text"
                value={titleDraft}
                autoFocus
                onChange={(e) => setTitleDraft(e.target.value)}
                onBlur={() => {
                  if (titleDraft !== (ch.chapter_title ?? '')) saveTitle.mutate(titleDraft)
                  if (titleDraft) setEditingTitle(false)
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') (e.target as HTMLInputElement).blur()
                }}
                placeholder="Chapter title"
                className="w-full bg-[#111] text-zinc-200 text-sm rounded-md px-3 py-1.5 border border-white/[0.08] focus:outline-none focus:border-white/20 placeholder:text-zinc-600"
              />
            ) : (
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm text-zinc-500 truncate">{ch.chapter_title}</span>
                <button
                  onClick={() => setEditingTitle(true)}
                  className="text-xs text-zinc-600 hover:text-zinc-300 transition-colors flex-shrink-0"
                >
                  Edit
                </button>
              </div>
            )}
          </div>
          <div className="pt-3">
            {!editing && ch.note ? (
              <div className="space-y-2">
                <p className="text-sm text-zinc-300 leading-relaxed whitespace-pre-wrap">{ch.note}</p>
                <button
                  onClick={() => { setDraft(ch.note ?? ''); setEditing(true) }}
                  className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
                >
                  Edit
                </button>
              </div>
            ) : editing || !ch.note ? (
              <div className="space-y-2">
                <textarea
                  value={draft}
                  onChange={(e) => setDraft(e.target.value)}
                  placeholder="What are your thoughts?"
                  rows={3}
                  className="w-full bg-[#111] text-zinc-200 text-sm rounded-md px-3 py-2 border border-white/[0.08] focus:outline-none focus:border-white/20 placeholder:text-zinc-600 resize-none"
                />
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => saveNote.mutate(draft)}
                    disabled={saveNote.isPending}
                    className="text-xs bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white px-3 py-1 rounded-md transition-colors"
                  >
                    {saveNote.isPending ? 'Saving…' : 'Save'}
                  </button>
                  {editing && (
                    <button
                      onClick={() => setEditing(false)}
                      className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
                    >
                      Cancel
                    </button>
                  )}
                </div>
              </div>
            ) : null}
          </div>
        </div>
      )}
    </div>
  )
}

export default function ChapterTracker({ mediaId }: Props) {
  const qc = useQueryClient()
  const { data: chapters, isLoading } = useQuery({
    queryKey: ['chapters', mediaId],
    queryFn: () => chaptersApi.list(mediaId).then((r) => r.data),
  })

  const [showImport, setShowImport] = useState(false)
  const [importCount, setImportCount] = useState('')
  const [importResult, setImportResult] = useState<ImportResult | null>(null)

  const importChapters = useMutation({
    mutationFn: () => chaptersApi.import(mediaId, importCount ? +importCount : undefined).then((r) => r.data),
    onSuccess: (data) => {
      qc.invalidateQueries({ queryKey: ['chapters', mediaId] })
      setImportResult(data)
      setShowImport(false)
      setImportCount('')
    },
  })

  if (isLoading) return <LoadingSpinner />

  const list = chapters ?? []
  const reviewed = list.filter((c) => c.note).length
  const total = list.length
  const pct = total > 0 ? Math.round((reviewed / total) * 100) : 0

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-zinc-300 uppercase tracking-wider">Chapter tracker</h3>
        <button
          onClick={() => { setShowImport((v) => !v); setImportResult(null) }}
          className="text-xs text-zinc-500 hover:text-zinc-300 transition-colors"
        >
          {showImport ? 'Cancel' : 'Import chapters'}
        </button>
      </div>

      {importResult && (
        <p className="text-xs text-zinc-400">
          {importResult.source === 'openlibrary'
            ? `Imported ${importResult.imported} chapters from Open Library.`
            : `Created ${importResult.imported} numbered chapters.`}
        </p>
      )}

      {showImport && (
        <div className="bg-[#1a1a1a] rounded-lg ring-1 ring-white/[0.06] p-3 space-y-3">
          <p className="text-xs text-zinc-400 leading-relaxed">
            If this book is from Open Library and has chapter data, it will be imported automatically.
            Otherwise, enter the number of chapters to create numbered placeholders.
          </p>
          <div>
            <label className="block text-xs text-zinc-600 mb-1">Number of chapters (fallback)</label>
            <input
              type="number"
              min={1}
              max={5000}
              value={importCount}
              onChange={(e) => setImportCount(e.target.value)}
              placeholder="e.g. 24"
              className="w-full bg-[#111] text-zinc-200 text-sm rounded-md px-3 py-1.5 border border-white/[0.08] focus:outline-none focus:border-white/20 placeholder:text-zinc-600"
            />
          </div>
          {importChapters.isError && (
            <p className="text-xs text-red-400">Import failed. Try again.</p>
          )}
          <button
            onClick={() => importChapters.mutate()}
            disabled={importChapters.isPending}
            className="w-full text-sm bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white px-3 py-1.5 rounded-md transition-colors"
          >
            {importChapters.isPending ? 'Importing…' : 'Import'}
          </button>
        </div>
      )}

      {total > 0 && (
        <div className="space-y-1.5">
          <div className="h-1 bg-white/[0.06] rounded-full overflow-hidden">
            <div
              className="h-full bg-indigo-500 rounded-full transition-all"
              style={{ width: `${pct}%` }}
            />
          </div>
          <p className="text-xs text-zinc-600">
            {reviewed} of {total} chapter{total !== 1 ? 's' : ''} reviewed
          </p>
        </div>
      )}

      {list.length > 0 && (
        <div className="space-y-1.5">
          {list
            .slice()
            .sort((a, b) => a.chapter_number - b.chapter_number)
            .map((ch) => (
              <ChapterRow key={ch.id} ch={ch} mediaId={mediaId} />
            ))}
        </div>
      )}

    </div>
  )
}
