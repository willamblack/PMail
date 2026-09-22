import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { ElementPlusResolver } from 'unplugin-vue-components/resolvers'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    AutoImport({
      resolvers: [ElementPlusResolver()],
    }),
    Components({
      resolvers: [ElementPlusResolver()],
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    // Development proxies expose the local mailbox API. Do not share them on
    // the LAN or grant arbitrary websites access to their responses.
    host: '127.0.0.1',
    cors: false,
    proxy: {
      "/api": "http://127.0.0.1/",
      "/attachments":"http://127.0.0.1/"
    }
  },
  preview: {
    host: '127.0.0.1',
    cors: false,
  }
})
