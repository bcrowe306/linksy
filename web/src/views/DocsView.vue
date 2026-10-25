<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <h1 class="page-title">API Reference</h1>
        <p class="page-subtitle">Every endpoint, with an interactive console. Authenticate with a Bearer API key from your profile.</p>
      </div>

      <div class="flex items-center gap-3">
        <a
          href="/api/v1/openapi.json"
          target="_blank"
          class="btn-secondary font-mono text-xs"
        >
          <FileCode class="w-3.5 h-3.5" />
          <span>openapi.json</span>
        </a>
        <a
          href="/api/docs"
          target="_blank"
          class="btn-primary"
        >
          <ExternalLink class="w-3.5 h-3.5" />
          <span>Full Scalar Portal</span>
        </a>
      </div>
    </div>

    <!-- Quick Code Snippet Examples -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="card p-5 space-y-2">
        <span class="text-xs font-bold text-brand-500 flex items-center gap-1.5">
          <Terminal class="w-3.5 h-3.5" />
          <span>Generate Dynamic QR Code (cURL)</span>
        </span>
        <pre class="bg-[#0c0c0e] p-3 rounded-xl font-mono text-[11px] text-zinc-300 overflow-x-auto leading-relaxed">curl -X POST https://lynqr.app/api/v1/qr \
  -H "Authorization: Bearer lk_live_…" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "website",
    "title": "Summer Menu",
    "target_url": "https://example.com/menu",
    "is_dynamic": true,
    "style": { "fg_color": "#1E40AF", "border_width": 2 }
  }'</pre>
      </div>

      <div class="card p-5 space-y-2">
        <span class="text-xs font-bold text-emerald-500 flex items-center gap-1.5">
          <Download class="w-3.5 h-3.5" />
          <span>Export Formats (PNG, JPEG, PDF, SVG)</span>
        </span>
        <pre class="bg-[#0c0c0e] p-3 rounded-xl font-mono text-[11px] text-zinc-300 overflow-x-auto leading-relaxed"># Download Print-Ready PDF
GET /api/v1/qr/{id}/export?format=pdf&download=true

# High Res SVG Vector
GET /api/v1/qr/{id}/export?format=svg&size=1024</pre>
      </div>
    </div>

    <!-- Embedded Scalar API Reference Iframe -->
    <div class="card overflow-hidden h-[75vh] min-h-[600px]">
      <iframe src="/api/docs" title="Lynqr API reference" class="w-full h-full border-0"></iframe>
    </div>
  </div>
</template>

<script setup>
import { FileCode, ExternalLink, Terminal, Download } from 'lucide-vue-next'
</script>
