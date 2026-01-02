export function Dashboard() {
  const stats = [
    { name: 'Total Routes', value: '12', change: '+2', changeType: 'increase' },
    { name: 'Active Redirects', value: '48', change: '+5', changeType: 'increase' },
    { name: 'Requests Today', value: '2,340', change: '+12%', changeType: 'increase' },
    { name: 'Avg Response Time', value: '45ms', change: '-3ms', changeType: 'decrease' },
  ]

  return (
    <div>
      <div className="mb-8">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">Overview</h2>
        <p className="mt-1 text-slate-500 dark:text-slate-400">Monitor your traffic routing and redirects</p>
      </div>

      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => (
          <div
            key={stat.name}
            className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700"
          >
            <p className="text-sm font-medium text-slate-500 dark:text-slate-400">{stat.name}</p>
            <div className="mt-2 flex items-baseline gap-2">
              <p className="text-3xl font-semibold text-slate-900 dark:text-white">{stat.value}</p>
              <span
                className={`text-sm font-medium ${
                  stat.changeType === 'increase'
                    ? 'text-emerald-600 dark:text-emerald-400'
                    : 'text-red-600 dark:text-red-400'
                }`}
              >
                {stat.change}
              </span>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-8 grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Recent Activity</h3>
          <div className="mt-4 space-y-4">
            {[
              { action: 'Route created', target: '/api/v2/users', time: '2 minutes ago' },
              { action: 'Redirect updated', target: '/old-page → /new-page', time: '15 minutes ago' },
              { action: 'Route deleted', target: '/deprecated/endpoint', time: '1 hour ago' },
              { action: 'Config reloaded', target: 'System', time: '3 hours ago' },
            ].map((activity, index) => (
              <div key={index} className="flex items-center justify-between py-2 border-b border-slate-100 dark:border-slate-700 last:border-0">
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-white">{activity.action}</p>
                  <p className="text-sm text-slate-500 dark:text-slate-400">{activity.target}</p>
                </div>
                <span className="text-xs text-slate-400">{activity.time}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="rounded-xl bg-white dark:bg-slate-800 p-6 shadow-sm border border-slate-200 dark:border-slate-700">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">Quick Actions</h3>
          <div className="mt-4 grid grid-cols-2 gap-4">
            <button className="flex items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-brand-purple to-brand-indigo px-4 py-3 text-sm font-medium text-white hover:opacity-90 transition-opacity">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
              Add Route
            </button>
            <button className="flex items-center justify-center gap-2 rounded-lg bg-gradient-to-r from-brand-cyan to-brand-sky px-4 py-3 text-sm font-medium text-white hover:opacity-90 transition-opacity">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
              Add Redirect
            </button>
            <button className="flex items-center justify-center gap-2 rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-3 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Reload Config
            </button>
            <button className="flex items-center justify-center gap-2 rounded-lg border border-slate-300 dark:border-slate-600 px-4 py-3 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors">
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              View Logs
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
