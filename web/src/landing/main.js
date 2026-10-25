import '../style.css'
import { THEME_KEY, applyTheme, readThemePreference, onSystemThemeChange } from '../lib/theme'
import { MONTHLY_CENTS, ANNUAL_CENTS } from '../lib/pricing'

applyTheme()
onSystemThemeChange(() => readThemePreference() === 'system' && applyTheme('system'))

document.querySelector('[data-theme-toggle]')?.addEventListener('click', () => {
  const next = document.documentElement.classList.contains('dark') ? 'light' : 'dark'
  localStorage.setItem(THEME_KEY, next)
  applyTheme(next)
})

const menuBtn = document.querySelector('[data-menu-toggle]')
const menu = document.getElementById('mobile-menu')
menuBtn?.addEventListener('click', () => {
  const open = menu.classList.toggle('hidden') === false
  menuBtn.setAttribute('aria-expanded', String(open))
})
menu?.querySelectorAll('a').forEach((a) => a.addEventListener('click', () => menu.classList.add('hidden')))

// Pricing interval toggle
const money = (cents) => `$${(cents / 100).toFixed(cents % 100 ? 2 : 0)}`
const priceEl = document.querySelector('[data-price]')
const noteEl = document.querySelector('[data-price-note]')
const buttons = document.querySelectorAll('[data-interval]')
function showInterval(interval) {
  if (!priceEl) return // pages without the pricing card
  buttons.forEach((b) => {
    const on = b.dataset.interval === interval
    b.setAttribute('aria-checked', String(on))
    b.classList.toggle('bg-surface', on)
    b.classList.toggle('text-fg', on)
    b.classList.toggle('shadow-sm', on)
    b.classList.toggle('text-fg-muted', !on)
  })
  if (interval === 'year') {
    priceEl.textContent = money(ANNUAL_CENTS / 12)
    noteEl.textContent = `${money(ANNUAL_CENTS)} billed once a year`
  } else {
    priceEl.textContent = money(MONTHLY_CENTS)
    noteEl.textContent = 'Billed monthly · cancel any time'
  }
}
buttons.forEach((b) => b.addEventListener('click', () => showInterval(b.dataset.interval)))
showInterval('year')

// Sign-up CTAs carry a checkout code from ?code= through to Billing.
const code = new URLSearchParams(location.search).get('code')
if (code) {
  document.querySelectorAll('a[href="/signup"]').forEach((a) => {
    a.href = `/signup?redirect=${encodeURIComponent(`/settings?tab=billing&code=${code}`)}`
  })
}

document.querySelectorAll('[data-year]').forEach((el) => (el.textContent = String(new Date().getFullYear())))
