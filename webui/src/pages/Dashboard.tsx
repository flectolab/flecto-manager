import { useQuery } from '@apollo/client/react'
import { Link } from 'react-router-dom'
import { useDocumentTitle } from '../hooks/useDocumentTitle'
import { useCurrentProject } from '../hooks/useCurrentProject'
import { GetProjectDashboardDocument } from '../generated/graphql'
import { RelativeTime } from '../components/RelativeTime'

export function Dashboard() {
  const { namespaceCode, projectCode, namespace, project } = useCurrentProject()
  useDocumentTitle(namespaceCode && projectCode ? `Dashboard - ${namespaceCode}/${projectCode}` : 'Dashboard')

  const { data, loading, error } = useQuery(GetProjectDashboardDocument, {
    variables: { namespaceCode: namespaceCode ?? '', projectCode: projectCode ?? '' },
    skip: !namespaceCode || !projectCode,
  })

  const dashboard = data?.projectDashboard

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-brand-purple border-t-transparent"></div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="rounded-xl bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-6">
        <p className="text-red-700 dark:text-red-400">Error loading dashboard: {error.message}</p>
      </div>
    )
  }

  const isPublished = dashboard?.publishedAt && !dashboard.publishedAt.startsWith('0001-')

  return (
    <div>
      {/* Header */}
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Dashboard</h2>
        <p className="mt-1 text-slate-500 dark:text-slate-400">
          Overview of {namespace?.name} / {project?.name}
        </p>
      </div>

      {/* Project Info Card */}
      <div className="mb-6 rounded-xl bg-gradient-to-r from-brand-purple to-brand-indigo p-6 text-white">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold opacity-90">Project Version</h3>
            <p className="text-4xl font-bold mt-1">v{dashboard?.version ?? 0}</p>
          </div>
          {isPublished && (
            <div className="text-right">
              <p className="text-sm opacity-75">Last Published</p>
              <p className="text-lg font-medium mt-1">
                <RelativeTime date={dashboard!.publishedAt} className="text-white" />
              </p>
            </div>
          )}
          {!isPublished && (
            <div className="text-right">
              <p className="text-sm opacity-75">Status</p>
              <p className="text-lg font-medium mt-1">Not published yet</p>
            </div>
          )}
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-6">
        {/* Redirects Card */}
        <Link
          to={`/${namespaceCode}/${projectCode}/redirects`}
          className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700 hover:border-brand-purple dark:hover:border-brand-purple transition-colors"
        >
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-cyan-100 dark:bg-cyan-900/30">
              <svg className="w-5 h-5 text-cyan-600 dark:text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Redirects</h3>
          </div>
          <p className="text-3xl font-bold text-slate-900 dark:text-white mb-3">
            {dashboard?.redirectStats.total ?? 0}
          </p>
          <div className="space-y-1.5 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Basic</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{dashboard?.redirectStats.countBasic ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Host</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{dashboard?.redirectStats.countBasicHost ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Regex</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{dashboard?.redirectStats.countRegex ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Regex Host</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{dashboard?.redirectStats.countRegexHost ?? 0}</span>
            </div>
          </div>
        </Link>

        {/* Redirect Drafts Card */}
        <Link
          to={`/${namespaceCode}/${projectCode}/redirects`}
          className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700 hover:border-brand-purple dark:hover:border-brand-purple transition-colors"
        >
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-amber-100 dark:bg-amber-900/30">
              <svg className="w-5 h-5 text-amber-600 dark:text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Redirect Drafts</h3>
          </div>
          <p className="text-3xl font-bold text-slate-900 dark:text-white mb-3">
            {dashboard?.redirectDraftStats.total ?? 0}
          </p>
          <div className="space-y-1.5 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">New</span>
              <span className="font-medium text-green-600 dark:text-green-400">{dashboard?.redirectDraftStats.countCreate ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Modified</span>
              <span className="font-medium text-amber-600 dark:text-amber-400">{dashboard?.redirectDraftStats.countUpdate ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Deleted</span>
              <span className="font-medium text-red-600 dark:text-red-400">{dashboard?.redirectDraftStats.countDelete ?? 0}</span>
            </div>
          </div>
        </Link>

        {/* Pages Card */}
        <Link
          to={`/${namespaceCode}/${projectCode}/pages`}
          className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700 hover:border-brand-purple dark:hover:border-brand-purple transition-colors"
        >
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-purple-100 dark:bg-purple-900/30">
              <svg className="w-5 h-5 text-purple-600 dark:text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Pages</h3>
          </div>
          <p className="text-3xl font-bold text-slate-900 dark:text-white mb-3">
            {dashboard?.pageStats.total ?? 0}
          </p>
          <div className="space-y-1.5 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Basic</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{dashboard?.pageStats.countBasic ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Host</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{dashboard?.pageStats.countBasicHost ?? 0}</span>
            </div>
          </div>
        </Link>

        {/* Page Drafts Card */}
        <Link
          to={`/${namespaceCode}/${projectCode}/pages`}
          className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700 hover:border-brand-purple dark:hover:border-brand-purple transition-colors"
        >
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-amber-100 dark:bg-amber-900/30">
              <svg className="w-5 h-5 text-amber-600 dark:text-amber-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Page Drafts</h3>
          </div>
          <p className="text-3xl font-bold text-slate-900 dark:text-white mb-3">
            {dashboard?.pageDraftStats.total ?? 0}
          </p>
          <div className="space-y-1.5 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">New</span>
              <span className="font-medium text-green-600 dark:text-green-400">{dashboard?.pageDraftStats.countCreate ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Modified</span>
              <span className="font-medium text-amber-600 dark:text-amber-400">{dashboard?.pageDraftStats.countUpdate ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Deleted</span>
              <span className="font-medium text-red-600 dark:text-red-400">{dashboard?.pageDraftStats.countDelete ?? 0}</span>
            </div>
          </div>
        </Link>

        {/* Agents Card */}
        <Link
          to={`/${namespaceCode}/${projectCode}/agents`}
          className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700 hover:border-brand-purple dark:hover:border-brand-purple transition-colors"
        >
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
              <svg className="w-5 h-5 text-emerald-600 dark:text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Agents</h3>
          </div>
          <p className="text-3xl font-bold text-slate-900 dark:text-white mb-3">
            {dashboard?.agentStats.totalOnline ?? 0}
          </p>
          <div className="space-y-1.5 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Online</span>
              <span className="font-medium text-emerald-600 dark:text-emerald-400">{dashboard?.agentStats.totalOnline ?? 0}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">With Errors</span>
              <span className={`font-medium ${(dashboard?.agentStats.countError ?? 0) > 0 ? 'text-red-600 dark:text-red-400' : 'text-slate-700 dark:text-slate-300'}`}>
                {dashboard?.agentStats.countError ?? 0}
              </span>
            </div>
          </div>
        </Link>
      </div>
    </div>
  )
}
