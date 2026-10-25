import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { applyTheme } from './stores/theme'
import './style.css'

applyTheme()

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
