import { queryClientOptions } from '@app/queryClient'
import { router } from '@app/router'
import { VueQueryPlugin } from '@tanstack/vue-query'
import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

const app = createApp(App)
app.use(router)
app.use(VueQueryPlugin, queryClientOptions)

router.isReady().then(() => {
  app.mount('#app')
})
