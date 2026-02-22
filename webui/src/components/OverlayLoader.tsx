interface OverlayLoaderProps {
  message?: string
}

export function OverlayLoader({ message = 'Loading...' }: OverlayLoaderProps) {
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center bg-black/30 backdrop-blur-sm">
      <div className="flex items-center gap-3 rounded-xl bg-white dark:bg-slate-800 px-6 py-4 shadow-lg border border-slate-200 dark:border-slate-700">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-brand-purple border-t-transparent" />
        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{message}</span>
      </div>
    </div>
  )
}
