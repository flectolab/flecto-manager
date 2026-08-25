import { useEffect, useMemo, useState } from 'react'
import { CdnFieldInput, type CdnFieldDescriptor } from './CdnFieldInput'

export interface CdnProviderDescriptor {
  type: string
  label: string
  doc?: string | null
  fields: CdnFieldDescriptor[]
}

export interface CdnConfigDraft {
  name: string
  type: string
  enabled: boolean
  values: Record<string, unknown>
}

interface CdnConfigFormProps {
  descriptors: CdnProviderDescriptor[]
  /** Set when editing: the name and the provider type are then fixed. */
  existing?: {
    name: string
    type: string
    enabled: boolean
    settings: Record<string, unknown>
    secretsSet: string[]
  }
  disabled: boolean
  submitting: boolean
  error?: string
  onSubmit: (draft: CdnConfigDraft) => void
  onCancel: () => void
}

const NAME_PATTERN = /^[a-zA-Z0-9_-]+$/

export function CdnConfigForm({
  descriptors,
  existing,
  disabled,
  submitting,
  error,
  onSubmit,
  onCancel,
}: CdnConfigFormProps) {
  const isEditing = !!existing

  const [name, setName] = useState(existing?.name ?? '')
  const [type, setType] = useState(existing?.type ?? descriptors[0]?.type ?? '')
  const [enabled, setEnabled] = useState(existing?.enabled ?? true)
  const [values, setValues] = useState<Record<string, unknown>>(existing?.settings ?? {})
  const [nameError, setNameError] = useState('')

  const descriptor = useMemo(
    () => descriptors.find((candidate) => candidate.type === type),
    [descriptors, type],
  )

  // Switching provider clears the values: they belong to the provider that
  // declared them, and nothing guarantees the next one has the same fields.
  useEffect(() => {
    if (isEditing || !descriptor) return

    setValues(defaultValues(descriptor))
  }, [descriptor, isEditing])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!isEditing && !NAME_PATTERN.test(name)) {
      setNameError('Name may only contain letters, digits, underscores and hyphens')
      return
    }

    setNameError('')
    onSubmit({ name, type, enabled, values: normalize(descriptor, values) })
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
      <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
        {isEditing ? `Edit ${existing?.name}` : 'Add a CDN'}
      </h3>

      {error && (
        <div className="mb-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        </div>
      )}

      <div className="mb-4">
        <label htmlFor="cdn-name" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
          Name<span className="text-red-500 ml-1">*</span>
        </label>
        <input
          id="cdn-name"
          type="text"
          required
          className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-slate-900 dark:text-white focus:border-brand-purple focus:outline-none focus:ring-1 focus:ring-brand-purple disabled:opacity-60"
          value={name}
          placeholder="prod"
          disabled={disabled || isEditing}
          onChange={(e) => setName(e.target.value)}
        />
        {nameError && <p className="mt-1 text-xs text-red-600 dark:text-red-400">{nameError}</p>}
      </div>

      <div className="mb-4">
        <label htmlFor="cdn-type" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
          Provider<span className="text-red-500 ml-1">*</span>
        </label>
        <select
          id="cdn-type"
          className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-slate-900 dark:text-white focus:border-brand-purple focus:outline-none focus:ring-1 focus:ring-brand-purple disabled:opacity-60"
          value={type}
          // The stored values belong to the original provider, so the type is fixed
          // once the configuration exists.
          disabled={disabled || isEditing}
          onChange={(e) => setType(e.target.value)}
        >
          {descriptors.map((candidate) => (
            <option key={candidate.type} value={candidate.type}>
              {candidate.label}
            </option>
          ))}
        </select>
        {descriptor?.doc && <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{descriptor.doc}</p>}
      </div>

      <div className="mb-4">
        <span className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Status</span>
        <label className="flex items-center gap-2">
          <input
            type="checkbox"
            className="h-4 w-4 rounded border-slate-300 text-brand-purple focus:ring-brand-purple"
            checked={enabled}
            disabled={disabled}
            onChange={(e) => setEnabled(e.target.checked)}
          />
          <span className="text-sm text-slate-600 dark:text-slate-400">Enabled</span>
        </label>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
          When disabled, the zone and its credentials are kept but publishing the project no longer
          purges this CDN.
        </p>
      </div>

      <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
        {descriptor?.fields.map((field) => (
          <CdnFieldInput
            key={field.name}
            field={field}
            value={values[field.name]}
            isSecretSet={existing?.secretsSet.includes(field.name) ?? false}
            disabled={disabled}
            onChange={(value) => setValues((current) => ({ ...current, [field.name]: value }))}
          />
        ))}
      </div>

      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={disabled || submitting}
          className="rounded-lg bg-brand-purple px-4 py-2 text-white font-medium hover:bg-brand-purple/90 disabled:opacity-60"
        >
          {submitting ? 'Saving…' : 'Save'}
        </button>
        <button
          type="button"
          onClick={onCancel}
          className="rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
        >
          Cancel
        </button>
      </div>
    </form>
  )
}

/**
 * normalize trims the list fields and drops the entries left empty. It happens on
 * submit and not while typing, so clearing an input does not make its row vanish.
 */
function normalize(
  descriptor: CdnProviderDescriptor | undefined,
  values: Record<string, unknown>,
): Record<string, unknown> {
  if (!descriptor) return values

  const cleaned = { ...values }
  for (const field of descriptor.fields) {
    if (field.kind !== 'STRING_LIST') continue

    const list = cleaned[field.name]
    if (!Array.isArray(list)) continue

    cleaned[field.name] = list.map((entry) => String(entry).trim()).filter((entry) => entry !== '')
  }

  return cleaned
}

/** defaultValues seeds a new configuration from what the provider declares. */
function defaultValues(descriptor: CdnProviderDescriptor): Record<string, unknown> {
  const seeded: Record<string, unknown> = {}

  for (const field of descriptor.fields) {
    if (field.defaultValue == null) continue

    switch (field.kind) {
      case 'BOOL':
        seeded[field.name] = field.defaultValue === 'true'
        break
      case 'INT':
        seeded[field.name] = Number(field.defaultValue)
        break
      default:
        seeded[field.name] = field.defaultValue
    }
  }

  return seeded
}
