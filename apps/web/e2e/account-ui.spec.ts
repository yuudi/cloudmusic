import { expect, test } from '@playwright/test';

// Keep mocked API requests visible to Playwright, including on new tabs.
test.use({ serviceWorkers: 'block' });

const guest = { id: 'guest', username: null, display_name: '游客的名字', is_guest: true };

test('同源标签页同步账号切换和退出，不保留上个账号的资料', async ({ page, context }) => {
	await context.route('**/api/v1/auth/me', (route) => {
		const second = route.request().headers().authorization === 'Bearer second';
		return route.fulfill({
			json: {
				user: {
					...guest,
					id: second ? 'second' : 'first',
					display_name: second ? '第二个账号' : '第一个账号'
				}
			}
		});
	});
	await page.goto('/');
	await page.evaluate(() => localStorage.setItem('cloudmusic.session', 'first'));
	await page.goto('/settings');
	await page.reload();
	await expect(page.getByLabel('显示名', { exact: true })).toHaveValue('第一个账号');
	const other = await context.newPage();
	try {
		await other.goto('/settings');
		await expect(other.getByLabel('显示名', { exact: true })).toHaveValue('第一个账号');
		await other.evaluate(() => localStorage.setItem('cloudmusic.session', 'second'));
		await expect(page.getByLabel('显示名', { exact: true })).toHaveValue('第二个账号');
		await other.evaluate(() => localStorage.removeItem('cloudmusic.session'));
		await expect(page.getByRole('button', { name: '以游客身份继续' })).toBeVisible();
		await expect(page.getByLabel('显示名', { exact: true })).toHaveCount(0);
	} finally {
		await other.close();
	}
});

test('注册密码校验、用户名冲突与重试', async ({ page }) => {
	let submissions = 0;
	await page.route('**/api/v1/auth/register', async (route) => {
		submissions++;
		await route.fulfill({ status: 409, json: { error: 'username_taken' } });
	});
	await page.goto('/register');
	await page.getByLabel('用户名', { exact: true }).fill('music');
	await page.getByLabel('密码', { exact: true }).fill('password');
	await page.getByLabel('确认密码', { exact: true }).fill('different');
	await page.getByRole('button', { name: '注册并登录' }).click();
	await expect(page.getByRole('alert')).toHaveText('两次输入的密码不一致。');
	expect(submissions).toBe(0);
	await page.getByLabel('确认密码', { exact: true }).fill('password');
	await page.getByRole('button', { name: '注册并登录' }).click();
	await expect(page.getByRole('alert')).toContainText('用户名已被占用');
	expect(submissions).toBe(1);
	await expect(page.getByRole('button', { name: '注册并登录' })).toBeEnabled();
	await expect(page.getByLabel('用户名', { exact: true })).toHaveValue('music');
});

test('恢复失败时保留身份并允许重试，过期后不自动创建游客', async ({ page }) => {
	await page.addInitScript(() => localStorage.setItem('cloudmusic.session', 'saved-token'));
	let status = 503;
	await page.route('**/api/v1/auth/me', (route) =>
		route.fulfill({ status, json: status === 200 ? { user: guest } : { error: 'unavailable' } })
	);
	await page.goto('/settings');
	await expect(page.getByRole('alert')).toContainText('已保留本机凭据');
	expect(await page.evaluate(() => localStorage.getItem('cloudmusic.session'))).toBe('saved-token');
	status = 200;
	await page.getByRole('button', { name: '重试身份验证' }).click();
	await expect(page.getByRole('heading', { name: '游客账号', exact: true })).toBeVisible();
	status = 401;
	await page.reload();
	await expect(page.getByText(/会话已过期或已退出/)).toBeVisible();
	expect(await page.evaluate(() => localStorage.getItem('cloudmusic.session'))).toBeNull();
	await expect(page.getByRole('button', { name: '以游客身份继续' })).toBeVisible();
});

test('深浅色跟随系统，窄屏账户表单无横向溢出', async ({ page }) => {
	await page.setViewportSize({ width: 375, height: 812 });
	await page.goto('/register');
	for (const colorScheme of ['dark', 'light'] as const) {
		await page.emulateMedia({ colorScheme });
		const expected = colorScheme === 'dark' ? 'rgb(18, 18, 18)' : 'rgb(250, 250, 250)';
		await expect
			.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).backgroundColor))
			.toBe(expected);
		expect(
			await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)
		).toBe(true);
		await expect(page.getByLabel('密码', { exact: true })).toBeVisible();
	}
});
