// Theme preference shared by the app and the public pages.
export const THEME_KEY = 'lynqr_theme_pref'

const media = window.matchMedia('(prefers-color-scheme: dark)')

export function readThemePreference() {
  const saved = localStorage.getItem(THEME_KEY)
  return ['system', 'dark', 'light'].includes(saved) ? saved : 'dark' // Dark by default (PRD §1.12)
}

export function prefersDark() {
  return media.matches
}

export function onSystemThemeChange(fn) {
  media.addEventListener('change', fn)
}

// Applies the theme class before Vue mounts to avoid a flash of the wrong theme.
export function applyTheme(pref = readThemePreference()) {
  const dark = pref === 'dark' || (pref === 'system' && media.matches)
  const root = document.documentElement
  root.classList.toggle('dark', dark)
  root.style.colorScheme = dark ? 'dark' : 'light'
  document.querySelector('meta[name="theme-color"]')?.setAttribute('content', dark ? '#0c0c0e' : '#f8f8fa')
  return dark
}
