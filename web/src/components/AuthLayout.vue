<template>
  <div class="grid min-h-screen lg:grid-cols-[1.05fr_1fr]">
    <!-- Brand panel -->
    <aside class="relative hidden overflow-hidden bg-[#0b0a1a] lg:flex lg:flex-col lg:justify-between p-12 text-white">
      <div class="pointer-events-none absolute -left-32 -top-32 h-[28rem] w-[28rem] rounded-full bg-brand-600/40 blur-[120px]" />
      <div class="pointer-events-none absolute -bottom-40 right-0 h-[26rem] w-[26rem] rounded-full bg-fuchsia-600/25 blur-[120px]" />
      <div class="pointer-events-none absolute inset-0 opacity-[0.07] [background-image:linear-gradient(white_1px,transparent_1px),linear-gradient(90deg,white_1px,transparent_1px)] [background-size:44px_44px]" />

      <router-link to="/" class="relative"><span class="[&_span]:!text-white"><LynqrLogo size="lg" /></span></router-link>

      <div class="relative max-w-md">
        <h2 class="text-4xl font-bold leading-tight tracking-tight">
          Every scan and click,<br /><span class="bg-gradient-to-r from-brand-300 via-violet-300 to-fuchsia-300 bg-clip-text text-transparent">beautifully tracked.</span>
        </h2>
        <ul class="mt-8 space-y-4 text-sm text-white/75">
          <li v-for="f in features" :key="f.text" class="flex items-start gap-3">
            <span class="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-lg bg-white/10 ring-1 ring-white/15"><component :is="f.icon" class="h-3.5 w-3.5" /></span>
            {{ f.text }}
          </li>
        </ul>
      </div>

      <p class="relative text-xs text-white/40">© {{ new Date().getFullYear() }} Lynqr · Dynamic QR codes & short links</p>
    </aside>

    <!-- Form panel -->
    <main class="flex flex-col items-center justify-center px-6 py-12 sm:px-12">
      <div class="w-full max-w-sm animate-slide-up">
        <div class="mb-8 lg:hidden"><LynqrLogo /></div>
        <h1 class="text-2xl font-bold tracking-tight text-fg">{{ title }}</h1>
        <p v-if="subtitle" class="mt-1.5 text-sm text-fg-muted">{{ subtitle }}</p>
        <div class="mt-8"><slot /></div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { Zap, BarChart3, Users, ShieldCheck } from 'lucide-vue-next'
import LynqrLogo from './ui/LynqrLogo.vue'

defineProps({ title: String, subtitle: String })
const features = [
  { icon: Zap, text: 'Dynamic QR codes and short links you can re-point without reprinting.' },
  { icon: BarChart3, text: 'Real-time analytics by device, browser, referrer and location.' },
  { icon: Users, text: 'Workspaces with roles, invitations and a full audit trail.' },
  { icon: ShieldCheck, text: 'Enterprise SSO via WorkOS, with Google, Microsoft and GitHub sign-in.' }
]
</script>
