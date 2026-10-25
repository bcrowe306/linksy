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
import ProfileView from '../views/ProfileView.vue'

const routes = [
  { path: '/', name: 'dashboard', component: DashboardView, meta: { requiresAuth: true } },
  { path: '/signup', name: 'signup', component: SignupView, meta: { guestOnly: true } },
  { path: '/login', name: 'login', component: LoginView, meta: { guestOnly: true } },
  { path: '/profile', name: 'profile', component: ProfileView, meta: { requiresAuth: true } },
  { path: '/links', name: 'links', component: LinksView, meta: { requiresAuth: true } },
  { path: '/qr-codes', name: 'qr-codes', component: QRCodesView, meta: { requiresAuth: true } },
  { path: '/qr-studio', name: 'qr-studio', component: QREditorView, meta: { requiresAuth: true } },
  { path: '/qr-studio/:id', name: 'qr-studio-edit', component: QREditorView, meta: { requiresAuth: true } },
  { path: '/documents', name: 'documents', component: DocumentsView, meta: { requiresAuth: true } },
  { path: '/analytics', name: 'analytics', component: AnalyticsView, meta: { requiresAuth: true } },
  { path: '/api-docs', name: 'docs', component: DocsView }, // Public interactive documentation
  { path: '/settings', name: 'settings', component: SettingsView, meta: { requiresAuth: true } },
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})

// Authentication Route Guard
router.beforeEach((to, from, next) => {
  // Check if this is an OAuth callback redirect arriving on root URL
  const searchParams = new URLSearchParams(window.location.search)
  const isOAuthCallback = searchParams.get('auth_success') === 'true' || !!searchParams.get('token')

  const token = localStorage.getItem('linksy_token')
  const user = localStorage.getItem('linksy_user')
  const hasAuthCookie = document.cookie.split(';').some(c => c.trim().startsWith('linksy_session='))
  const isAuthenticated = !!(token && user) || hasAuthCookie || isOAuthCallback

  if (to.meta.requiresAuth && !isAuthenticated) {
    // Redirect unauthenticated user to login with original target path
    next({
      path: '/login',
      query: to.fullPath !== '/' ? { redirect: to.fullPath } : {}
    })
  } else if (to.meta.guestOnly && isAuthenticated && !isOAuthCallback) {
    // Already authenticated, redirect away from guest-only pages to dashboard
    next({ path: '/' })
  } else {
    next()
  }
})
