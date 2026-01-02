import { useState, useRef } from 'react'
import { useMutation, useApolloClient } from '@apollo/client/react'
import {
  ImportRedirectDraftDocument,
  GetProjectRedirectsDocument,
  GetProjectDocument,
} from '../../generated/graphql'
import type { ImportRedirectDraftMutation, ImportErrorReason } from '../../generated/graphql'
import { PathCell } from '../PathCell'

const MAX_FILE_SIZE = 2 * 1024 * 1024 // 2MB
const ERRORS_PER_PAGE = 10

interface ImportModalProps {
  namespaceCode: string
  projectCode: string
  onClose: () => void
  onSuccess: () => void
}

type ImportResult = NonNullable<ImportRedirectDraftMutation['importRedirectDraft']>

const errorReasonLabels: Record<ImportErrorReason, string> = {
  'INVALID_FORMAT': 'Invalid format',
  'INVALID_REDIRECT': 'Invalid redirect',
  'INVALID_TYPE': 'Invalid type',
  'INVALID_STATUS': 'Invalid status',
  'EMPTY_SOURCE': 'Empty source',
  'EMPTY_TARGET': 'Empty target',
  'DUPLICATE_SOURCE_IN_FILE': 'Duplicate in file',
  'SOURCE_ALREADY_EXISTS': 'Source exists',
  'DATABASE_ERROR': 'Database error',
}

function exportErrorsToCsv(errors: ImportResult['errors']) {
  const headers = ['line', 'source', 'reason', 'message']
  const rows = errors.map(err => [
    err.line.toString(),
    err.source || '',
    errorReasonLabels[err.reason] || err.reason,
    err.message,
  ])

  const csvContent = [
    headers.join('\t'),
    ...rows.map(row => row.map(cell => `"${cell.replace(/"/g, '""')}"`).join('\t')),
  ].join('\n')

  const blob = new Blob([csvContent], { type: 'text/tab-separated-values;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `import-errors-${new Date().toISOString().slice(0, 10)}.tsv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

export function ImportModal({ namespaceCode, projectCode, onClose, onSuccess }: ImportModalProps) {
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [dragOver, setDragOver] = useState(false)
  const [importResult, setImportResult] = useState<ImportResult | null>(null)
  const [errorPage, setErrorPage] = useState(0)
  const [showFormatHelp, setShowFormatHelp] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const client = useApolloClient()
  const [importRedirect, { loading }] = useMutation(ImportRedirectDraftDocument)

  const handleFileSelect = (selectedFile: File | null) => {
    if (selectedFile) {
      const ext = selectedFile.name.toLowerCase()
      if (!ext.endsWith('.csv') && !ext.endsWith('.tsv')) {
        setError('Invalid file type. Only .csv and .tsv files are allowed.')
        return
      }
      if (selectedFile.size > MAX_FILE_SIZE) {
        setError(`File too large. Maximum size is 2MB. Your file is ${(selectedFile.size / 1024 / 1024).toFixed(2)}MB.`)
        return
      }
      setFile(selectedFile)
      setError(null)
      setImportResult(null)
    }
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const droppedFile = e.dataTransfer.files[0]
    handleFileSelect(droppedFile)
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(true)
  }

  const handleDragLeave = () => {
    setDragOver(false)
  }

  const handleImport = async () => {
    if (!file) return

    try {
      setError(null)
      const result = await importRedirect({
        variables: {
          namespaceCode,
          projectCode,
          file,
          input: { overwrite: true },
        },
      })

      const importData = result.data?.importRedirectDraft
      if (importData) {
        setImportResult(importData)
        setErrorPage(0)
      }
    } catch (err) {
      console.error('Import error:', err)
      const message = err instanceof Error ? err.message : 'Import failed'
      setError(message)
    }
  }

  const handleClose = async () => {
    if (importResult?.importedCount && importResult.importedCount > 0) {
      // Refetch data only when closing and if something was imported
      await client.refetchQueries({
        include: [GetProjectRedirectsDocument, GetProjectDocument],
      })
      onSuccess()
    }
    onClose()
  }

  // Pagination for errors
  const paginatedErrors = importResult?.errors.slice(
    errorPage * ERRORS_PER_PAGE,
    (errorPage + 1) * ERRORS_PER_PAGE
  ) || []
  const totalErrorPages = Math.ceil((importResult?.errors.length || 0) / ERRORS_PER_PAGE)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={handleClose} />
      <div className="relative w-full max-w-2xl mx-4 rounded-xl bg-white dark:bg-slate-800 shadow-xl border border-slate-200 dark:border-slate-700 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700 px-6 py-4 flex items-center justify-between z-10">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
            {importResult ? 'Import Results' : 'Import Redirects'}
          </h3>
          <button
            onClick={handleClose}
            className="p-2 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 dark:hover:text-slate-300 dark:hover:bg-slate-700 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {importResult ? (
            // Import Results View
            <>
              {/* Summary Stats */}
              <div className={`rounded-lg p-4 ${
                importResult.success
                  ? 'bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800'
                  : 'bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800'
              }`}>
                <div className="flex items-center gap-3 mb-4">
                  {importResult.success ? (
                    <svg className="w-6 h-6 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  ) : (
                    <svg className="w-6 h-6 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                  )}
                  <span className={`font-semibold ${
                    importResult.success
                      ? 'text-green-800 dark:text-green-300'
                      : 'text-amber-800 dark:text-amber-300'
                  }`}>
                    {importResult.success ? 'Import completed successfully!' : 'Import completed with errors'}
                  </span>
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
                  <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-slate-800/50">
                    <div className="text-2xl font-bold text-slate-900 dark:text-white">{importResult.totalLines}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">Total Lines</div>
                  </div>
                  <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-slate-800/50">
                    <div className="text-2xl font-bold text-green-600 dark:text-green-400">{importResult.importedCount}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">Imported</div>
                  </div>
                  <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-slate-800/50">
                    <div className="text-2xl font-bold text-slate-500 dark:text-slate-400">{importResult.skippedCount}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">Skipped</div>
                  </div>
                  <div className="text-center p-3 rounded-lg bg-white/50 dark:bg-slate-800/50">
                    <div className="text-2xl font-bold text-red-600 dark:text-red-400">{importResult.errorCount}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">Errors</div>
                  </div>
                </div>
              </div>

              {/* Errors Table */}
              {importResult.errors.length > 0 && (
                <div className="rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
                  <div className="bg-slate-50 dark:bg-slate-700/50 px-4 py-3 border-b border-slate-200 dark:border-slate-700 flex items-center justify-between">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                      <svg className="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                      </svg>
                      Errors ({importResult.errors.length})
                    </h4>
                    <button
                      onClick={() => exportErrorsToCsv(importResult.errors)}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border border-slate-200 dark:border-slate-600 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                      </svg>
                      Export
                    </button>
                  </div>

                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead className="bg-slate-50 dark:bg-slate-700/30">
                        <tr>
                          <th className="text-left py-2 px-4 text-slate-600 dark:text-slate-400 font-medium">Line</th>
                          <th className="text-left py-2 px-4 text-slate-600 dark:text-slate-400 font-medium">Source</th>
                          <th className="text-left py-2 px-4 text-slate-600 dark:text-slate-400 font-medium">Reason</th>
                          <th className="text-left py-2 px-4 text-slate-600 dark:text-slate-400 font-medium">Message</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                        {paginatedErrors.map((err, idx) => (
                          <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-slate-700/30">
                            <td className="py-2 px-4 text-slate-900 dark:text-white font-mono">{err.line}</td>
                            <td className="py-2 px-4">
                              <PathCell value={err.source || ''} maxWidth="max-w-[150px]" className="text-slate-700 dark:text-slate-300 text-xs" />
                            </td>
                            <td className="py-2 px-4">
                              <span className="px-2 py-0.5 rounded text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400">
                                {errorReasonLabels[err.reason]}
                              </span>
                            </td>
                            <td className="py-2 px-4 text-slate-600 dark:text-slate-400 text-xs max-w-[200px]" title={err.message}>
                              <span className="line-clamp-2">{err.message}</span>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Pagination */}
                  {totalErrorPages > 1 && (
                    <div className="bg-slate-50 dark:bg-slate-700/30 px-4 py-3 border-t border-slate-200 dark:border-slate-700 flex items-center justify-between">
                      <span className="text-sm text-slate-500 dark:text-slate-400">
                        Showing {errorPage * ERRORS_PER_PAGE + 1} - {Math.min((errorPage + 1) * ERRORS_PER_PAGE, importResult.errors.length)} of {importResult.errors.length}
                      </span>
                      <div className="flex gap-2">
                        <button
                          onClick={() => setErrorPage(p => Math.max(0, p - 1))}
                          disabled={errorPage === 0}
                          className="px-3 py-1 text-sm rounded border border-slate-200 dark:border-slate-600 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          Previous
                        </button>
                        <button
                          onClick={() => setErrorPage(p => Math.min(totalErrorPages - 1, p + 1))}
                          disabled={errorPage >= totalErrorPages - 1}
                          className="px-3 py-1 text-sm rounded border border-slate-200 dark:border-slate-600 text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                          Next
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </>
          ) : (
            // File Upload View
            <>
              {/* Collapsible Format Help Section */}
              <div className="rounded-lg border border-slate-200 dark:border-slate-600 overflow-hidden">
                <button
                  type="button"
                  onClick={() => setShowFormatHelp(!showFormatHelp)}
                  className="w-full px-4 py-3 bg-slate-50 dark:bg-slate-700/50 flex items-center justify-between text-left hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                >
                  <span className="text-sm font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                    <svg className="w-5 h-5 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    File Format Help
                  </span>
                  <svg
                    className={`w-5 h-5 text-slate-400 transition-transform duration-200 ${showFormatHelp ? 'rotate-180' : ''}`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>

                {showFormatHelp && (
                  <div className="border-t border-slate-200 dark:border-slate-600 divide-y divide-slate-200 dark:divide-slate-600">
                    {/* File Format Rules */}
                    <div className="p-4">
                      <h5 className="text-sm font-medium text-slate-900 dark:text-white mb-2">Format Requirements</h5>
                      <div className="space-y-2 text-sm">
                        <div>
                          <span className="font-medium text-slate-700 dark:text-slate-300">Separator:</span>
                          <span className="ml-2 px-2 py-0.5 rounded bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-400 font-mono text-xs">TAB</span>
                          <span className="ml-2 text-slate-500 dark:text-slate-400">(not comma, to support regex patterns)</span>
                        </div>
                        <div>
                          <span className="font-medium text-slate-700 dark:text-slate-300">File extension:</span>
                          <span className="ml-2 text-slate-600 dark:text-slate-400">.csv or .tsv</span>
                        </div>
                        <div>
                          <span className="font-medium text-slate-700 dark:text-slate-300">Required headers:</span>
                          <div className="mt-1 flex gap-2">
                            {['type', 'source', 'target', 'status'].map((header) => (
                              <span key={header} className="px-2 py-0.5 rounded bg-slate-200 dark:bg-slate-600 text-slate-700 dark:text-slate-300 font-mono text-xs">
                                {header}
                              </span>
                            ))}
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Column Values */}
                    <div className="p-4">
                      <h5 className="text-sm font-medium text-slate-900 dark:text-white mb-2">Column Values</h5>
                      <div className="space-y-2 text-sm">
                        <div>
                          <span className="font-medium text-slate-700 dark:text-slate-300">type:</span>
                          <div className="mt-1 flex flex-wrap gap-2">
                            {['BASIC', 'BASIC_HOST', 'REGEX', 'REGEX_HOST'].map((type) => (
                              <span key={type} className="px-2 py-0.5 rounded bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-400 font-mono text-xs">
                                {type}
                              </span>
                            ))}
                          </div>
                        </div>
                        <div>
                          <span className="font-medium text-slate-700 dark:text-slate-300">status:</span>
                          <div className="mt-1 flex flex-wrap gap-2">
                            {[
                              { label: 'MOVED_PERMANENT', code: '301' },
                              { label: 'FOUND', code: '302' },
                              { label: 'TEMPORARY_REDIRECT', code: '307' },
                              { label: 'PERMANENT_REDIRECT', code: '308' },
                            ].map((status) => (
                              <span key={status.label} className="px-2 py-0.5 rounded bg-cyan-100 dark:bg-cyan-900/30 text-cyan-700 dark:text-cyan-400 font-mono text-xs">
                                {status.label} <span className="text-slate-500">({status.code})</span>
                              </span>
                            ))}
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Examples */}
                    <div className="p-4">
                      <h5 className="text-sm font-medium text-slate-900 dark:text-white mb-2">Examples</h5>
                      <div className="overflow-x-auto">
                        <table className="w-full text-xs font-mono">
                          <thead>
                            <tr className="border-b border-slate-300 dark:border-slate-600">
                              <th className="text-left py-2 pr-4 text-slate-600 dark:text-slate-400">type</th>
                              <th className="text-left py-2 pr-4 text-slate-600 dark:text-slate-400">source</th>
                              <th className="text-left py-2 pr-4 text-slate-600 dark:text-slate-400">target</th>
                              <th className="text-left py-2 text-slate-600 dark:text-slate-400">status</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                            <tr>
                              <td className="py-2 pr-4 text-purple-600 dark:text-purple-400">BASIC</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">/old-page</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">/new-page</td>
                              <td className="py-2 text-cyan-600 dark:text-cyan-400">301</td>
                            </tr>
                            <tr>
                              <td className="py-2 pr-4 text-purple-600 dark:text-purple-400">BASIC_HOST</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">old.example.com/path</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">https://new.example.com/path</td>
                              <td className="py-2 text-cyan-600 dark:text-cyan-400">MOVED_PERMANENT</td>
                            </tr>
                            <tr>
                              <td className="py-2 pr-4 text-purple-600 dark:text-purple-400">REGEX</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">/product/([0-9]+)</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">/item/$1</td>
                              <td className="py-2 text-cyan-600 dark:text-cyan-400">302</td>
                            </tr>
                            <tr>
                              <td className="py-2 pr-4 text-purple-600 dark:text-purple-400">REGEX_HOST</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">([a-z]+).old.com/(.*)</td>
                              <td className="py-2 pr-4 text-slate-700 dark:text-slate-300">https://$1.new.com/$2</td>
                              <td className="py-2 text-cyan-600 dark:text-cyan-400">308</td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    </div>
                  </div>
                )}
              </div>

              {/* Behavior Note */}
              <div className="rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-4">
                <div className="flex gap-3">
                  <svg className="w-5 h-5 text-amber-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                  </svg>
                  <div className="text-sm text-amber-800 dark:text-amber-300">
                    <strong>Note:</strong> If a source already exists, the existing redirect will be updated with the new values.
                    All changes are imported as drafts and must be published to take effect.
                  </div>
                </div>
              </div>

              {/* File Drop Zone */}
              <div
                onDrop={handleDrop}
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onClick={() => fileInputRef.current?.click()}
                className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
                  dragOver
                    ? 'border-brand-purple bg-brand-purple/5'
                    : file
                    ? 'border-green-500 bg-green-50 dark:bg-green-900/10'
                    : 'border-slate-300 dark:border-slate-600 hover:border-brand-purple hover:bg-slate-50 dark:hover:bg-slate-700/50'
                }`}
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".csv,.tsv"
                  onChange={(e) => handleFileSelect(e.target.files?.[0] || null)}
                  className="hidden"
                />
                {file ? (
                  <div className="flex items-center justify-center gap-3">
                    <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <div className="text-left">
                      <p className="font-medium text-slate-900 dark:text-white">{file.name}</p>
                      <p className="text-sm text-slate-500 dark:text-slate-400">
                        {(file.size / 1024).toFixed(1)} KB - Click to change
                      </p>
                    </div>
                  </div>
                ) : (
                  <>
                    <svg className="w-12 h-12 mx-auto text-slate-400 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                    </svg>
                    <p className="text-slate-600 dark:text-slate-400 mb-1">
                      Drag and drop your file here, or <span className="text-brand-purple font-medium">browse</span>
                    </p>
                    <p className="text-sm text-slate-500 dark:text-slate-500">.csv or .tsv files only (max 2MB)</p>
                  </>
                )}
              </div>

              {/* Error Message */}
              {error && (
                <div className="rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-4">
                  <div className="flex gap-3">
                    <svg className="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                    <pre className="text-sm text-red-700 dark:text-red-400 whitespace-pre-wrap font-mono">{error}</pre>
                  </div>
                </div>
              )}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="sticky bottom-0 bg-white dark:bg-slate-800 border-t border-slate-200 dark:border-slate-700 px-6 py-4 flex gap-3 justify-end">
          {importResult ? (
            <button
              onClick={handleClose}
              className="px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity"
            >
              Close
            </button>
          ) : (
            <>
              <button
                onClick={handleClose}
                className="px-4 py-2 text-sm font-medium rounded-lg border border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleImport}
                disabled={!file || loading}
                className="px-4 py-2 text-sm font-medium rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo text-white hover:opacity-90 transition-opacity disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {loading ? (
                  <>
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                    Importing...
                  </>
                ) : (
                  <>
                    <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                    </svg>
                    Import
                  </>
                )}
              </button>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
