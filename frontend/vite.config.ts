import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  base: './',
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/subtitle': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
      '/remux': {
        target: 'http://localhost:8091',
        changeOrigin: true,
      },
    },
  },
});
