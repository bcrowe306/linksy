import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api, onUnauthorized, onPaymentRequired } from '../api/client'

// Session state is derived from the server (/api/v1/auth/me); the session
// itself lives in an HttpOnly cookie the browser JavaScript can't read.
export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const team = ref(null)
  const teams = ref([])
  const role = ref(null)
  const permissions = ref([])
  const pendingInvitations = ref(0)
  const billing = ref(null)
  const authConfig = ref({ enabled: false, dev_login: false, providers: [] })
  const loaded = ref(false)
  const loading = ref(false)
  const error = ref('')
  let loadPromise = null

  const isAuthenticated = computed(() => !!user.value)
  const initials = computed(() => {
    const name = user.value?.name || user.value?.email || ''
    const parts = name.trim().split(/\s+/).filter(Boolean)
    if (parts.length >= 2) return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
    return (name.substring(0, 2) || 'LQ').toUpperCase()
  })

  function can(permission) {
    return permissions.value.includes(permission)
  }

  function applyMe(me) {
    if (me?.authenticated) {
      user.value = me.user
      team.value = me.team
      teams.value = me.teams || []
      role.value = me.role
      permissions.value = me.permissions || []
      pendingInvitations.value = me.pending_invitations || 0
      billing.value = me.billing || null
    } else {
      user.value = null
      team.value = null
      teams.value = []
      role.value = null
      permissions.value = []
      pendingInvitations.value = 0
      billing.value = null
      if (me?.error) error.value = me.error
    }
  }

  async function refresh() {
    try {
      applyMe(await api.me())
    } catch {
      applyMe(null)
    } finally {
      loaded.value = true
    }
  }

  // ensureLoaded resolves the session once per page load (used by the router guard).
  function ensureLoaded() {
    if (loaded.value) return Promise.resolve()
    if (!loadPromise) loadPromise = refresh()
    return loadPromise
  }

  async function checkAuthConfig() {
    try {
      authConfig.value = await api.authConfig()
    } catch (e) {
      console.error('Failed to load auth config:', e)
    }
  }

  // Runs an auth call; returns { verificationRequired, pendingToken } when the
  // provider asks for an emailed verification code.
  async function runAuth(fn) {
    loading.value = true
    error.value = ''
    try {
      const data = await fn()
      if (data?.verification_required) {
        return { verificationRequired: true, pendingToken: data.pending_token, email: data.email, message: data.message }
      }
      await refresh()
      return { verificationRequired: false }
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  const loginWithPassword = (email, password) => runAuth(() => api.login(email, password))
  const signup = (payload) => runAuth(() => api.signup(payload))
  const verifyEmail = (pendingToken, code) => runAuth(() => api.verifyEmail(pendingToken, code))
  const devLogin = (payload) => runAuth(() => api.devLogin(payload))

  function loginWithOAuth(provider, { screenHint = '', redirect = '' } = {}) {
    const params = new URLSearchParams()
    if (screenHint) params.set('screen_hint', screenHint)
    if (redirect) params.set('redirect', redirect)
    const qs = params.toString()
    window.location.href = `/api/v1/auth/oauth/${provider}${qs ? `?${qs}` : ''}`
  }

  async function switchTeam(teamId) {
    await api.switchTeam(teamId)
    await refresh()
  }

  async function logout() {
    try {
      await api.logout()
    } catch {
      // session may already be gone
    }
    applyMe(null)
    window.location.href = '/login'
  }

  // A 401 anywhere means the session ended; send the user to sign in.
  onUnauthorized(() => {
    if (user.value) {
      applyMe(null)
      const here = window.location.pathname + window.location.search
      window.location.href = `/login?redirect=${encodeURIComponent(here)}`
    }
  })

  // The plan lapsed mid-session: pick up the read-only state and banner.
  onPaymentRequired(() => refresh())

  checkAuthConfig()

  return {
    user,
    team,
    teams,
    role,
    permissions,
    pendingInvitations,
    billing,
    readOnly: computed(() => !!billing.value?.read_only),
    authConfig,
    loaded,
    loading,
    error,
    isAuthenticated,
    initials,
    can,
    refresh,
    ensureLoaded,
    loginWithPassword,
    signup,
    verifyEmail,
    devLogin,
    loginWithOAuth,
    switchTeam,
    logout
  }
})
