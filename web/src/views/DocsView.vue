<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-white">Public REST API Reference</h1>
        <p class="text-xs text-gray-400 mt-0.5">Fully documented endpoints with interactive testing console.</p>
      </div>

      <div class="flex items-center gap-3">
        <a
          href="/api/v1/openapi.json"
          target="_blank"
          class="bg-gray-800 hover:bg-gray-750 border border-gray-700 text-gray-300 px-3 py-1.5 rounded-xl text-xs font-mono flex items-center gap-1.5"
        >
          <FileCode class="w-3.5 h-3.5" />
          <span>openapi.json</span>
        </a>
        <a
          href="/api/docs"
          target="_blank"
          class="bg-linksy-600 hover:bg-linksy-500 text-white px-3.5 py-1.5 rounded-xl text-xs font-semibold flex items-center gap-1.5"
        >
          <ExternalLink class="w-3.5 h-3.5" />
          <span>Full Scalar Portal</span>
        </a>
      </div>
    </div>

    <!-- Quick Code Snippet Examples -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-5 space-y-2">
        <span class="text-xs font-bold text-linksy-400 flex items-center gap-1.5">
          <Terminal class="w-3.5 h-3.5" />
          <span>Generate Dynamic QR Code (cURL)</span>
        </span>
        <pre class="bg-gray-950 p-3 rounded-xl font-mono text-[11px] text-gray-300 overflow-x-auto leading-relaxed">curl -X POST http://localhost:8080/api/v1/qr \
  -H "Content-Type: application/json" \
  -d '{
    "type": "website",
    "title": "Summer Menu",
    "target_url": "https://example.com/menu",
    "is_dynamic": true,
    "style": { "fg_color": "#1E40AF", "border_width": 2 }
  }'</pre>
      </div>

      <div class="bg-gray-800/60 border border-gray-700/80 rounded-2xl p-5 space-y-2">
        <span class="text-xs font-bold text-emerald-400 flex items-center gap-1.5">
          <Download class="w-3.5 h-3.5" />
          <span>Export Formats (PNG, JPEG, PDF, SVG)</span>
        </span>
        <pre class="bg-gray-950 p-3 rounded-xl font-mono text-[11px] text-gray-300 overflow-x-auto leading-relaxed"># Download Print-Ready PDF
GET /api/v1/qr/{id}/export?format=pdf&download=true

# High Res SVG Vector
GET /api/v1/qr/{id}/export?format=svg&size=1024</pre>
      </div>
    </div>

    <!-- Embedded Scalar API Reference Iframe -->
    <div class="bg-gray-900 border border-gray-800 rounded-2xl overflow-hidden shadow-2xl h-[700px]">
      <iframe src="/api/docs" class="w-full h-full border-0"></iframe>
    </div>
  </div>
</template>

<script setup>
import { FileCode, ExternalLink, Terminal, Download } from 'lucide-vue-next'
</script>
