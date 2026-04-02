import { svelte } from '@sveltejs/vite-plugin-svelte';
import { defineConfig, createLogger } from 'vite';
import path from 'path';

const logger = createLogger();
const originalWarn = logger.warn.bind(logger);
logger.warn = (msg, options) => {
  if (msg.includes('optimizeDeps.esbuildOptions')) return;
  originalWarn(msg, options);
};

export default defineConfig({
  customLogger: logger,
  plugins: [svelte()],
  resolve: {
    alias: {
      '$lib': path.resolve(__dirname, 'src/lib'),
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:5555',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['src/lib/test/setup.ts'],
  },
});
