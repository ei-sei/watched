import { useAuth } from '@/hooks/useAuth'
import { useMediaList } from '@/hooks/useMedia'
import MediaCard from '@/components/media/MediaCard'
import LoadingSpinner from '@/components/ui/LoadingSpinner'

export default function Dashboard() {
  const { user } = useAuth()
  const { data: inProgress, isLoading } = useMediaList({ status: 'in_progress', per_page: 8 })
  const { data: recentlyAdded } = useMediaList({ per_page: 8, sort: 'created_at', order: 'desc' })

  return (
    <div className="space-y-10">
      {/* Greeting */}
      <div>
        <h1 className="text-3xl font-semibold text-white tracking-tight">
          Good {getGreeting()}, {user?.display_name ?? user?.username}
        </h1>
        <p className="text-zinc-500 mt-1 text-sm">What are you watching, reading, or tracking today?</p>
      </div>

      {/* In progress */}
      <section>
        <h2 className="text-xs font-semibold text-zinc-500 uppercase tracking-widest mb-4">In Progress</h2>
        {isLoading && <LoadingSpinner />}
        {!isLoading && (!inProgress || inProgress.items.length === 0) && (
          <p className="text-zinc-600 text-sm">Nothing in progress yet.</p>
        )}
        {inProgress && inProgress.items.length > 0 && (
          <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8 gap-3">
            {inProgress.items.map((item) => (
              <MediaCard key={item.id} item={item} />
            ))}
          </div>
        )}
      </section>

      {/* Recently added */}
      {recentlyAdded && recentlyAdded.items.length > 0 && (
        <section>
          <h2 className="text-xs font-semibold text-zinc-500 uppercase tracking-widest mb-4">Recently Added</h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8 gap-3">
            {recentlyAdded.items.map((item) => (
              <MediaCard key={item.id} item={item} />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

function getGreeting() {
  const h = new Date().getHours()
  if (h < 12) return 'morning'
  if (h < 18) return 'afternoon'
  return 'evening'
}
