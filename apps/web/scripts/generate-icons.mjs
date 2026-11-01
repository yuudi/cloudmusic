import { readFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

const staticDir = new URL('../static/', import.meta.url);
const svg = await readFile(new URL('icon.svg', staticDir), 'utf8');
const browser = await chromium.launch();

try {
	const page = await browser.newPage({ deviceScaleFactor: 1 });
	for (const [size, filename, maskable] of [
		[192, 'pwa-192x192.png', false],
		[512, 'pwa-512x512.png', false],
		[512, 'pwa-maskable-512x512.png', true],
		[180, 'apple-touch-icon.png', true]
	]) {
		await page.setViewportSize({ width: size, height: size });
		const source = maskable ? svg.replace(' rx="112"', '') : svg;
		await page.setContent(
			`<style>body{margin:0}svg{display:block;width:100%;height:100%}</style>${source}`
		);
		await page.screenshot({
			path: fileURLToPath(new URL(filename, staticDir)),
			omitBackground: true
		});
	}
} finally {
	await browser.close();
}
