import { expect, test } from '@playwright/test';

test('页面通过客户端路由导航，支持前进、后退和深层链接刷新', async ({ page }) => {
	await page.goto('/');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('你的音乐，随身聆听');
	const timeOrigin = await page.evaluate(() => performance.timeOrigin);

	await page.getByRole('navigation').getByRole('link', { name: '设置' }).click();
	await expect(page).toHaveURL('/settings');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置');
	expect(await page.evaluate(() => performance.timeOrigin)).toBe(timeOrigin);
	await page.goBack();
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('你的音乐，随身聆听');
	await page.goForward();
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置');
	await page.reload();
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置');
});

test('可直接访问深层链接，移动视口无横向溢出', async ({ page }) => {
	await page.goto('/settings');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置');
	expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(
		true
	);
});

test('提供安装清单、标准图标和 Apple 触屏图标', async ({ page }) => {
	await page.goto('/');
	const manifestLink = page.locator('link[rel="manifest"]');
	await expect(manifestLink).toHaveAttribute('href', /manifest\.webmanifest$/);
	const manifestUrl = await manifestLink.getAttribute('href');
	const response = await page.request.get(manifestUrl!);
	expect(response.ok()).toBe(true);
	const manifest = await response.json();
	expect(manifest).toMatchObject({ start_url: '/', scope: '/', display: 'standalone' });
	expect(manifest.icons).toEqual(
		expect.arrayContaining([
			expect.objectContaining({ sizes: '192x192', type: 'image/png' }),
			expect.objectContaining({ sizes: '512x512', purpose: 'maskable' })
		])
	);
	for (const icon of [...manifest.icons, { src: '/apple-touch-icon.png', sizes: '180x180' }]) {
		const dimensions = await page.evaluate(async (src: string) => {
			const image = new Image();
			image.src = src;
			await image.decode();
			return `${image.naturalWidth}x${image.naturalHeight}`;
		}, icon.src);
		expect(dimensions).toBe(icon.sizes);
	}
});

test('首次联网后可以离线打开未访问的路由，API 和音频不进入缓存', async ({ page, context }) => {
	await page.goto('/');
	await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
	await page.reload();
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	expect(await page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);

	// 绕过浏览器 HTTP 缓存，单独验证 Service Worker 不缓存业务资源。
	await page.evaluate(async () => {
		await Promise.all([
			fetch('/api/pwa-probe', { cache: 'no-store' }),
			fetch('/audio/pwa-probe.mp3', { cache: 'no-store' })
		]);
	});
	await context.setOffline(true);
	await page.goto('/settings');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置');
	await page.reload();
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置');

	for (const path of ['/api/pwa-probe', '/audio/pwa-probe.mp3']) {
		expect(
			await page.evaluate(async (url) => {
				try {
					await fetch(url, { cache: 'no-store' });
					return true;
				} catch {
					return false;
				}
			}, path)
		).toBe(false);
	}
});
