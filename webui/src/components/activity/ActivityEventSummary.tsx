import { useState } from 'react'
import type { ActivityAction, ActivityResource } from '../../generated/graphql'
import { useIsTruncated } from '../../hooks/useIsTruncated'
import { ActivityDetailModal } from './ActivityDetailModal'
import { describeActivityEvent, type ActivityFieldDescriptor } from './describeActivityEvent'

/** `label: value`, or `label: before → after` when the field changed. */
function InlineField({ field }: { field: ActivityFieldDescriptor }) {
  return (
    <span className="whitespace-nowrap">
      <span className="text-slate-500 dark:text-slate-400">{field.label}: </span>
      {field.before !== undefined && (
        <>
          <span className="font-mono line-through text-slate-400 dark:text-slate-500">
            {field.before}
          </span>
          <span className="text-slate-400 dark:text-slate-500"> &rarr; </span>
        </>
      )}
      <span className="font-mono text-slate-900 dark:text-white">{field.after}</span>
    </span>
  )
}

interface ActivityEventSummaryProps {
  resource: ActivityResource
  action: ActivityAction
  data: unknown
}

/**
 * One line per event: every field as `label: value`, changes as
 * `label: before → after`. The line never wraps; when it does not fit it is
 * truncated and a button opens the full detail in a modal.
 */
export function ActivityEventSummary({ resource, action, data }: ActivityEventSummaryProps) {
  const [modalOpen, setModalOpen] = useState(false)
  const { ref, isTruncated } = useIsTruncated<HTMLDivElement>()

  const description = describeActivityEvent(resource, action, data)

  // Import errors and unknown payloads have no place on a single line, so they get
  // the button whether or not the line itself overflows.
  const hasExtra = (description.errors?.length ?? 0) > 0 || description.raw !== undefined
  const showButton = isTruncated || hasExtra

  return (
    <>
      <div className="flex items-center gap-3 min-w-0">
        <div ref={ref} className="flex-1 min-w-0 truncate text-sm">
          {description.fields.map((field, index) => (
            <span key={field.label}>
              {index > 0 && <span className="text-slate-300 dark:text-slate-600"> · </span>}
              <InlineField field={field} />
            </span>
          ))}
        </div>
        {showButton && (
          <button
            onClick={() => setModalOpen(true)}
            className="shrink-0 text-xs font-medium text-brand-purple hover:underline whitespace-nowrap"
          >
            View changes
          </button>
        )}
      </div>

      {modalOpen && (
        <ActivityDetailModal
          resource={resource}
          action={action}
          description={description}
          onClose={() => setModalOpen(false)}
        />
      )}
    </>
  )
}
