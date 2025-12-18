import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { episodesApi } from '@/api/episodes'
import type { EpisodeLog } from '@/types/media'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

interface Props { mediaId: number }

export default function EpisodeTracker({ mediaId }: Props) {
  const qc = useQueryClient()
  const { data: episodes, isLoading } = useQuery({
    queryKey: ['episodes', mediaId],
    queryFn: () => episodesApi.list(mediaId).then((r) => r.data),
  })

  const log = useMutation({
    mutationFn: ({ season, episode }: { season: number; episode: number }) =>
      episodesApi.log(mediaId, { season_number: season, episode_number: episode }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['episodes', mediaId] }),
  })

  const remove = useMutation({
    mutationFn: (epId: number) => episodesApi.delete(mediaId, epId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['episodes', mediaId] }),
  })

  const [newSeason, setNewSeason] = useState(1)
  const [newEpisode, setNewEpisode] = useState(1)

  if (isLoading) return <LoadingSpinner />

  const byseason = (episodes ?? []).reduce<Record<number, EpisodeLog[]>>((acc, ep) => {
    if (!acc[ep.season_number]) acc[ep.season_number] = []
    acc[ep.season_number].push(ep)
    return acc
  }, {})

  return (
    <div className="space-y-4">
      <h3 className="font-semibold text-white">Episode Tracker</h3>

      <div className="flex gap-2 items-end">
        <div>
          <label className="block text-xs text-slate-400 mb-1">Season</label>
          <input type="number" min={1} value={newSeason} onChange={(e) => setNewSeason(+e.target.value)}
            className="w-20 bg-slate-700 text-white rounded px-2 py-1 text-sm" />
        </div>
        <div>
          <label className="block text-xs text-slate-400 mb-1">Episode</label>
          <input type="number" min={1} value={newEpisode} onChange={(e) => setNewEpisode(+e.target.value)}
            className="w-20 bg-slate-700 text-white rounded px-2 py-1 text-sm" />
        </div>
        <button onClick={() => log.mutate({ season: newSeason, episode: newEpisode })}
          className="bg-indigo-600 hover:bg-indigo-700 text-white px-3 py-1 rounded text-sm">
          Log
        </button>
      </div>

      {Object.entries(byseason).sort(([a], [b]) => +a - +b).map(([season, eps]) => (
        <div key={season}>
          <p className="text-sm font-medium text-slate-300 mb-2">Season {season}</p>
          <div className="flex flex-wrap gap-2">
            {eps.sort((a, b) => a.episode_number - b.episode_number).map((ep) => (
              <button key={ep.id} onClick={() => remove.mutate(ep.id)}
                title="Click to remove"
                className="w-9 h-9 rounded bg-indigo-600 text-white text-xs font-bold hover:bg-red-600 transition-colors">
                {ep.episode_number}
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
