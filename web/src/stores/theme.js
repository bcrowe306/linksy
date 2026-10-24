import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  // 'system' | 'dark' | 'light'
  const preference = ref(localStorage.getItem('linksy_theme_pref') || 'system')
  const systemPrefersDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)

  const effectiveTheme = computed(() => {
    if (preference.value === 'system') {
      return systemPrefersDark.value ? 'dark' : 'light'
    }
    return preference.value
  })

  function applyTheme() {
    const isDark = effectiveTheme.value === 'dark'
    if (isDark) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function setTheme(pref) {
    preference.value = pref
    localStorage.setItem('linksy_theme_pref', pref)
    applyTheme()
  }

  function cycleTheme() {
    if (preference.value === 'system') {
      setTheme('dark')
    } else if (preference.value === 'dark') {
      setTheme('light')
    } else {
      setTheme('system')
    }
  }

  // Listen to OS system preference changes in real-time
  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', (e) => {
    systemPrefersDark.value = e.matches
    if (preference.value === 'system') {
      applyTheme()
    }
  })

  // Initialize on load
  applyTheme()

  return {
    preference,
    effectiveTheme,
    setTheme,
    cycleTheme
  }
})
