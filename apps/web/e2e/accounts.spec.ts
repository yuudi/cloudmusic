import { expect, test, type Page } from '@playwright/test';

test.skip(!process.env.CLOUDMUSIC_ACCOUNT_E2E, '通过 tests/e2e/run.sh 启动独立后端后运行');

async function identity(page: Page) {
	const token = await page.evaluate(() => localStorage.getItem('cloudmusic.session'));
	const response = await page.request.get('/api/v1/auth/me', {
		headers: { Authorization: `Bearer ${token}` }
	});
	expect(response.ok()).toBe(true);
	return (await response.json()).user;
}

async function login(page: Page, username: string, password: string) {
	await page.goto('/login');
	await page.getByLabel('用户名', { exact: true }).fill(username);
	await page.getByLabel('密码', { exact: true }).fill(password);
	await page.getByRole('button', { name: '登录', exact: true }).click();
	await expect(page.getByRole('heading', { name: '个人账号', exact: true })).toBeVisible();
}

test('真实 API：游客升级、资料隔离、多会话退出和修改密码', async ({ page, browser }) => {
	test.setTimeout(90_000);
	const suffix = Date.now().toString(36);
	const username = `music_${suffix}`;
	const renamed = `renamed_${suffix}`;
	const password = 'Initial password 123';
	const updatedPassword = 'Updated password 456';
	const secondContext = await browser.newContext({
		baseURL: 'http://127.0.0.1:4173',
		locale: 'zh-CN'
	});
	try {
		await page.goto('/');
		await page.getByRole('button', { name: '以游客身份继续' }).click();
		await expect(page.getByRole('heading', { name: '游客账号', exact: true })).toBeVisible();
		const guest = await identity(page);
		const guestToken = await page.evaluate(() => localStorage.getItem('cloudmusic.session'));
		const profile = page.getByRole('form', { name: '个人资料' });
		await profile.getByLabel('显示名', { exact: true }).fill('我的音乐');
		await profile.getByRole('button', { name: '保存资料' }).click();
		await expect(page.getByText('个人资料已保存。')).toBeVisible();
		await page.reload();
		await expect(page.getByLabel('显示名', { exact: true })).toHaveValue('我的音乐');
		expect((await identity(page)).id).toBe(guest.id);

		const upgrade = page.getByRole('form', { name: '升级账号' });
		await upgrade.getByLabel('用户名', { exact: true }).fill(username);
		await upgrade.getByLabel('新密码', { exact: true }).fill(password);
		await upgrade.getByLabel('确认新密码', { exact: true }).fill(password);
		await upgrade.getByRole('button', { name: '升级为正式账号' }).click();
		await expect(page.getByRole('heading', { name: '个人账号', exact: true })).toBeVisible();
		expect((await identity(page)).id).toBe(guest.id);
		expect(await page.evaluate(() => localStorage.getItem('cloudmusic.session'))).toBe(guestToken);
		await profile.getByLabel('用户名', { exact: true }).fill(renamed);
		await profile.getByRole('button', { name: '保存资料' }).click();
		await expect(page.getByText('个人资料已保存。')).toBeVisible();

		// A separate user can only edit their own profile.
		const otherPage = await secondContext.newPage();
		await otherPage.goto('/register');
		await otherPage.getByLabel('用户名', { exact: true }).fill(`other_${suffix}`);
		await otherPage.getByLabel('密码', { exact: true }).fill(password);
		await otherPage.getByLabel('确认密码', { exact: true }).fill(password);
		await otherPage.getByRole('button', { name: '注册并登录' }).click();
		await expect(otherPage.getByRole('heading', { name: '个人账号', exact: true })).toBeVisible();
		const other = await identity(otherPage);
		expect(other.id).not.toBe(guest.id);
		await otherPage.getByLabel('显示名', { exact: true }).fill('另一个人的昵称');
		await otherPage.getByRole('button', { name: '保存资料' }).click();
		await expect(otherPage.getByText('个人资料已保存。')).toBeVisible();
		expect((await identity(page)).display_name).toBe('我的音乐');
		await otherPage.getByRole('button', { name: '退出当前会话' }).click();
		await expect(otherPage.getByRole('button', { name: '以游客身份继续' })).toBeVisible();

		await login(otherPage, renamed.toUpperCase(), password);
		await page.getByRole('button', { name: '退出当前会话' }).click();
		await expect(page.getByRole('button', { name: '以游客身份继续' })).toBeVisible();
		expect((await identity(otherPage)).id).toBe(guest.id);
		await login(page, renamed, password);

		const change = page.getByRole('form', { name: '修改密码' });
		await change.getByLabel('当前密码', { exact: true }).fill('wrong password');
		await change.getByLabel('新密码', { exact: true }).fill(updatedPassword);
		await change.getByLabel('确认新密码', { exact: true }).fill(updatedPassword);
		await change.getByRole('button', { name: '修改密码并退出' }).click();
		await expect(page.getByRole('alert')).toContainText('身份验证失败');
		expect((await identity(page)).id).toBe(guest.id);
		await change.getByLabel('当前密码', { exact: true }).fill(password);
		await change.getByRole('button', { name: '修改密码并退出' }).click();
		await expect(
			page.getByText('密码已修改，所有设备的会话已退出，请使用新密码登录。')
		).toBeVisible();
		await otherPage.reload();
		await expect(otherPage.getByRole('button', { name: '以游客身份继续' })).toBeVisible();
		await login(page, renamed, updatedPassword);
		await login(otherPage, renamed, updatedPassword);
		page.once('dialog', (dialog) => dialog.accept());
		await page.getByRole('button', { name: '退出所有设备' }).click();
		await expect(page.getByText('已退出所有设备。', { exact: true })).toBeVisible();
		await otherPage.reload();
		await expect(otherPage.getByRole('button', { name: '以游客身份继续' })).toBeVisible();
	} finally {
		await secondContext.close();
	}
});
