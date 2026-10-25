import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { THEME_KEY, readThemePreference, applyTheme, prefersDark, onSystemThemeChange } from '../lib/theme'

export { applyTheme }

export const useThemeStore = defineStore('theme', () => {
  // 'system' | 'dark' | 'light'
  const preference = ref(readThemePreference())
  const systemPrefersDark = ref(prefersDark())

  const effectiveTheme = computed(() => {
    if (preference.value === 'system') return systemPrefersDark.value ? 'dark' : 'light'
    return preference.value
  })

  function setTheme(pref) {
    preference.value = pref
    localStorage.setItem(THEME_KEY, pref)
    applyTheme(pref)
  }

  function cycleTheme() {
    const order = ['dark', 'light', 'system']
    setTheme(order[(order.indexOf(preference.value) + 1) % order.length])
  }

  onSystemThemeChange((e) => {
    systemPrefersDark.value = e.matches
    if (preference.value === 'system') applyTheme('system')
  })

  applyTheme(preference.value)

  return { preference, effectiveTheme, setTheme, cycleTheme }
})
