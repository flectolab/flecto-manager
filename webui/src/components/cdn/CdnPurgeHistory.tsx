import { useQuery } from '@apollo/client/react'
import { GetProjectCdnPurgeJobsDocument } from '../../generated/graphql'
import { RelativeTime } from '../RelativeTime'

interface CdnPurgeHistoryProps {
  namespaceCode: string
  projectCode: string
}

const statusStyles: Record<string, string> = {
  WAITING_AGENTS: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  RUNNING: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  SUCCEEDED: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  PARTIAL: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  FAILED: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  SKIPPED: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300',
}

const statusLabels: Record<string, string> = {
  WAITING_AGENTS: 'Waiting for agents',
  RUNNING: 'Running',
  SUCCEEDED: 'Succeeded',
  PARTIAL: 'Partial',
  FAILED: 'Failed',
  SKIPPED: 'Skipped',
}

const skipReasonLabels: Record<string, string> = {
  PATTERN: 'regex',
  NO_HOST_MATCH: 'hostname not served',
  NO_HOST: 'no hostname configured',
  INVALID: 'invalid',
  TOO_MANY_TARGETS: 'too many URLs',
  CONFIG_DISABLED: 'configuration disabled',
}

export function CdnPurgeHistory({ namespaceCode, projectCode }: CdnPurgeHistoryProps) {
  const { data, loading } = useQuery(GetProjectCdnPurgeJobsDocument, {
    variables: { namespaceCode, projectCode, limit: 20 },
    fetchPolicy: 'cache-and-network',
    // Jobs progress in the background.
    pollInterval: 15000,
  })

  const jobs = data?.projectCdnPurgeJobs ?? []

  if (loading && jobs.length === 0) {
    return <p className="text-sm text-slate-500 dark:text-slate-400">Loading…</p>
  }

  if (jobs.length === 0) {
    return (
      <p className="text-sm text-slate-500 dark:text-slate-400">
        No purge yet. Publishing the project queues one for each enabled CDN.
      </p>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-xs uppercase tracking-wide text-slate-500 dark:text-slate-400">
            <th className="pb-2 pr-4 font-medium">When</th>
            <th className="pb-2 pr-4 font-medium">CDN</th>
            <th className="pb-2 pr-4 font-medium">Version</th>
            <th className="pb-2 pr-4 font-medium">Status</th>
            <th className="pb-2 font-medium">URLs</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 dark:divide-slate-700">
          {jobs.map((job) => (
            <tr key={job.id} className="align-top">
              <td className="py-2 pr-4 whitespace-nowrap text-slate-600 dark:text-slate-400">
                <RelativeTime date={job.finishedAt ?? job.createdAt} />
              </td>
              <td className="py-2 pr-4 text-slate-900 dark:text-white">{job.cdnConfigName}</td>
              <td className="py-2 pr-4 text-slate-600 dark:text-slate-400">
                {/* A manual purge answers to no publish, so it carries no version. */}
                {job.projectVersion > 0 ? `v${job.projectVersion}` : 'manual'}
              </td>
              <td className="py-2 pr-4">
                <span
                  className={`inline-flex px-2 py-0.5 text-xs font-semibold rounded-full ${
                    statusStyles[job.status] ?? statusStyles.SKIPPED
                  }`}
                >
                  {statusLabels[job.status] ?? job.status}
                </span>
                {job.status === 'WAITING_AGENTS' && (
                  // Without it a job waiting normally looks the same as a stuck one.
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {deadlineLabel(job.deadlineAt)}
                  </p>
                )}
                {job.skipReason && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                    {skipReasonLabels[job.skipReason] ?? job.skipReason}
                  </p>
                )}
                {!job.waitedForAgents && (
                  <p className="mt-1 text-xs text-amber-600 dark:text-amber-400">
                    Purged before the agents caught up
                  </p>
                )}
                {job.error && (
                  <p className="mt-1 max-w-md text-xs text-red-600 dark:text-red-400 break-words">{job.error}</p>
                )}
              </td>
              <td className="py-2 text-slate-600 dark:text-slate-400">
                {job.purged}/{job.requested}
                {job.failed > 0 && <span className="text-red-600 dark:text-red-400"> · {job.failed} failed</span>}
                {skippedLabel(job.skipped) && (
                  <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{skippedLabel(job.skipped)}</p>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function skippedLabel(skipped: unknown): string {
  if (!skipped || typeof skipped !== 'object') return ''

  const entries = Object.entries(skipped as Record<string, number>).filter(([, count]) => count > 0)
  if (entries.length === 0) return ''

  return `skipped: ${entries
    .map(([reason, count]) => `${count} ${skipReasonLabels[reason] ?? reason}`)
    .join(' · ')}`
}

/** Shows the clock time, which stays right between two polls, plus the remaining delay. */
function deadlineLabel(deadlineAt: string): string {
  const deadline = new Date(deadlineAt)
  const at = deadline.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
  const minutes = Math.ceil((deadline.getTime() - Date.now()) / 60000)

  if (minutes <= 0) return `Purges anyway at the next run (deadline ${at})`
  return `Purges anyway at ${at} (in ${minutes} min) if the agents have not caught up`
}
