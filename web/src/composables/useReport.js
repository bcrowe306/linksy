import { ref, watch } from 'vue'
import { api } from '../api/client'
import { useAuthStore } from '../stores/auth'

// useReport fetches an analytics report whenever params change. The previous
// report stays on screen (dimmed via `loading`) while the next one loads.
export function useReport(paramsRef) {
  const report = ref(null)
  const loading = ref(false)
  const error = ref('')
  const auth = useAuthStore()
  let seq = 0

  async function load() {
    const id = ++seq
    loading.value = true
    try {
      const r = await api.getReport(paramsRef.value)
      if (id === seq) {
        report.value = r
        error.value = ''
      }
    } catch (err) {
      if (id === seq) error.value = err.message
    } finally {
      if (id === seq) loading.value = false
    }
  }

  watch([paramsRef, () => auth.team?.id], load, { deep: true, immediate: true })
  return { report, loading, error, reload: load }
}
