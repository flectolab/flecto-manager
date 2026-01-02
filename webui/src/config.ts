declare global {
  interface Window {
    APP_CONFIG?: {
      apiUrl?: string
      authHeaderName?: string
    }
  }
}

export const config = {
  apiUrl: window.APP_CONFIG?.apiUrl ?? '',
  authHeaderName: window.APP_CONFIG?.authHeaderName ?? 'Authorization',
}
