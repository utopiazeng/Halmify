import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig(({ command }) => ({
  base: command === 'build' ? '/demo/' : '/',
  plugins: [vue()],
  build: {
    sourcemap: false,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }
          if (id.includes('/@iconify/vue/')) {
            return 'icons'
          }
          if (id.includes('/vue/') || id.includes('/vue-router/')) {
            return 'vue-vendor'
          }
          if (id.includes('/axios/')) {
            return 'http'
          }
          if (id.includes('/canvas-confetti/')) {
            return 'celebration'
          }
        }
      }
    }
  },
  server: {
    host: '0.0.0.0',
    port: 3001,
    proxy: {
      '/api': {
        target: 'http://localhost:8092',
        changeOrigin: true,
        ws: true
      }
    }
  }
}))
