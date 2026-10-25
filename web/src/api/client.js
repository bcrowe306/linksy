const BASE_URL = ''

function getHeaders(custom = {}) {
  const token = localStorage.getItem('lynqr_token') || localStorage.getItem('linqr_token') || localStorage.getItem('linksy_token')
  const headers = { ...custom }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  return headers
}

export const api = {
  // Links
  async getLinks(limit = 50, offset = 0) {
    const res = await fetch(`${BASE_URL}/api/v1/links?limit=${limit}&offset=${offset}`, {
      headers: getHeaders()
    })
    return res.json()
  },
  async createLink(data) {
    const res = await fetch(`${BASE_URL}/api/v1/links`, {
      method: 'POST',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(data)
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to create link')
    }
    return res.json()
  },
  async updateLink(id, data) {
    const res = await fetch(`${BASE_URL}/api/v1/links/${id}`, {
      method: 'PUT',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(data)
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to update link')
    }
    return res.json()
  },
  async deleteLink(id) {
    const res = await fetch(`${BASE_URL}/api/v1/links/${id}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (!res.ok) throw new Error('Failed to delete link')
    return true
  },

  // QR Codes
  async getQRCodes(limit = 50, offset = 0) {
    const res = await fetch(`${BASE_URL}/api/v1/qr?limit=${limit}&offset=${offset}`, {
      headers: getHeaders()
    })
    return res.json()
  },
  async createQRCode(data) {
    const res = await fetch(`${BASE_URL}/api/v1/qr`, {
      method: 'POST',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(data)
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to create QR code')
    }
    return res.json()
  },
  async updateQRCode(id, data) {
    const res = await fetch(`${BASE_URL}/api/v1/qr/${id}`, {
      method: 'PUT',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(data)
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to update QR code')
    }
    return res.json()
  },
  async deleteQRCode(id) {
    const res = await fetch(`${BASE_URL}/api/v1/qr/${id}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (!res.ok) throw new Error('Failed to delete QR code')
    return true
  },
  getExportUrl(id, format = 'png', size = 512, download = true) {
    return `${BASE_URL}/api/v1/qr/${id}/export?format=${format}&size=${size}&download=${download}`
  },

  // Documents
  async uploadDocument(file) {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch(`${BASE_URL}/api/v1/documents/upload`, {
      method: 'POST',
      headers: getHeaders(),
      body: formData
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to upload document')
    }
    return res.json()
  },
  async updateDocument(id, file) {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch(`${BASE_URL}/api/v1/documents/${id}`, {
      method: 'PUT',
      headers: getHeaders(),
      body: formData
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to update document')
    }
    return res.json()
  },
  async getDocuments() {
    const res = await fetch(`${BASE_URL}/api/v1/documents`, {
      headers: getHeaders()
    })
    return res.json()
  },
  async deleteDocument(id, deleteQR = false) {
    const res = await fetch(`${BASE_URL}/api/v1/documents/${id}?delete_qr=${deleteQR}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to delete document')
    }
    return res.json()
  },
  async bulkDeleteDocuments(documentIds, deleteQR = false) {
    const res = await fetch(`${BASE_URL}/api/v1/documents/bulk-delete`, {
      method: 'POST',
      headers: {
        ...getHeaders(),
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ document_ids: documentIds, delete_qr: deleteQR })
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to delete documents')
    }
    return res.json()
  },

  // Team & User Management
  async getTeamMembers() {
    const res = await fetch(`${BASE_URL}/api/v1/team/members`, {
      headers: getHeaders()
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to fetch team members')
    }
    return res.json()
  },
  async inviteTeamMember(email, role = 'member') {
    const res = await fetch(`${BASE_URL}/api/v1/team/members/invite`, {
      method: 'POST',
      headers: {
        ...getHeaders(),
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ email, role })
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to invite team member')
    }
    return res.json()
  },
  async updateMemberRole(userId, role) {
    const res = await fetch(`${BASE_URL}/api/v1/team/members/${userId}/role`, {
      method: 'PUT',
      headers: {
        ...getHeaders(),
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ role })
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to update member role')
    }
    return res.json()
  },
  async removeTeamMember(userId) {
    const res = await fetch(`${BASE_URL}/api/v1/team/members/${userId}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to remove member')
    }
    return res.json()
  },
  async resetMemberCredentials(userId) {
    const res = await fetch(`${BASE_URL}/api/v1/team/members/${userId}/reset-credentials`, {
      method: 'POST',
      headers: getHeaders()
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to reset credentials')
    }
    return res.json()
  },
  async revokeInvitation(invitationId) {
    const res = await fetch(`${BASE_URL}/api/v1/team/invitations/${invitationId}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (!res.ok) {
      const err = await res.json()
      throw new Error(err.error || 'Failed to revoke invitation')
    }
    return res.json()
  },

  // Analytics
  async getAnalyticsSummary(linkId = null, qrId = null) {
    let url = `${BASE_URL}/api/v1/analytics/summary`
    const params = new URLSearchParams()
    if (linkId) params.append('link_id', linkId)
    if (qrId) params.append('qr_id', qrId)
    if (params.toString()) url += `?${params.toString()}`

    const res = await fetch(url, {
      headers: getHeaders()
    })
    return res.json()
  }
}
