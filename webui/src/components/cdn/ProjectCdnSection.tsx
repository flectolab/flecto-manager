import { useState } from 'react'
import { useQuery, useMutation } from '@apollo/client/react'
import {
  GetCdnProviderTypesDocument,
  GetProjectCdnConfigsDocument,
  CreateProjectCdnConfigDocument,
  UpdateProjectCdnConfigDocument,
  DeleteProjectCdnConfigDocument,
  TestProjectCdnConfigDocument,
} from '../../generated/graphql'
import { RelativeTime } from '../RelativeTime'
import { CdnConfigForm, type CdnConfigDraft } from './CdnConfigForm'
import { CdnPurgeHistory } from './CdnPurgeHistory'
import { CdnManualPurge } from './CdnManualPurge'

type TestOutcome = { success: boolean; error?: string | null }

interface ProjectCdnSectionProps {
  namespaceCode: string
  projectCode: string
  canWrite: boolean
}

/** Lives in the admin section because a configuration holds third-party credentials. */
export function ProjectCdnSection({ namespaceCode, projectCode, canWrite }: ProjectCdnSectionProps) {
  const [editing, setEditing] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [formError, setFormError] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [testOutcomes, setTestOutcomes] = useState<Record<string, TestOutcome>>({})
  const [testing, setTesting] = useState<string | null>(null)

  const { data: providerData, loading: providerLoading } = useQuery(GetCdnProviderTypesDocument)
  const { data, loading, refetch } = useQuery(GetProjectCdnConfigsDocument, {
    variables: { namespaceCode, projectCode },
  })

  const [createConfig, { loading: creatingLoading }] = useMutation(CreateProjectCdnConfigDocument)
  const [updateConfig, { loading: updatingLoading }] = useMutation(UpdateProjectCdnConfigDocument)
  const [deleteConfig] = useMutation(DeleteProjectCdnConfigDocument)
  const [testConfig] = useMutation(TestProjectCdnConfigDocument)

  const descriptors = providerData?.cdnProviderTypes ?? []
  const encryptionAvailable = providerData?.cdnEncryptionAvailable ?? true
  const configs = data?.projectCdnConfigs ?? []

  const handleSubmit = async (draft: CdnConfigDraft) => {
    setFormError('')
    try {
      if (editing) {
        await updateConfig({
          variables: {
            namespaceCode,
            projectCode,
            name: editing,
            input: { enabled: draft.enabled, values: draft.values },
          },
        })
      } else {
        await createConfig({
          variables: {
            namespaceCode,
            projectCode,
            input: { name: draft.name, type: draft.type, enabled: draft.enabled, values: draft.values },
          },
        })
      }
      setEditing(null)
      setCreating(false)
      refetch()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save the configuration')
    }
  }

  const handleToggleEnabled = async (name: string, enabled: boolean, settings: Record<string, unknown>) => {
    // Disabling keeps the zone and the credentials, it only stops the purges.
    await updateConfig({
      variables: {
        namespaceCode,
        projectCode,
        name,
        input: { enabled, values: settings },
      },
    })
    refetch()
  }

  const handleDelete = async (name: string) => {
    await deleteConfig({ variables: { namespaceCode, projectCode, name } })
    setDeleteConfirm(null)
    refetch()
  }

  const handleTest = async (name: string) => {
    setTesting(name)
    try {
      const result = await testConfig({
        variables: { namespaceCode, projectCode, name },
      })
      const outcome = result.data?.testProjectCdnConfig
      if (outcome) {
        setTestOutcomes((current) => ({ ...current, [name]: outcome }))
      }
    } catch (err) {
      setTestOutcomes((current) => ({
        ...current,
        [name]: { success: false, error: err instanceof Error ? err.message : 'Test failed' },
      }))
    } finally {
      setTesting(null)
    }
  }

  if (providerLoading || loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  const editingConfig = configs.find((config) => config.name === editing)

  return (
    <div>
      <div className="mb-6 flex items-start justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">CDN configurations</h3>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            Cache invalidated on publish, for the redirects and pages that carry an address a CDN can purge.
          </p>
        </div>
        {canWrite && !creating && !editing && (
          <button
            onClick={() => {
              setCreating(true)
              setFormError('')
            }}
            disabled={!encryptionAvailable || descriptors.length === 0}
            className="shrink-0 rounded-lg bg-brand-purple px-4 py-2 text-white font-medium hover:bg-brand-purple/90 disabled:opacity-60"
          >
            Add a CDN
          </button>
        )}
      </div>

      {!encryptionAvailable && (
        <div className="mb-6 rounded-xl bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-4">
          <p className="text-sm text-amber-800 dark:text-amber-300">
            This instance has no encryption key, so CDN credentials cannot be stored. Set{' '}
            <code className="font-mono">security.encryption_key</code> in the configuration — generate one with{' '}
            <code className="font-mono">flecto-manager security generate-key</code>.
          </p>
        </div>
      )}

      {(creating || editing) && (
        <div className="mb-6">
          <CdnConfigForm
            descriptors={descriptors}
            existing={
              editingConfig
                ? {
                    name: editingConfig.name,
                    type: editingConfig.type,
                    enabled: editingConfig.enabled,
                    settings: editingConfig.settings ?? {},
                    secretsSet: editingConfig.secretsSet,
                  }
                : undefined
            }
            disabled={!canWrite}
            submitting={creatingLoading || updatingLoading}
            error={formError}
            onSubmit={handleSubmit}
            onCancel={() => {
              setCreating(false)
              setEditing(null)
              setFormError('')
            }}
          />
        </div>
      )}

      {configs.length === 0 && !creating ? (
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-8 text-center">
          <p className="text-slate-600 dark:text-slate-400">
            No CDN configured. Publishing changes nothing in any cache until one is added.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {configs.map((config) => {
            const hosts = Array.isArray(config.settings?.hosts) ? (config.settings.hosts as string[]) : []
            const outcome = testOutcomes[config.name]

            return (
              <div
                key={config.id}
                className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6"
              >
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <h3 className="text-lg font-semibold text-slate-900 dark:text-white truncate">{config.name}</h3>
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-300">
                        {config.type}
                      </span>
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          config.enabled
                            ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                            : 'bg-slate-100 text-slate-500 dark:bg-slate-700 dark:text-slate-400'
                        }`}
                      >
                        {config.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </div>

                    <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
                      {hosts.length > 0 ? hosts.join(', ') : 'No hostname declared'}
                    </p>

                    {config.secretsSet.length > 0 && (
                      <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                        Stored secrets: {config.secretsSet.join(', ')}
                      </p>
                    )}

                    <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                      Updated <RelativeTime date={config.updatedAt} />
                    </p>

                    {outcome && (
                      <p
                        className={`mt-2 text-sm ${
                          outcome.success ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'
                        }`}
                      >
                        {outcome.success ? 'Credentials accepted' : outcome.error}
                      </p>
                    )}
                  </div>

                  {canWrite && (
                    <div className="flex shrink-0 items-center gap-2">
                      <button
                        onClick={() => handleTest(config.name)}
                        disabled={testing === config.name}
                        className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-60"
                      >
                        {testing === config.name ? 'Testing…' : 'Test'}
                      </button>
                      <button
                        onClick={() => handleToggleEnabled(config.name, !config.enabled, config.settings ?? {})}
                        className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
                      >
                        {config.enabled ? 'Disable' : 'Enable'}
                      </button>
                      <button
                        onClick={() => {
                          setEditing(config.name)
                          setCreating(false)
                          setFormError('')
                        }}
                        className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => setDeleteConfirm(config.name)}
                        className="rounded-lg border border-red-300 dark:border-red-800 px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
                      >
                        Delete
                      </button>
                    </div>
                  )}
                </div>

                {deleteConfirm === config.name && (
                  <div className="mt-4 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3">
                    <p className="text-sm text-red-700 dark:text-red-400">
                      Delete <strong>{config.name}</strong> and its stored credentials? Publishes stop purging this CDN.
                    </p>
                    <div className="mt-3 flex items-center gap-2">
                      <button
                        onClick={() => handleDelete(config.name)}
                        className="rounded-lg bg-red-600 px-3 py-1.5 text-sm text-white hover:bg-red-700"
                      >
                        Delete
                      </button>
                      <button
                        onClick={() => setDeleteConfirm(null)}
                        className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700"
                      >
                        Cancel
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      <div className="mt-8">
        <div className="mb-4">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Purge history</h3>
          <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
            One entry per publish and per CDN. A job waits for the agents to serve the published
            version before purging, so it can sit here for a while before it runs.
          </p>
        </div>

        {/* Full width: closed it renders a button, open it renders a form. */}
        {canWrite && (
          <div className="mb-4">
            <CdnManualPurge
              namespaceCode={namespaceCode}
              projectCode={projectCode}
              configNames={configs.filter((config) => config.enabled).map((config) => config.name)}
            />
          </div>
        )}

        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-6">
          <CdnPurgeHistory
            namespaceCode={namespaceCode}
            projectCode={projectCode}
          />
        </div>
      </div>
    </div>
  )
}
