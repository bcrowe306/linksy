<template>
  <Teleport to="body">
  <div v-if="isOpen" class="fixed inset-0 z-50 overflow-hidden flex justify-end" role="dialog" aria-modal="true" aria-label="Help and quick docs" @keydown.esc="$emit('close')">
    <!-- Backdrop -->
    <div @click="$emit('close')" class="absolute inset-0 bg-black/60 backdrop-blur-sm"></div>

    <!-- Drawer Panel -->
    <div class="relative w-full max-w-md bg-elevated border-l border-line h-full p-6 overflow-y-auto z-10 flex flex-col justify-between shadow-pop animate-slide-in-right">
      <div>
        <div class="flex items-center justify-between pb-4 border-b border-line mb-6">
          <div class="flex items-center gap-2">
            <BookOpen class="w-5 h-5 text-brand-500" />
            <h2 class="text-base font-bold text-fg">Lynqr Quick Help & Docs</h2>
          </div>
          <button @click="$emit('close')" class="p-1.5 rounded-lg text-fg-muted hover:text-fg hover:bg-gray-800">
            <X class="w-4 h-4" />
          </button>
        </div>

        <div class="space-y-6 text-xs text-fg-muted">
          <!-- Guide 1: Dynamic QR Codes -->
          <div class="bg-subtle/60 p-4 rounded-xl border border-line space-y-2">
            <div class="flex items-center gap-2 text-sm font-semibold text-brand-500">
              <RefreshCw class="w-4 h-4" />
              <span>Dynamic QR Codes (Never Reprint)</span>
            </div>
            <p class="leading-relaxed text-fg-muted">
              Dynamic QR codes route through a shortened redirect link (e.g. <code class="text-brand-500">/s/slug</code>). This allows you to change the destination website, menu, or file at any time without needing to reprint stickers, flyers, or merchandise.
            </p>
          </div>

          <!-- Guide 2: Document Sharing -->
          <div class="bg-subtle/60 p-4 rounded-xl border border-line space-y-2">
            <div class="flex items-center gap-2 text-sm font-semibold text-emerald-400">
              <FileUp class="w-4 h-4" />
              <span>Document QR Sharing & Reuse</span>
            </div>
            <p class="leading-relaxed text-fg-muted">
              Upload PDF reports or restaurant menus. When your document changes next week or next quarter, simply upload the new file in the Documents tab. Your physical QR codes will immediately serve the latest version.
            </p>
          </div>

          <!-- Guide 3: QR Code Types -->
          <div class="bg-subtle/60 p-4 rounded-xl border border-line space-y-2">
            <div class="flex items-center gap-2 text-sm font-semibold text-amber-400">
              <QrCode class="w-4 h-4" />
              <span>Specialized QR Payloads</span>
            </div>
            <ul class="list-disc list-inside space-y-1 text-fg-muted">
              <li><strong>Contact Cards (vCard)</strong>: Adds phone, email, and company directly to phone contacts.</li>
              <li><strong>Wi-Fi Network</strong>: Automatically prompts guests to connect without typing passwords.</li>
              <li><strong>SMS / Email</strong>: Pre-populates recipient and subject with one scan.</li>
            </ul>
          </div>

          <!-- Guide 4: Developer REST API -->
          <div class="bg-subtle/60 p-4 rounded-xl border border-line space-y-2">
            <div class="flex items-center gap-2 text-sm font-semibold text-purple-400">
              <Code2 class="w-4 h-4" />
              <span>Developer API Integration</span>
            </div>
            <p class="leading-relaxed text-fg-muted">
              Automate link generation from your backend or CI/CD with our REST API:
            </p>
            <pre class="bg-gray-950 p-2.5 rounded-lg font-mono text-[11px] text-fg-muted overflow-x-auto">curl -X POST https://lynqr.app/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"target_url": "https://myapp.com"}'</pre>
          </div>
        </div>
      </div>

      <!-- External Docs Link -->
      <div class="pt-6 border-t border-line">
        <router-link
          to="/api-docs"
          @click="$emit('close')"
          class="btn-primary w-full"
        >
          <ExternalLink class="w-4 h-4" />
          <span>Open Full Interactive API Reference</span>
        </router-link>
      </div>
    </div>
  </div>
  </Teleport>
</template>

<script setup>
import { BookOpen, X, RefreshCw, FileUp, QrCode, Code2, ExternalLink } from 'lucide-vue-next'

defineProps({
  isOpen: { type: Boolean, default: false }
})
defineEmits(['close'])
</script>
