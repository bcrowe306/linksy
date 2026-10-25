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
import InviteView from '../views/InviteView.vue'
import ResetPasswordView from '../views/ResetPasswordView.vue'
import { useAuthStore } from '../stores/auth'

// The admin console is loaded on demand (staff only).
const admin = (name) => () => import(`../views/admin/${name}.vue`)

const routes = [
  { path: '/', name: 'dashboard', component: DashboardView, meta: { requiresAuth: true } },
  { path: '/signup', name: 'signup', component: SignupView, meta: { guestOnly: true, bare: true } },
  { path: '/login', name: 'login', component: LoginView, meta: { guestOnly: true, bare: true } },
  { path: '/invite/:token', name: 'invite', component: InviteView, meta: { bare: true } },
  { path: '/reset-password', name: 'reset-password', component: ResetPasswordView, meta: { bare: true } },
  { path: '/profile', name: 'profile', component: ProfileView, meta: { requiresAuth: true } },
  { path: '/links', name: 'links', component: LinksView, meta: { requiresAuth: true } },
  { path: '/qr-codes', name: 'qr-codes', component: QRCodesView, meta: { requiresAuth: true } },
  { path: '/qr-studio', name: 'qr-studio', component: QREditorView, meta: { requiresAuth: true } },
  { path: '/qr-studio/:id', name: 'qr-studio-edit', component: QREditorView, meta: { requiresAuth: true } },
  { path: '/documents', name: 'documents', component: DocumentsView, meta: { requiresAuth: true } },
  { path: '/analytics', name: 'analytics', component: AnalyticsView, meta: { requiresAuth: true } },
  { path: '/api-docs', name: 'docs', component: DocsView }, // Public interactive documentation
  { path: '/settings', name: 'settings', component: SettingsView, meta: { requiresAuth: true } },
  {
    path: '/admin',
    component: admin('AdminView'),
    meta: { requiresAuth: true, noindex: true },
    children: [
      { path: '', name: 'admin', component: admin('AdminOverview') },
      { path: 'workspaces', name: 'admin-workspaces', component: admin('AdminWorkspaces') },
      { path: 'users', name: 'admin-users', component: admin('AdminUsers') },
      { path: 'codes', name: 'admin-codes', component: admin('AdminCodes') },
      { path: 'moderation', name: 'admin-moderation', component: admin('AdminModeration') },
      { path: 'audit', name: 'admin-audit', component: admin('AdminAudit') }
    ]
  },
  { path: '/team', redirect: '/settings' },
  { path: '/users', redirect: '/settings' },
]

export const router = createRouter({
  history: createWebHistory(),
  routes
})

// Authentication route guard: the session lives in an HttpOnly cookie, so the
// guard asks the server (once per page load) whether we're signed in.
router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.ensureLoaded()

  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { path: '/login', query: to.fullPath !== '/' ? { redirect: to.fullPath } : {} }
  }
  if (to.meta.guestOnly && auth.isAuthenticated) {
    const redirect = String(to.query.redirect || '/')
    return redirect.startsWith('/') && !redirect.startsWith('//') ? redirect : '/'
  }
  return true
})
