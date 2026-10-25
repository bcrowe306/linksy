// QR content types: metadata, defaults, and raw_data (de)serialization shared
// by QR Studio and the quick-create modal. Field names match internal/qr/payload.go.
import { Globe, FileText, Contact, Wifi, Smartphone, Type, Mail, MessageSquare, Phone } from 'lucide-vue-next'

export const QR_TYPES = [
  { id: 'website', label: 'Website', icon: Globe, dynamic: 'optional', description: 'Open a web page. Dynamic codes can be re-pointed later.' },
  { id: 'document', label: 'Document', icon: FileText, dynamic: 'always', description: 'Share a PDF, menu or file. Replace it any time.' },
  { id: 'vcard', label: 'Contact', icon: Contact, dynamic: 'optional', description: 'Save a contact card to the phone.' },
  { id: 'wifi', label: 'Wi-Fi', icon: Wifi, dynamic: 'never', description: 'Join a Wi-Fi network without typing the password.' },
  { id: 'app', label: 'App stores', icon: Smartphone, dynamic: 'always', description: 'Send iPhone and Android users to the right store.' },
  { id: 'text', label: 'Text', icon: Type, dynamic: 'never', description: 'Show a short plain-text message.' },
  { id: 'email', label: 'Email', icon: Mail, dynamic: 'never', description: 'Compose an email with recipient and subject filled in.' },
  { id: 'sms', label: 'SMS', icon: MessageSquare, dynamic: 'never', description: 'Start a text message with a prefilled body.' },
  { id: 'phone', label: 'Phone call', icon: Phone, dynamic: 'never', description: 'Dial a phone number.' }
]

export const typeMeta = (id) => QR_TYPES.find((t) => t.id === id) || QR_TYPES[0]

export function defaultsFor(type) {
  switch (type) {
    case 'vcard':
      return { format: 'vcard3', first_name: '', last_name: '', organization: '', title: '', phone: '', mobile: '', email: '', website: '', street: '', city: '', region: '', postal_code: '', country: '', note: '' }
    case 'wifi':
      return { ssid: '', password: '', encryption: 'WPA', hidden: false }
    case 'app':
      return { app_store_url: '', play_store_url: '', web_url: '' }
    case 'email':
      return { email: '', subject: '', body: '' }
    case 'sms':
      return { phone: '', message: '' }
    case 'phone':
      return { phone: '' }
    case 'text':
      return { text: '' }
  }
  return {}
}

// rawDataFor serializes typed fields into the API's raw_data string.
export function rawDataFor(type, data) {
  if (type === 'text') return data.text || ''
  if (['website', 'document'].includes(type)) return ''
  return JSON.stringify(data)
}

export function parseRawData(type, raw) {
  const base = defaultsFor(type)
  if (type === 'text') return { text: raw || '' }
  if (!raw) return base
  try {
    const parsed = JSON.parse(raw)
    // Legacy single-line vCard address
    if (type === 'vcard' && parsed.address && !parsed.street) parsed.street = parsed.address
    return { ...base, ...parsed }
  } catch {
    return base
  }
}

// isDynamicFor mirrors service.dynamicFor.
export function isDynamicFor(type, requested) {
  const d = typeMeta(type).dynamic
  return d === 'always' || (d === 'optional' && !!requested)
}

// validate returns a human-readable problem or '' when content is complete.
export function validateContent(content) {
  const { type, target_url, document_id, data } = content
  const urlOk = (u) => /^https?:\/\/[^\s.]+\.[^\s]+$/i.test((u || '').trim())
  switch (type) {
    case 'website':
      return urlOk(target_url) ? '' : 'Enter a full URL starting with https://'
    case 'document':
      return document_id ? '' : 'Choose or upload a document'
    case 'vcard':
      return (data.first_name || data.last_name || data.organization) ? '' : 'Add a name or organization'
    case 'wifi':
      return data.ssid ? '' : 'Enter the network name (SSID)'
    case 'app':
      if (!data.app_store_url && !data.play_store_url && !data.web_url) return 'Add at least one store or web link'
      return [data.app_store_url, data.play_store_url, data.web_url].filter(Boolean).every(urlOk) ? '' : 'Store links must be full https:// URLs'
    case 'text':
      return (data.text || '').trim() ? '' : 'Enter some text'
    case 'email':
      return /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(data.email || '') ? '' : 'Enter a valid recipient email'
    case 'sms':
    case 'phone':
      return /[0-9]{3,}/.test((data.phone || '').replace(/\D/g, '')) ? '' : 'Enter a phone number'
  }
  return ''
}

export const DEFAULT_STYLE = Object.freeze({
  fg_color: '#111827',
  bg_color: '#ffffff',
  is_transparent: false,
  gradient_type: 'none',
  gradient_start: '#4f46e5',
  gradient_end: '#a855f7',
  gradient_angle: 45,
  dot_style: 'square',
  eye_shape: 'square',
  eye_inner_shape: 'square',
  eye_color: '',
  eye_inner_color: '',
  error_correction: 'M',
  margin: 2,
  logo_base64: '',
  logo_size: 0.22,
  logo_bg_shape: 'rounded',
  logo_padding: 8,
  frame_style: 'none',
  border_width: 10,
  border_color: '#111827',
  border_radius: 24,
  frame_bg_color: '#ffffff',
  label_text: 'SCAN ME',
  label_color: '#ffffff',
  label_size: 100
})

export const STYLE_PRESETS = [
  { name: 'Classic', swatch: ['#111827', '#ffffff'], style: {} },
  { name: 'Indigo', swatch: ['#4f46e5', '#a855f7'], style: { gradient_type: 'linear', gradient_start: '#312e81', gradient_end: '#7c3aed', gradient_angle: 45, dot_style: 'extra-rounded', eye_shape: 'extra-rounded', eye_inner_shape: 'rounded', eye_color: '#312e81', frame_style: 'banner-bottom', border_color: '#4f46e5', label_text: 'SCAN ME' } },
  { name: 'Emerald', swatch: ['#065f46', '#ecfdf5'], style: { fg_color: '#064e3b', bg_color: '#ecfdf5', frame_bg_color: '#ecfdf5', dot_style: 'dots', eye_shape: 'circle', eye_inner_shape: 'circle', eye_color: '#047857', frame_style: 'card', border_width: 0, label_color: '#065f46', label_text: 'EXPLORE' } },
  { name: 'Sunset', swatch: ['#db2777', '#f59e0b'], style: { gradient_type: 'linear', gradient_start: '#be185d', gradient_end: '#d97706', gradient_angle: 135, dot_style: 'classy', eye_shape: 'leaf', eye_inner_shape: 'leaf', eye_color: '#be185d', eye_inner_color: '#d97706', frame_style: 'pill', border_color: '#be185d', label_text: 'VISIT US' } },
  { name: 'Ocean', swatch: ['#0369a1', '#38bdf8'], style: { gradient_type: 'radial', gradient_start: '#0c4a6e', gradient_end: '#0284c7', dot_style: 'rounded', eye_shape: 'rounded', eye_inner_shape: 'circle', frame_style: 'border', border_color: '#0369a1', border_width: 6 } },
  { name: 'Noir', swatch: ['#f4f4f5', '#18181b'], style: { fg_color: '#18181b', bg_color: '#ffffff', frame_style: 'banner-top', frame_bg_color: '#ffffff', border_color: '#18181b', dot_style: 'extra-rounded', eye_shape: 'circle', eye_inner_shape: 'circle', label_text: 'MENU' } }
]
