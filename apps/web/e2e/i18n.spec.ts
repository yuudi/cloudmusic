import { expect, test } from '@playwright/test';

test.use({ locale: 'en-US' });

test('切换语言即时更新标题、标签和已有错误，保留表单及页面状态', async ({ page }) => {
	await page.goto('/register');
	await expect(page).toHaveTitle('Create an account · Cloud Music');
	await expect(page.locator('html')).toHaveAttribute('lang', 'en');
	const origin = await page.evaluate(() => performance.timeOrigin);
	await page.getByLabel('Username', { exact: true }).fill('music_fan');
	await page.getByLabel('Display name (optional)', { exact: true }).fill('音乐爱好者');
	await page.getByLabel('Password', { exact: true }).fill('password');
	await page.getByLabel('Confirm password', { exact: true }).fill('different');
	await page.getByRole('button', { name: 'Create account and log in' }).click();
	await expect(page.getByRole('alert')).toHaveText('The passwords do not match.');
	await page.getByRole('combobox', { name: 'Language' }).selectOption('zh-CN');
	await expect(page).toHaveTitle('注册 · Cloud Music');
	await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN');
	await expect(page.getByRole('alert')).toHaveText('两次输入的密码不一致。');
	await expect(page.getByLabel('用户名', { exact: true })).toHaveValue('music_fan');
	await expect(page.getByLabel('显示名（可选）', { exact: true })).toHaveValue('音乐爱好者');
	await expect(page.getByLabel('密码', { exact: true })).toHaveValue('password');
	expect(await page.evaluate(() => performance.timeOrigin)).toBe(origin);
	await page.reload();
	await expect(page).toHaveTitle('注册 · Cloud Music');
	await expect(page.getByRole('combobox', { name: '语言' })).toHaveValue('zh-CN');
});

test.describe('模拟 API 的身份场景', () => {
	test.use({ serviceWorkers: 'block' });

	test('语言偏好跨标签页同步，登录身份、用户内容和未保存资料保持不变', async ({
		page,
		context
	}) => {
		await context.addInitScript(() => localStorage.setItem('cloudmusic.session', 'saved-token'));
		let checks = 0;
		await context.route('**/api/v1/auth/me', (route) => {
			checks++;
			return route.fulfill({
				json: { user: { id: 'guest', username: null, display_name: '用户原文', is_guest: true } }
			});
		});
		await page.goto('/settings');
		await expect(page.getByLabel('Display name', { exact: true })).toHaveValue('用户原文');
		await page.getByLabel('Display name', { exact: true }).fill('未保存的名字');
		await page.getByLabel('New password', { exact: true }).fill('unsaved-password');
		await page.getByRole('combobox', { name: 'Language' }).selectOption('zh-CN');
		await expect(page.getByLabel('显示名', { exact: true })).toHaveValue('未保存的名字');
		await expect(page.getByLabel('新密码', { exact: true })).toHaveValue('unsaved-password');
		expect(await page.evaluate(() => localStorage.getItem('cloudmusic.session'))).toBe(
			'saved-token'
		);
		const other = await context.newPage();
		try {
			await other.goto('/settings');
			await expect(other.getByLabel('显示名', { exact: true })).toHaveValue('用户原文');
			const before = checks;
			await other.getByRole('combobox', { name: '语言' }).selectOption('en');
			await expect(page.locator('html')).toHaveAttribute('lang', 'en');
			await expect(page.getByLabel('Display name', { exact: true })).toHaveValue('未保存的名字');
			expect(checks).toBe(before);
		} finally {
			await other.close();
		}
	});
});

test('离线时可切换语言，重新打开页面仍可加载两种语言', async ({ page, context }) => {
	await page.goto('/register');
	await page.evaluate(() => navigator.serviceWorker.ready.then(() => undefined));
	await page.reload();
	await expect(page.getByRole('combobox', { name: 'Language' })).toBeVisible();
	await context.setOffline(true);
	await page.getByRole('combobox', { name: 'Language' }).selectOption('zh-CN');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('创建账号');
	await page.reload();
	await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN');
	await page.getByRole('combobox', { name: '语言' }).selectOption('en');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('Create an account');
});

test('浏览器禁用语言偏好存储时仍能即时切换', async ({ page }) => {
	await page.addInitScript(() => {
		const get = Storage.prototype.getItem;
		const set = Storage.prototype.setItem;
		Storage.prototype.getItem = function (key) {
			if (key === 'cloudmusic.locale') throw new DOMException('Blocked', 'SecurityError');
			return get.call(this, key);
		};
		Storage.prototype.setItem = function (key, value) {
			if (key === 'cloudmusic.locale') throw new DOMException('Blocked', 'SecurityError');
			return set.call(this, key, value);
		};
	});
	await page.goto('/');
	await page.getByRole('combobox', { name: 'Language' }).selectOption('zh-CN');
	await expect(page.locator('html')).toHaveAttribute('lang', 'zh-CN');
	await expect(page.getByRole('heading', { level: 1 })).toHaveText('你的音乐，随身聆听');
});

test.describe('未支持的浏览器语言', () => {
	test.use({ locale: 'fr-FR' });
	test('首次访问回退到英文', async ({ page }) => {
		await page.goto('/');
		await expect(page.locator('html')).toHaveAttribute('lang', 'en');
		await expect(page.getByRole('heading', { level: 1 })).toHaveText('Your music, wherever you go');
	});
});
