interface Props { rating: number | null; size?: 'sm' | 'md' }

export default function RatingDisplay({ rating, size = 'md' }: Props) {
  if (rating === null) return null
  const cls = size === 'sm' ? 'text-sm' : 'text-base'
  return (
    <span className={`font-semibold text-yellow-400 ${cls}`}>
      ★ {rating.toFixed(1)}
    </span>
  )
}
