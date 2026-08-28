import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// El proxy es solo para `npm run dev` (localhost:5173). En el contenedor,
// quien traduce /api hacia el backend es nginx (ver nginx.conf).
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
