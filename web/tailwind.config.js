/** @type {import('tailwindcss').Config} */

// Neutral and semantic colors resolve to CSS variables (see src/style.css) so every
// utility automatically follows the active light / dark theme.
const v = (name) => `rgb(var(--${name}) / <alpha-value>)`
const scale = (prefix, steps) => Object.fromEntries(steps.map((s) => [s, v(`${prefix}-${s}`)]))

const neutralSteps = [50, 100, 200, 300, 400, 500, 600, 700, 750, 800, 850, 900, 950]
const brandSteps = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950]
const brand = scale('brand', brandSteps)

export default {
  darkMode: 'class',
  content: ['./index.html', './home.html', '../internal/web/templates/*.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        gray: scale('gray', neutralSteps),
        brand,
        lynqr: brand,
        // Semantic tokens for new UI
        canvas: v('canvas'),
        surface: v('surface'),
        elevated: v('elevated'),
        subtle: v('subtle'),
        line: v('line'),
        'line-strong': v('line-strong'),
        fg: v('fg'),
        'fg-muted': v('fg-muted'),
        'fg-subtle': v('fg-subtle')
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', '-apple-system', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace']
      },
      boxShadow: {
        card: '0 1px 2px rgb(0 0 0 / 0.04), 0 1px 3px rgb(0 0 0 / 0.06)',
        pop: '0 10px 38px -10px rgb(0 0 0 / 0.35), 0 10px 20px -15px rgb(0 0 0 / 0.2)',
        glow: '0 0 0 1px rgb(var(--brand-500) / 0.25), 0 8px 24px -6px rgb(var(--brand-500) / 0.35)'
      },
      keyframes: {
        'fade-in': { from: { opacity: 0 }, to: { opacity: 1 } },
        'scale-in': { from: { opacity: 0, transform: 'scale(.96)' }, to: { opacity: 1, transform: 'scale(1)' } },
        'slide-up': { from: { opacity: 0, transform: 'translateY(8px)' }, to: { opacity: 1, transform: 'translateY(0)' } },
        'slide-in-right': { from: { transform: 'translateX(100%)' }, to: { transform: 'translateX(0)' } },
        shimmer: { '100%': { transform: 'translateX(100%)' } }
      },
      animation: {
        'fade-in': 'fade-in .15s ease-out',
        'scale-in': 'scale-in .16s cubic-bezier(.16,1,.3,1)',
        'slide-up': 'slide-up .2s cubic-bezier(.16,1,.3,1)',
        'slide-in-right': 'slide-in-right .25s cubic-bezier(.16,1,.3,1)',
        shimmer: 'shimmer 1.4s infinite'
      }
    }
  },
  plugins: []
}
