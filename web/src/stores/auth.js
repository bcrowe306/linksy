import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('linksy_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('linksy_user') || 'null'))
  const authConfig = ref({ enabled: false, login_url: '', providers: [] })
  const loading = ref(false)
  const error = ref('')

  const isAuthenticated = computed(() => !!token.value && !!user.value)
  const initials = computed(() => {
    if (!user.value || !user.value.name) return 'LK'
    const parts = user.value.name.trim().split(' ')
    if (parts.length >= 2) {
      return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
    }
    return user.value.name.substring(0, 2).toUpperCase()
  })

  async function checkAuthConfig() {
    try {
      const res = await fetch('/api/v1/auth/config')
      if (res.ok) {
        authConfig.value = await res.json()
      }
    } catch (e) {
      console.error('Failed to load auth config:', e)
    }
  }

  function handleAuthRedirect() {
    const params = new URLSearchParams(window.location.search)
    if (params.get('auth_success') === 'true' || (params.get('token') && params.get('email'))) {
      const u = {
        id: params.get('user_id') || 'usr_' + Date.now(),
        email: params.get('email'),
        name: params.get('name') || params.get('email')
      }
      const tok = params.get('token')

      token.value = tok
      user.value = u

      localStorage.setItem('linksy_token', tok)
      localStorage.setItem('linksy_user', JSON.stringify(u))

      // Clean query params from address bar
      const cleanUrl = window.location.pathname + window.location.hash
      window.history.replaceState({}, document.title, cleanUrl)
    }
  }

  async function signup(firstName, lastName, email, password) {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/v1/auth/signup', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          first_name: firstName,
          last_name: lastName,
          email,
          password
        })
      })

      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error || 'Failed to create account')
      }

      token.value = data.token
      user.value = data.user
      localStorage.setItem('linksy_token', data.token)
      localStorage.setItem('linksy_user', JSON.stringify(data.user))
      return data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  async function loginWithPassword(email, password) {
    loading.value = true
    error.value = ''
    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      })

      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error || 'Invalid email or password')
      }

      token.value = data.token
      user.value = data.user
      localStorage.setItem('linksy_token', data.token)
      localStorage.setItem('linksy_user', JSON.stringify(data.user))
      return data
    } catch (err) {
      error.value = err.message
      throw err
    } finally {
      loading.value = false
    }
  }

  function loginWithOAuth(provider, screenHint = '') {
    let url = `/api/v1/auth/oauth/${provider}`
    if (screenHint) {
      url += `?screen_hint=${screenHint}`
    }
    window.location.href = url
  }

  async function logout() {
    try {
      await fetch('/api/v1/auth/logout', { method: 'POST' })
    } catch (e) {
      // ignore
    }
    token.value = ''
    user.value = null
    localStorage.removeItem('linksy_token')
    localStorage.removeItem('linksy_user')
    window.location.href = '/login'
  }

  // Initialize
  checkAuthConfig()
  handleAuthRedirect()

  return {
    token,
    user,
    authConfig,
    loading,
    error,
    isAuthenticated,
    initials,
    signup,
    loginWithPassword,
    loginWithOAuth,
    logout
  }
})
