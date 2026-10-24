import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const currentTheme = ref(localStorage.getItem('linksy_theme') || 'dark')

  function setTheme(theme) {
    currentTheme.value = theme
    localStorage.setItem('linksy_theme', theme)
    if (theme === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function toggleTheme() {
    setTheme(currentTheme.value === 'dark' ? 'light' : 'dark')
  }

  // Initialize
  setTheme(currentTheme.value)

  return {
    currentTheme,
    setTheme,
    toggleTheme
  }
})
