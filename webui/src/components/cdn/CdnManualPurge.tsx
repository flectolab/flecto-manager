import { useState } from 'react'
import { useMutation } from '@apollo/client/react'
import { GetProjectCdnPurgeJobsDocument, PurgeProjectCdnUrlsDocument } from '../../generated/graphql'
import { StringListInput } from '../StringListInput'

interface CdnManualPurgeProps {
  namespaceCode: string
  projectCode: string
  /** Names of the enabled configurations, which are the only ones a purge can go through. */
  configNames: string[]
}

/**
 * Queues a purge of explicit URLs, outside of any publish.
 *
 * It is the way out when a purge was skipped or failed: someone reads the history,
 * sees what was missed, and sends those addresses through by hand.
 */
export function CdnManualPurge({ namespaceCode, projectCode, configNames }: CdnManualPurgeProps) {
  const [open, setOpen] = useState(false)
  const [chosenName, setChosenName] = useState('')
  // Derived on every render rather than kept in state: the list changes under the
  // form when a configuration is created, disabled or deleted, and a choice no
  // longer in it would be sent as is.
  const configName = configNames.includes(chosenName) ? chosenName : (configNames[0] ?? '')
  const [urls, setUrls] = useState<string[]>([''])
  const [error, setError] = useState('')
  const [queued, setQueued] = useState('')

  // Refetching the history from here shows the new job straight away, instead of
  // at the next poll.
  const [purgeUrls, { loading }] = useMutation(PurgeProjectCdnUrlsDocument, {
    refetchQueries: [GetProjectCdnPurgeJobsDocument],
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setQueued('')

    const cleaned = urls.map((url) => url.trim()).filter((url) => url !== '')
    if (cleaned.length === 0) {
      setError('Add at least one URL')
      return
    }

    try {
      const result = await purgeUrls({
        variables: { namespaceCode, projectCode, name: configName, urls: cleaned },
      })
      const job = result.data?.purgeProjectCdnUrls
      setQueued(`${job?.requested ?? 0} URL(s) queued`)
      setUrls([''])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to queue the purge')
    }
  }

  if (configNames.length === 0) {
    return null
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
      >
        Purge URLs manually
      </button>
    )
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6"
    >
      <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-1">Purge URLs manually</h3>
      <p className="mb-4 text-sm text-slate-600 dark:text-slate-400">
        Full URLs as they are served. Only the ones whose hostname the chosen CDN covers are sent; the
        rest are reported as skipped.
      </p>

      {error && (
        <div className="mb-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        </div>
      )}

      {queued && (
        <div className="mb-4 rounded-lg bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 p-3">
          <p className="text-sm text-green-600 dark:text-green-400">{queued}</p>
        </div>
      )}

      <div className="mb-4">
        <label htmlFor="purge-config" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
          CDN
        </label>
        <select
          id="purge-config"
          value={configName}
          onChange={(e) => setChosenName(e.target.value)}
          className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-900 px-3 py-2 text-slate-900 dark:text-white focus:border-brand-purple focus:outline-none focus:ring-1 focus:ring-brand-purple"
        >
          {configNames.map((name) => (
            <option key={name} value={name}>
              {name}
            </option>
          ))}
        </select>
      </div>

      <div className="mb-4">
        <label htmlFor="purge-urls" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
          URLs
        </label>
        <StringListInput
          id="purge-urls"
          values={urls}
          placeholder="https://www.example.com/promo"
          addLabel="Add URL"
          emptyLabel="No URL yet"
          onChange={setUrls}
        />
      </div>

      <div className="flex items-center gap-3">
        <button
          type="submit"
          disabled={loading}
          className="rounded-lg bg-brand-purple px-4 py-2 text-white font-medium hover:bg-brand-purple/90 disabled:opacity-60"
        >
          {loading ? 'Queueing…' : 'Queue purge'}
        </button>
        <button
          type="button"
          onClick={() => {
            setOpen(false)
            setError('')
            setQueued('')
          }}
          className="rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-2 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
        >
          Close
        </button>
      </div>
    </form>
  )
}
