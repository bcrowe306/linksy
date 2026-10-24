import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import LinksView from '../views/LinksView.vue'
import QRCodesView from '../views/QRCodesView.vue'
import DocumentsView from '../views/DocumentsView.vue'
import AnalyticsView from '../views/AnalyticsView.vue'
import DocsView from '../views/DocsView.vue'
import SettingsView from '../views/SettingsView.vue'

const routes = [
  { path: '/', name: 'dashboard', component: DashboardView },
  { path: '/links', name: 'links', component: LinksView },
  { path: '/qr-codes', name: 'qr-codes', component: QRCodesView },
  { path: '/documents', name: 'documents', component: DocumentsView },
  { path: '/analytics', name: 'analytics', component: AnalyticsView },
  { path: '/api-docs', name: 'docs', component: DocsView },
  { path: '/settings', name: 'settings', component: SettingsView },
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
