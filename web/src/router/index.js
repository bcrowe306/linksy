import { createRouter, createWebHistory } from 'vue-router'
import DashboardView from '../views/DashboardView.vue'
import LinksView from '../views/LinksView.vue'
import QRCodesView from '../views/QRCodesView.vue'
import QREditorView from '../views/QREditorView.vue'
import DocumentsView from '../views/DocumentsView.vue'
import AnalyticsView from '../views/AnalyticsView.vue'
import DocsView from '../views/DocsView.vue'
import SettingsView from '../views/SettingsView.vue'
import SignupView from '../views/SignupView.vue'
import LoginView from '../views/LoginView.vue'

const routes = [
  { path: '/', name: 'dashboard', component: DashboardView },
  { path: '/signup', name: 'signup', component: SignupView },
  { path: '/login', name: 'login', component: LoginView },
  { path: '/links', name: 'links', component: LinksView },
  { path: '/qr-codes', name: 'qr-codes', component: QRCodesView },
  { path: '/qr-studio', name: 'qr-studio', component: QREditorView },
  { path: '/qr-studio/:id', name: 'qr-studio-edit', component: QREditorView },
  { path: '/documents', name: 'documents', component: DocumentsView },
  { path: '/analytics', name: 'analytics', component: AnalyticsView },
  { path: '/api-docs', name: 'docs', component: DocsView },
  { path: '/settings', name: 'settings', component: SettingsView },
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})
