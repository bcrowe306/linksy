import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

const backend = { target: 'http://localhost:8080', changeOrigin: true }

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    // Guides live in ../docs/guides so the Go server can reuse them for public pages.
    fs: { allow: ['..'] },
    proxy: {
      '/api': { ...backend, ws: true },
      '/s/': backend,
      '/q/': backend,
      '/d/': backend,
      '/qr/r': backend
    }
  }
})
