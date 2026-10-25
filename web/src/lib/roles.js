// Mirrors internal/auth/rbac.go so the UI can explain and preview permissions.
import { Crown, ShieldCheck, Pencil, Eye } from 'lucide-vue-next'

export const ROLES = [
  { id: 'owner', label: 'Owner', icon: Crown, badge: 'badge-warning', description: 'Full control, including billing, ownership transfer and deleting the workspace.' },
  { id: 'admin', label: 'Admin', icon: ShieldCheck, badge: 'badge-brand', description: 'Manage members and invitations, rename the workspace, view the audit log.' },
  { id: 'member', label: 'Member', icon: Pencil, badge: 'badge-success', description: 'Create, edit and delete links, QR codes and documents.' },
  { id: 'viewer', label: 'Viewer', icon: Eye, badge: 'badge-neutral', description: 'Read-only access to assets and analytics, plus downloads.' }
]

export const roleMeta = (id) => ROLES.find((r) => r.id === id) || ROLES[3]

export const PERMISSION_MATRIX = [
  { label: 'View links, QR codes & analytics', roles: ['owner', 'admin', 'member', 'viewer'] },
  { label: 'Download QR codes & reports', roles: ['owner', 'admin', 'member', 'viewer'] },
  { label: 'Create & edit links, QR codes, documents', roles: ['owner', 'admin', 'member'] },
  { label: 'Invite, suspend & remove members', roles: ['owner', 'admin'] },
  { label: 'Rename workspace, view audit log', roles: ['owner', 'admin'] },
  { label: 'Grant the Admin role', roles: ['owner'] },
  { label: 'Billing, transfer ownership, delete workspace', roles: ['owner'] }
]

// Human-readable audit actions (internal/service audit calls).
export const AUDIT_ACTIONS = {
  'link.create': 'created a short link',
  'link.update': 'updated a short link',
  'link.delete': 'deleted a short link',
  'qr.create': 'created a QR code',
  'qr.update': 'updated a QR code',
  'qr.delete': 'deleted a QR code',
  'document.upload': 'uploaded a document',
  'document.replace': 'replaced a document',
  'document.delete': 'deleted a document',
  'api_key.create': 'created an API key',
  'api_key.revoke': 'revoked an API key',
  'team.create': 'created the workspace',
  'team.rename': 'renamed the workspace',
  'team.transfer_ownership': 'transferred ownership',
  'member.invite': 'invited a member',
  'member.invite_revoke': 'revoked an invitation',
  'member.invite_decline': 'declined an invitation',
  'member.join': 'joined the workspace',
  'member.leave': 'left the workspace',
  'member.role_change': 'changed a member’s role',
  'member.suspend': 'suspended a member',
  'member.reactivate': 'reactivated a member',
  'member.remove': 'removed a member',
  'member.password_reset': 'sent a password reset',
  'user.password_change': 'changed their password'
}
