import type { CdnFieldKind } from '../../generated/graphql'
import { StringListInput } from '../StringListInput'

export interface CdnFieldDescriptor {
  name: string
  label: string
  kind: CdnFieldKind
  required: boolean
  help?: string | null
  placeholder?: string | null
  defaultValue?: string | null
  options?: Array<{ value: string; label: string }> | null
}

interface CdnFieldInputProps {
  field: CdnFieldDescriptor
  value: unknown
  /** Names of the secrets already stored, so a secret input can say so instead of looking empty. */
  isSecretSet: boolean
  disabled: boolean
  onChange: (value: unknown) => void
}

const inputClass =
  'w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-slate-900 dark:text-white placeholder-slate-400 focus:border-brand-purple focus:outline-none focus:ring-1 focus:ring-brand-purple disabled:opacity-60'

/**
 * Renders one provider field from its runtime descriptor. The front-end knows
 * nothing about the providers: whatever the server describes is what appears.
 */
export function CdnFieldInput({ field, value, isSecretSet, disabled, onChange }: CdnFieldInputProps) {
  const help = field.kind === 'SECRET' && isSecretSet
    ? 'A value is stored. Leave empty to keep it, or type a new one to replace it.'
    : field.help

  return (
    <div className="mb-4">
      <label htmlFor={field.name} className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
        {field.label}
        {field.required && <span className="text-red-500 ml-1">*</span>}
      </label>

      {renderInput(field, value, disabled, onChange)}

      {help && <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{help}</p>}
    </div>
  )
}

function renderInput(
  field: CdnFieldDescriptor,
  value: unknown,
  disabled: boolean,
  onChange: (value: unknown) => void,
) {
  switch (field.kind) {
    case 'SECRET':
      return (
        <input
          id={field.name}
          type="password"
          autoComplete="new-password"
          className={inputClass}
          value={typeof value === 'string' ? value : ''}
          placeholder={field.placeholder ?? ''}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
        />
      )

    case 'BOOL':
      return (
        <label className="flex items-center gap-2">
          <input
            id={field.name}
            type="checkbox"
            className="h-4 w-4 rounded border-slate-300 text-brand-purple focus:ring-brand-purple"
            checked={value === true}
            disabled={disabled}
            onChange={(e) => onChange(e.target.checked)}
          />
          <span className="text-sm text-slate-600 dark:text-slate-400">{field.placeholder ?? 'Enabled'}</span>
        </label>
      )

    case 'INT':
      return (
        <input
          id={field.name}
          type="number"
          className={inputClass}
          value={typeof value === 'number' || typeof value === 'string' ? String(value) : ''}
          placeholder={field.placeholder ?? ''}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value === '' ? null : Number(e.target.value))}
        />
      )

    case 'SELECT':
      return (
        <select
          id={field.name}
          className={inputClass}
          value={typeof value === 'string' ? value : ''}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
        >
          {!field.required && <option value="">—</option>}
          {(field.options ?? []).map((option) => (
            <option key={option.value} value={option.value}>
              {option.label || option.value}
            </option>
          ))}
        </select>
      )

    case 'STRING_LIST':
      return (
        <StringListInput
          id={field.name}
          values={Array.isArray(value) ? (value as string[]) : []}
          placeholder={field.placeholder ?? ''}
          disabled={disabled}
          addLabel={`Add ${field.label.toLowerCase().replace(/s$/, '')}`}
          emptyLabel={`No ${field.label.toLowerCase()} yet`}
          onChange={onChange}
        />
      )

    default:
      return (
        <input
          id={field.name}
          type="text"
          className={inputClass}
          value={typeof value === 'string' ? value : ''}
          placeholder={field.placeholder ?? ''}
          disabled={disabled}
          onChange={(e) => onChange(e.target.value)}
        />
      )
  }
}
