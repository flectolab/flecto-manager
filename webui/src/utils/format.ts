/**
 * Format a byte size into a human-readable string
 * @param bytes - The size in bytes
 * @returns Formatted string like "1.5 KB", "2.3 MB", etc.
 */
export function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const size = bytes / Math.pow(k, i)

  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}