import { useEffect } from 'react'

const APP_NAME = 'Flecto Manager'

/**
 * Hook to set the document title
 * @param title - The page title (will be suffixed with app name)
 * @param deps - Optional dependencies array to trigger title update
 */
export function useDocumentTitle(title: string | null | undefined, deps: unknown[] = []) {
  useEffect(() => {
    const previousTitle = document.title

    if (title) {
      document.title = `${title} - ${APP_NAME}`
    } else {
      document.title = APP_NAME
    }

    return () => {
      document.title = previousTitle
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [title, ...deps])
}
