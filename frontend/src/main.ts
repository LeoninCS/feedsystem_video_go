import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { reportError } from './utils/error-reporter'

const app = createApp(App)
app.use(createPinia())
app.use(router)

app.config.errorHandler = (err, _instance, info) => {
  reportError(err instanceof Error ? err : new Error(String(err)), { info })
}

app.mount('#app')
