import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/todos': 'http://localhost:8761',
      '/sections': 'http://localhost:8761',
      '/raw': 'http://localhost:8761',
    }
  }
})
