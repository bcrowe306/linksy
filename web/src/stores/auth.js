import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('linksy_token') || '')
  const user = ref(JSON.parse(localStorage.getItem('linksy_user') || 'null'))
  const authConfig = ref({ enabled: false, login_url: '' })

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
    if (params.get('auth_success') === 'true') {
      const u = {
        id: params.get('user_id'),
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

  function login() {
    if (authConfig.value.login_url) {
      window.location.href = authConfig.value.login_url
    } else {
      window.location.href = '/api/v1/auth/login'
    }
  }

  function logout() {
    token.value = ''
    user.value = null
    localStorage.removeItem('linksy_token')
    localStorage.removeItem('linksy_user')
    window.location.href = '/'
  }

  // Initialize
  checkAuthConfig()
  handleAuthRedirect()

  return {
    token,
    user,
    authConfig,
    isAuthenticated,
    initials,
    login,
    logout
  }
})
