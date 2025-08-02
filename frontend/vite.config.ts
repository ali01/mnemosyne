import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig, type UserConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: 'http://localhost:8080',
				changeOrigin: true
			}
		}
	},
	/**
	 * Test configuration:
	 * - Run tests: npm run test
	 * - Run with coverage: npm run test:coverage
	 * - View coverage report: open coverage/index.html
	 */
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}'],
		environment: 'jsdom',
		globals: true,
		setupFiles: ['src/lib/test/setup.ts'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'json', 'html'],
			thresholds: {
				global: {
					// Starting with reasonable thresholds, will increase gradually
					branches: 70,
					functions: 70,
					lines: 70,
					statements: 70
				}
			},
			exclude: [
				// Build outputs
				'coverage/**',
				'dist/**',
				'build/**',
				'.svelte-kit/**',

				// Dependencies
				'node_modules/**',

				// Test files
				'**/*{.,-}test.{js,cjs,mjs,ts,tsx,jsx}',
				'**/*{.,-}spec.{js,cjs,mjs,ts,tsx,jsx}',
				'**/__tests__/**',
				'src/lib/test/**',
				'tests/**',
				'cypress/**',

				// Type definitions
				'**/*.d.ts',
				'src/lib/types/**',

				// Config files
				'**/{karma,rollup,webpack,vite,vitest,jest,ava,babel,nyc,cypress,tsup,build}.config.*',
				'**/.{eslint,mocha,prettier}rc.{js,cjs,yml}',
				'svelte.config.js',
				'postcss.config.js',
				'playwright.config.ts',

				// Routes (already tested via integration tests)
				'src/routes/**',

				// Static assets
				'static/**'
			]
		}
	}
});
