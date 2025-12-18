interface Props { message: string; action?: React.ReactNode }

export default function EmptyState({ message, action }: Props) {
  return (
    <div className="flex flex-col items-center justify-center py-16 gap-4 text-slate-400">
      <p className="text-lg">{message}</p>
      {action}
    </div>
  )
}
