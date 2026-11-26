import type { RedirectOptionsInput, RedirectType } from '../../generated/graphql'

// A null option is not set and behaves like false, as on the server.
export interface RedirectOptionsValue {
  ignoreQueryString?: boolean | null
  preserveQueryString?: boolean | null
}

export const redirectOptionKeys = ['ignoreQueryString', 'preserveQueryString'] as const

export type RedirectOptionKey = (typeof redirectOptionKeys)[number]

export const redirectOptionLabels: Record<RedirectOptionKey, string> = {
  ignoreQueryString: 'Ignore query string',
  preserveQueryString: 'Preserve query string',
}

export const redirectOptionDescriptions: Record<RedirectOptionKey, string> = {
  ignoreQueryString:
    'Match the source whatever the query string of the request is (/old?utm_source=x matches /old). An exact rule with that query string still wins.',
  preserveQueryString:
    'Forward the query string of the request to the target (/old?utm_source=x redirects to /new?utm_source=x).',
}

export function supportsQueryStringOptions(type: RedirectType): boolean {
  return type === 'BASIC' || type === 'BASIC_HOST'
}

// toOptionsInput keeps only the input fields: a value read from the Apollo cache
// carries __typename, which the input type refuses.
export function toOptionsInput(options?: RedirectOptionsValue | null): RedirectOptionsInput {
  return {
    ignoreQueryString: options?.ignoreQueryString ?? null,
    preserveQueryString: options?.preserveQueryString ?? null,
  }
}

// optionsEqual tells a null option apart from a false one, like the server does
// when it compares drafts.
export function optionsEqual(a?: RedirectOptionsValue | null, b?: RedirectOptionsValue | null): boolean {
  return redirectOptionKeys.every((key) => (a?.[key] ?? null) === (b?.[key] ?? null))
}

export function enabledOptions(options?: RedirectOptionsValue | null): RedirectOptionKey[] {
  return redirectOptionKeys.filter((key) => options?.[key] === true)
}

// optionValue is what a checkbox stores: true when checked, otherwise the false
// read from the server if there was one, so unchecking an option that was not
// set does not turn into a change.
export function optionValue(checked: boolean, initial?: boolean | null): boolean | null {
  if (checked) return true
  return initial === false ? false : null
}
