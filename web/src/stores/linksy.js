import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useLynqrStore = defineStore('lynqr', () => {
  const links = ref([])
  const qrCodes = ref([])
  const documents = ref([])
  const analyticsSummary = ref(null)
  const loading = ref(false)
  const currentTeam = ref({
    id: 'team_default',
    name: 'Main Workspace',
    role: 'owner'
  })

  async function fetchLinks() {
    try {
      loading.value = true
      const res = await api.getLinks()
      links.value = res.data || []
    } catch (err) {
      console.error(err)
    } finally {
      loading.value = false
    }
  }

  async function fetchQRCodes() {
    try {
      loading.value = true
      const res = await api.getQRCodes()
      qrCodes.value = res.data || []
    } catch (err) {
      console.error(err)
    } finally {
      loading.value = false
    }
  }

  async function fetchDocuments() {
    try {
      loading.value = true
      const res = await api.getDocuments()
      documents.value = res.data || []
    } catch (err) {
      console.error(err)
    } finally {
      loading.value = false
    }
  }

  async function fetchAnalytics(linkId = null, qrId = null) {
    try {
      const res = await api.getAnalyticsSummary(linkId, qrId)
      analyticsSummary.value = res
    } catch (err) {
      console.error(err)
    }
  }

  async function refreshAll() {
    await Promise.all([fetchLinks(), fetchQRCodes(), fetchDocuments(), fetchAnalytics()])
  }

  return {
    links,
    qrCodes,
    documents,
    analyticsSummary,
    loading,
    currentTeam,
    fetchLinks,
    fetchQRCodes,
    fetchDocuments,
    fetchAnalytics,
    refreshAll
  }
})

export const useLinqrStore = useLynqrStore
export const useLinksyStore = useLynqrStore
