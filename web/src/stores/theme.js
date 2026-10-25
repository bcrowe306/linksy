import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const STORAGE_KEY = 'lynqr_theme_pref'
const media = window.matchMedia('(prefers-color-scheme: dark)')

function readPreference() {
  const saved = localStorage.getItem(STORAGE_KEY)
  return ['system', 'dark', 'light'].includes(saved) ? saved : 'dark' // Dark by default (PRD §1.12)
}

// Applies the theme class before Vue mounts to avoid a flash of the wrong theme.
export function applyTheme(pref = readPreference()) {
  const dark = pref === 'dark' || (pref === 'system' && media.matches)
  const root = document.documentElement
  root.classList.toggle('dark', dark)
  root.style.colorScheme = dark ? 'dark' : 'light'
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', dark ? '#0c0c0e' : '#f8f8fa')
  return dark
}

export const useThemeStore = defineStore('theme', () => {
  // 'system' | 'dark' | 'light'
  const preference = ref(readPreference())
  const systemPrefersDark = ref(media.matches)

  const effectiveTheme = computed(() => {
    if (preference.value === 'system') return systemPrefersDark.value ? 'dark' : 'light'
    return preference.value
  })

  function setTheme(pref) {
    preference.value = pref
    localStorage.setItem(STORAGE_KEY, pref)
    applyTheme(pref)
  }

  function cycleTheme() {
    const order = ['dark', 'light', 'system']
    setTheme(order[(order.indexOf(preference.value) + 1) % order.length])
  }

  media.addEventListener('change', (e) => {
    systemPrefersDark.value = e.matches
    if (preference.value === 'system') applyTheme('system')
  })

  applyTheme(preference.value)

  return { preference, effectiveTheme, setTheme, cycleTheme }
})
