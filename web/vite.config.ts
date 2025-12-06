import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  base: './', // Use relative paths so assets work with any path prefix
  build: {
    outDir: '../internal/assets/web/dist',
    emptyOutDir: true,
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/devops/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/devops/deploy': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
