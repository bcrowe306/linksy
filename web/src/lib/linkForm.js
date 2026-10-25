// Helpers shared by the create and edit link forms.

const UTM_KEYS = ['utm_source', 'utm_medium', 'utm_campaign', 'utm_term', 'utm_content']

// datetime-local wants "YYYY-MM-DDTHH:mm" in local time.
export function toLocalInput(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function blankLinkForm() {
  return { target_url: '', custom_slug: '', title: '', password: undefined, expires_local: '', fallback_url: '', ...Object.fromEntries(UTM_KEYS.map((k) => [k, ''])) }
}

export function linkToForm(link) {
  return {
    target_url: link.target_url,
    slug: link.slug,
    title: link.title,
    description: link.description,
    is_active: link.is_active,
    password: undefined,
    expires_local: toLocalInput(link.expires_at),
    fallback_url: link.fallback_url || '',
    ...Object.fromEntries(UTM_KEYS.map((k) => [k, link[k] || '']))
  }
}

// formToPayload converts the form into an API body. password is only sent
// when changed (undefined = keep, '' = remove).
export function formToPayload(form) {
  const { expires_local, password, ...rest } = form
  const body = { ...rest, expires_at: expires_local ? new Date(expires_local).toISOString() : null }
  if (password !== undefined && password !== 'change') body.password = password
  return body
}
