interface StringListInputProps {
  id: string
  values: string[]
  placeholder?: string
  disabled?: boolean
  addLabel?: string
  emptyLabel?: string
  onChange: (values: string[]) => void
}

/**
 * Edits a list of short strings as one input per entry.
 *
 * Empty entries are kept while editing - clearing a field must not make its row
 * vanish under the cursor - so the caller drops them when it submits.
 */
export function StringListInput({
  id,
  values,
  placeholder,
  disabled = false,
  addLabel = 'Add',
  emptyLabel = 'No entry yet',
  onChange,
}: StringListInputProps) {
  const handleChange = (index: number, value: string) => {
    const next = [...values]
    next[index] = value
    onChange(next)
  }

  const handleRemove = (index: number) => {
    onChange(values.filter((_, i) => i !== index))
  }

  return (
    <div>
      {values.length > 0 ? (
        <div className="space-y-2">
          {values.map((value, index) => (
            <div key={index} className="flex items-center gap-2">
              <input
                id={index === 0 ? id : undefined}
                type="text"
                value={value}
                placeholder={placeholder}
                disabled={disabled}
                onChange={(e) => handleChange(index, e.target.value)}
                className="flex-1 min-w-0 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2 px-3 text-sm font-mono text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-2 focus:ring-brand-purple/20 disabled:opacity-60"
              />
              <button
                type="button"
                onClick={() => handleRemove(index)}
                disabled={disabled}
                title="Remove"
                className="flex-shrink-0 p-2 rounded-lg text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors disabled:opacity-60"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          ))}
        </div>
      ) : (
        <p className="py-2 text-sm italic text-slate-400 dark:text-slate-500">{emptyLabel}</p>
      )}

      <button
        type="button"
        onClick={() => onChange([...values, ''])}
        disabled={disabled}
        className="mt-2 flex items-center gap-1 px-3 py-1.5 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-60"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
        {addLabel}
      </button>
    </div>
  )
}
