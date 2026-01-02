import { useState } from 'react'
import type { RedirectType, RedirectStatus, RedirectBase } from '../../generated/graphql'

export interface RedirectFormData {
  type: RedirectType
  source: string
  target: string
  status: RedirectStatus
}

interface RedirectFormModalProps {
  title: string
  initialData?: RedirectBase | null
  onSubmit: (data: RedirectFormData) => void
  onCancel: () => void
  isLoading?: boolean
}

const redirectTypes: { value: RedirectType; label: string; description: string }[] = [
  { value: 'BASIC', label: 'Basic', description: 'Simple path matching' },
  { value: 'BASIC_HOST', label: 'Host', description: 'Match with host header' },
  { value: 'REGEX', label: 'Regex', description: 'Regular expression matching' },
  { value: 'REGEX_HOST', label: 'Regex Host', description: 'Regex matching with host header' },
]

const redirectStatuses: { value: RedirectStatus; label: string; code: number }[] = [
  { value: 'MOVED_PERMANENT', label: 'Moved Permanently', code: 301 },
  { value: 'FOUND', label: 'Found', code: 302 },
  { value: 'TEMPORARY_REDIRECT', label: 'Temporary Redirect', code: 307 },
  { value: 'PERMANENT_REDIRECT', label: 'Permanent Redirect', code: 308 },
]

export function RedirectFormModal({
  title,
  initialData,
  onSubmit,
  onCancel,
  isLoading = false,
}: RedirectFormModalProps) {
  const [formData, setFormData] = useState<RedirectFormData>({
    type: initialData?.type ?? 'BASIC',
    source: initialData?.source ?? '',
    target: initialData?.target ?? '',
    status: initialData?.status ?? 'MOVED_PERMANENT',
  })

  const [errors, setErrors] = useState<Partial<Record<keyof RedirectFormData, string>>>({})

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof RedirectFormData, string>> = {}

    if (!formData.source.trim()) {
      newErrors.source = 'Source is required'
    }

    if (!formData.target.trim()) {
      newErrors.target = 'Target is required'
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (validate()) {
      onSubmit(formData)
    }
  }

  const handleChange = (field: keyof RedirectFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }))
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }))
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative w-full max-w-lg mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 p-6">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-6">{title}</h3>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Type */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Type
            </label>
            <div className="flex gap-2">
              {redirectTypes.map((type) => (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => handleChange('type', type.value)}
                  className={`flex-1 px-3 py-2 text-sm font-medium rounded-lg border transition-colors ${
                    formData.type === type.value
                      ? 'border-brand-purple bg-brand-purple/10 text-brand-purple'
                      : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700'
                  }`}
                  title={type.description}
                >
                  {type.label}
                </button>
              ))}
            </div>
          </div>

          {/* Source */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Source
            </label>
            <input
              type="text"
              value={formData.source}
              onChange={(e) => handleChange('source', e.target.value)}
              placeholder={formData.type === 'REGEX' ? '^/old-path/(.*)$' : '/old-path'}
              className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple/20 ${
                errors.source
                  ? 'border-red-500 focus:border-red-500'
                  : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple'
              }`}
            />
            {errors.source && <p className="mt-1 text-sm text-red-500">{errors.source}</p>}
          </div>

          {/* Target */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Target
            </label>
            <input
              type="text"
              value={formData.target}
              onChange={(e) => handleChange('target', e.target.value)}
              placeholder={formData.type === 'REGEX' ? '/new-path/$1' : '/new-path'}
              className={`w-full rounded-lg border bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-brand-purple/20 ${
                errors.target
                  ? 'border-red-500 focus:border-red-500'
                  : 'border-slate-200 dark:border-slate-700 focus:border-brand-purple'
              }`}
            />
            {errors.target && <p className="mt-1 text-sm text-red-500">{errors.target}</p>}
          </div>

          {/* Status */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              Status Code
            </label>
            <select
              value={formData.status}
              onChange={(e) => handleChange('status', e.target.value)}
              className="w-full rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 py-2.5 px-4 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-brand-purple/20 focus:border-brand-purple"
            >
              {redirectStatuses.map((status) => (
                <option key={status.value} value={status.value}>
                  {status.code} - {status.label}
                </option>
              ))}
            </select>
          </div>

          {/* Actions */}
          <div className="flex gap-3 justify-end pt-4">
            <button
              type="button"
              onClick={onCancel}
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity disabled:opacity-50 flex items-center gap-2"
            >
              {isLoading && (
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              )}
              {initialData ? 'Update' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}