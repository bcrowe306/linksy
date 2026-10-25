import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api/client'

export const useLynqrStore = defineStore('lynqr', () => {
  const links = ref([])
  const qrCodes = ref([])
  const documents = ref([])
  const loading = ref(false)

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

  function reset() {
    links.value = []
    qrCodes.value = []
    documents.value = []
  }

  async function refreshAll() {
    await Promise.all([fetchLinks(), fetchQRCodes(), fetchDocuments()])
  }

  return {
    links,
    qrCodes,
    documents,
    loading,
    reset,
    fetchLinks,
    fetchQRCodes,
    fetchDocuments,
    refreshAll
  }
})

export const useLinqrStore = useLynqrStore
export const useLinksyStore = useLynqrStore
