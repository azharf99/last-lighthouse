import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: { port: 5173 },
  // core.wasm dilayani apa adanya dari public/wasm; jangan diproses bundler.
  assetsInclude: ['**/*.wasm'],
});
