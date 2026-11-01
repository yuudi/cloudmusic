import { defineConfig } from 'vitest/config';
import { playwright } from '@vitest/browser-playwright';
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { withPwa } from '@vite-pwa/sveltekit';

export default defineConfig({
	server: {
		proxy: {
			'/api': { target: 'http://localhost:8080', changeOrigin: true }
		}
	},
	plugins: [
		withPwa(
			{
				preprocess: vitePreprocess(),
				compilerOptions: {
					// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
					runes: ({ filename }) =>
						filename.split(/[/\\]/).includes('node_modules') ? undefined : true
				},
				adapter: adapter({ fallback: '200.html' })
			},
			{
				registerType: 'prompt',
				injectRegister: false,
				// Cache the public entry URL; preview serves the adapter fallback at /.
				kit: { spa: { fallbackMapping: '/' }, adapterFallback: '200.html' },
				manifest: {
					id: '/',
					name: 'Cloud Music',
					short_name: 'Cloud Music',
					description: '连接自己的存储空间，随时聆听你的音乐。',
					lang: 'zh-CN',
					start_url: '/',
					scope: '/',
					display: 'standalone',
					theme_color: '#175e50',
					background_color: '#f7faf8',
					icons: [
						{ src: '/pwa-192x192.png', sizes: '192x192', type: 'image/png' },
						{ src: '/pwa-512x512.png', sizes: '512x512', type: 'image/png' },
						{
							src: '/pwa-maskable-512x512.png',
							sizes: '512x512',
							type: 'image/png',
							purpose: 'maskable'
						}
					]
				},
				generateSW: {
					// A pure SPA has no prerendered pages; the plugin includes that optional glob.
					globStrict: false,
					cleanupOutdatedCaches: true,
					navigateFallback: '/',
					navigateFallbackDenylist: [/^\/api(?:\/|$)/, /^\/audio(?:\/|$)/, /\.[^/]+$/]
				}
			}
		)
	],
	test: {
		expect: { requireAssertions: true },
		projects: [
			{
				extends: './vite.config.ts',
				test: {
					name: 'web',
					browser: {
						enabled: true,
						provider: playwright(),
						instances: [{ browser: 'chromium', headless: true }]
					},
					include: ['src/**/*.svelte.{test,spec}.{js,ts}'],
					exclude: ['src/lib/server/**']
				}
			},

			{
				extends: './vite.config.ts',
				test: {
					name: 'unit',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
