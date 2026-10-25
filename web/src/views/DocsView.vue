<template>
  <div class="space-y-6">
    <PageHeader title="API Reference" subtitle="Automate links, QR codes, documents and analytics. Authenticate with an API key from your profile.">
      <template #actions>
        <a href="/api/v1/openapi.yaml" class="btn-secondary font-mono text-xs" download><FileCode class="h-4 w-4" /> openapi.yaml</a>
        <a href="/api/docs" target="_blank" rel="noopener" class="btn-secondary"><ExternalLink class="h-4 w-4" /> Full screen</a>
        <router-link v-if="auth.isAuthenticated" to="/profile?tab=api-keys" class="btn-primary"><KeyRound class="h-4 w-4" /> Get an API key</router-link>
      </template>
    </PageHeader>

    <section class="card overflow-hidden">
      <header class="flex flex-wrap items-center justify-between gap-3 border-b border-line px-5 py-3">
        <h2 class="text-sm font-semibold text-fg">Quick start: create a dynamic QR code</h2>
        <div class="flex gap-1 rounded-xl bg-subtle p-1" role="tablist">
          <button v-for="l in langs" :key="l" role="tab" :aria-selected="lang === l" class="rounded-lg px-3 py-1 text-xs font-medium" :class="lang === l ? 'bg-surface text-fg shadow-card' : 'text-fg-muted'" @click="lang = l">{{ l }}</button>
        </div>
      </header>
      <div class="relative">
        <pre class="overflow-x-auto bg-[#0c0c0e] p-5 font-mono text-[12px] leading-relaxed text-zinc-200"><code>{{ samples[lang] }}</code></pre>
        <div class="absolute right-3 top-3"><CopyButton :value="samples[lang]" label="Copy code" /></div>
      </div>
    </section>

    <div class="card h-[78vh] min-h-[600px] overflow-hidden">
      <iframe :key="theme.effectiveTheme" :src="`/api/docs?theme=${theme.effectiveTheme}`" title="Lynqr API reference" class="h-full w-full border-0" />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { FileCode, ExternalLink, KeyRound } from 'lucide-vue-next'
import { useThemeStore } from '../stores/theme'
import { useAuthStore } from '../stores/auth'
import PageHeader from '../components/ui/PageHeader.vue'
import CopyButton from '../components/ui/CopyButton.vue'

const theme = useThemeStore()
const auth = useAuthStore()
const origin = window.location.origin
const langs = ['cURL', 'Python', 'JavaScript', 'Go']
const lang = ref('cURL')
const body = '{"type":"website","title":"Poster","target_url":"https://example.com","is_dynamic":true}'

const samples = {
  cURL: `# 1. Create the code
curl -X POST ${origin}/api/v1/qr \\
  -H "Authorization: Bearer $LYNQR_KEY" \\
  -H "Content-Type: application/json" \\
  -d '${body}'

# 2. Download it as SVG
curl -o poster.svg "${origin}/api/v1/qr/<id>/export?format=svg" \\
  -H "Authorization: Bearer $LYNQR_KEY"`,
  Python: `import os, requests

API = "${origin}/api/v1"
headers = {"Authorization": f"Bearer {os.environ['LYNQR_KEY']}"}

qr = requests.post(f"{API}/qr", headers=headers, json={
    "type": "website", "title": "Poster",
    "target_url": "https://example.com", "is_dynamic": True,
}).json()

svg = requests.get(f"{API}/qr/{qr['id']}/export", params={"format": "svg"}, headers=headers)
open("poster.svg", "wb").write(svg.content)
print("Scan URL:", qr["scan_url"])`,
  JavaScript: `const API = "${origin}/api/v1";
const headers = { Authorization: \`Bearer \${process.env.LYNQR_KEY}\`, "Content-Type": "application/json" };

const qr = await fetch(\`\${API}/qr\`, {
  method: "POST",
  headers,
  body: JSON.stringify({ type: "website", title: "Poster", target_url: "https://example.com", is_dynamic: true }),
}).then((r) => r.json());

const svg = await fetch(\`\${API}/qr/\${qr.id}/export?format=svg\`, { headers }).then((r) => r.text());
console.log("Scan URL:", qr.scan_url);`,
  Go: `package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	req, _ := http.NewRequest("POST", "${origin}/api/v1/qr",
		strings.NewReader(\`${body}\`))
	req.Header.Set("Authorization", "Bearer "+os.Getenv("LYNQR_KEY"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`
}
</script>
