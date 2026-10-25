// Lynqr API client. Authentication uses the HttpOnly session cookie set by the
// server, so no credentials are ever stored in localStorage.

export class ApiError extends Error {
  constructor(message, status, body) {
    super(message)
    this.status = status
    this.body = body
  }
}

let unauthorizedHandler = null
export function onUnauthorized(fn) {
  unauthorizedHandler = fn
}

export async function request(method, path, { body, form, query, raw } = {}) {
  let url = path
  if (query) {
    const params = new URLSearchParams()
    Object.entries(query).forEach(([k, v]) => {
      if (v !== undefined && v !== null && v !== '') params.append(k, v)
    })
    const qs = params.toString()
    if (qs) url += `?${qs}`
  }

  const init = { method, credentials: 'same-origin', headers: { Accept: 'application/json' } }
  if (form) {
    init.body = form
  } else if (body !== undefined) {
    init.headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  const res = await fetch(url, init)
  if (raw) return res

  let data = null
  const text = await res.text()
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = { message: text }
    }
  }

  if (!res.ok) {
    if (res.status === 401 && unauthorizedHandler) unauthorizedHandler()
    throw new ApiError(data?.error || `Request failed (${res.status})`, res.status, data)
  }
  return data
}

const get = (path, query) => request('GET', path, { query })
const post = (path, body) => request('POST', path, { body })
const put = (path, body) => request('PUT', path, { body })
const del = (path, body) => request('DELETE', path, { body })

export const api = {
  // Auth & session
  authConfig: () => get('/api/v1/auth/config'),
  me: () => get('/api/v1/auth/me'),
  login: (email, password) => post('/api/v1/auth/login', { email, password }),
  signup: (payload) => post('/api/v1/auth/signup', payload),
  verifyEmail: (pendingToken, code) => post('/api/v1/auth/verify-email', { pending_token: pendingToken, code }),
  devLogin: (payload) => post('/api/v1/auth/dev-login', payload),
  logout: () => post('/api/v1/auth/logout'),
  requestPasswordReset: (email) => post('/api/v1/auth/password-reset', { email }),
  confirmPasswordReset: (token, newPassword) => post('/api/v1/auth/password-reset/confirm', { token, new_password: newPassword }),

  // Profile, sessions & API keys
  getProfile: () => get('/api/v1/user/profile'),
  updateProfile: (data) => put('/api/v1/user/profile', data),
  changePassword: (currentPassword, newPassword) =>
    post('/api/v1/user/change-password', { current_password: currentPassword, new_password: newPassword }),
  getSessions: () => get('/api/v1/user/sessions'),
  revokeSession: (id) => del(`/api/v1/user/sessions/${id}`),
  revokeOtherSessions: () => post('/api/v1/user/sessions/revoke-others'),
  getApiKeys: () => get('/api/v1/api-keys'),
  createApiKey: (name, expiresInDays = 0) => post('/api/v1/api-keys', { name, expires_in_days: expiresInDays }),
  revokeApiKey: (id) => del(`/api/v1/api-keys/${id}`),

  // Workspaces
  getTeams: () => get('/api/v1/teams'),
  createTeam: (name) => post('/api/v1/teams', { name }),
  switchTeam: (teamId) => post('/api/v1/teams/switch', { team_id: teamId }),
  getTeam: () => get('/api/v1/team'),
  renameTeam: (name) => put('/api/v1/team', { name }),
  deleteTeam: (confirmName) => del('/api/v1/team', { confirm_name: confirmName }),
  transferOwnership: (userId) => post('/api/v1/team/transfer', { user_id: userId }),
  leaveTeam: () => post('/api/v1/team/leave'),
  getAuditLog: (limit = 50, offset = 0) => get('/api/v1/team/audit', { limit, offset }),

  // Members & invitations
  getTeamMembers: () => get('/api/v1/team/members'),
  inviteTeamMember: (email, role = 'member') => post('/api/v1/team/members/invite', { email, role }),
  updateMemberRole: (userId, role) => put(`/api/v1/team/members/${userId}/role`, { role }),
  suspendMember: (userId) => post(`/api/v1/team/members/${userId}/suspend`),
  reactivateMember: (userId) => post(`/api/v1/team/members/${userId}/reactivate`),
  removeTeamMember: (userId) => del(`/api/v1/team/members/${userId}`),
  resetMemberCredentials: (userId) => post(`/api/v1/team/members/${userId}/reset-credentials`),
  revokeInvitation: (id) => del(`/api/v1/team/invitations/${id}`),
  previewInvitation: (token) => get(`/api/v1/invitations/${encodeURIComponent(token)}`),
  acceptInvitation: (token) => post(`/api/v1/invitations/${encodeURIComponent(token)}/accept`),
  getMyInvitations: () => get('/api/v1/user/invitations'),
  respondToInvitation: (id, accept) => post(`/api/v1/user/invitations/${id}/${accept ? 'accept' : 'decline'}`),

  // Links
  getLinks: (limit = 100, offset = 0) => get('/api/v1/links', { limit, offset }),
  getLink: (id) => get(`/api/v1/links/${id}`),
  createLink: (data) => post('/api/v1/links', data),
  updateLink: (id, data) => put(`/api/v1/links/${id}`, data),
  deleteLink: (id) => del(`/api/v1/links/${id}`),

  // QR Codes
  getQRCodes: (limit = 100, offset = 0) => get('/api/v1/qr', { limit, offset }),
  getQRCode: (id) => get(`/api/v1/qr/${id}`),
  createQRCode: (data) => post('/api/v1/qr', data),
  updateQRCode: (id, data) => put(`/api/v1/qr/${id}`, data),
  deleteQRCode: (id) => del(`/api/v1/qr/${id}`),
  getExportUrl(id, format = 'png', size = 512, download = true) {
    return `/api/v1/qr/${id}/export?format=${format}&size=${size}&download=${download}`
  },

  // Documents
  uploadDocument(file) {
    const form = new FormData()
    form.append('file', file)
    return request('POST', '/api/v1/documents/upload', { form })
  },
  updateDocument(id, file) {
    const form = new FormData()
    form.append('file', file)
    return request('PUT', `/api/v1/documents/${id}`, { form })
  },
  getDocuments: () => get('/api/v1/documents'),
  deleteDocument: (id, deleteQR = false) => request('DELETE', `/api/v1/documents/${id}`, { query: { delete_qr: deleteQR } }),
  bulkDeleteDocuments: (documentIds, deleteQR = false) =>
    post('/api/v1/documents/bulk-delete', { document_ids: documentIds, delete_qr: deleteQR }),

  // Analytics
  getAnalyticsSummary: (params = {}) =>
    get('/api/v1/analytics/summary', {
      link_id: params.linkId,
      qr_id: params.qrId,
      from: params.from,
      to: params.to
    })
}
