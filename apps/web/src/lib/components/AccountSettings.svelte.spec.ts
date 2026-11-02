import { beforeEach, describe, expect, it, vi } from 'vitest';
import { page } from 'vitest/browser';
import { render } from 'vitest-browser-svelte';
import { ApiError, authApi, type User } from '#lib/api/auth.js';
import { account, tokenKey } from '#lib/auth/session.svelte.js';
import { setLanguage } from '#lib/i18n/index.js';
import AccountSettings from './AccountSettings.svelte';

const member: User = {
	id: 'account-id',
	username: 'saved_username',
	display_name: 'Saved name',
	is_guest: false
};

beforeEach(() => {
	vi.restoreAllMocks();
	localStorage.clear();
	localStorage.setItem(tokenKey, 'valid-token');
	setLanguage('en');
});

// Keep the user prop reactive, as the settings route does when /me returns.
async function renderSettings() {
	await account.restore();
	return render(AccountSettings, {
		get user() {
			return account.user!;
		}
	});
}

describe('账号设置的独立表单草稿', () => {
	it('当前密码错误后重新验证会话，保留未保存的用户名和显示名', async () => {
		const me = vi.spyOn(authApi, 'me').mockImplementation(async () => ({ user: { ...member } }));
		vi.spyOn(authApi, 'password').mockRejectedValue(new ApiError(401, 'unauthorized'));
		const profile = vi.spyOn(authApi, 'profile');
		await renderSettings();
		await page.getByLabelText('Username', { exact: true }).fill('unsaved_username');
		await page.getByLabelText('Display name', { exact: true }).fill('Unsaved name');
		await page.getByLabelText('Current password', { exact: true }).fill('wrong-password');
		await page.getByLabelText('New password', { exact: true }).fill('new-password');
		await page.getByLabelText('Confirm new password', { exact: true }).fill('new-password');
		await page.getByRole('button', { name: 'Change password and log out' }).click();
		await expect.element(page.getByRole('alert')).toHaveTextContent('Authentication failed.');
		expect(me).toHaveBeenCalledTimes(2);
		await expect
			.element(page.getByLabelText('Username', { exact: true }))
			.toHaveValue('unsaved_username');
		await expect
			.element(page.getByLabelText('Display name', { exact: true }))
			.toHaveValue('Unsaved name');
		expect(localStorage.getItem(tokenKey)).toBe('valid-token');
		expect(profile).not.toHaveBeenCalled();
	});

	it('保存资料成功后，使用重新读取的服务端资料更新表单', async () => {
		vi.spyOn(authApi, 'me')
			.mockResolvedValueOnce({ user: { ...member } })
			.mockResolvedValueOnce({
				user: { ...member, username: 'server_username', display_name: 'Server name' }
			});
		const profile = vi.spyOn(authApi, 'profile').mockResolvedValue();
		await renderSettings();
		await page.getByLabelText('Username', { exact: true }).fill('requested_username');
		await page.getByLabelText('Display name', { exact: true }).fill('Requested name');
		await page.getByRole('button', { name: 'Save profile' }).click();
		await expect
			.element(page.getByRole('status'))
			.toHaveTextContent('Your profile has been saved.');
		expect(profile).toHaveBeenCalledWith('valid-token', {
			username: 'requested_username',
			display_name: 'Requested name'
		});
		await expect
			.element(page.getByLabelText('Username', { exact: true }))
			.toHaveValue('server_username');
		await expect
			.element(page.getByLabelText('Display name', { exact: true }))
			.toHaveValue('Server name');
	});

	it('游客升级后填入正式用户名，同时保留未保存的显示名', async () => {
		vi.spyOn(authApi, 'me')
			.mockResolvedValueOnce({ user: { ...member, username: null, is_guest: true } })
			.mockResolvedValueOnce({ user: { ...member, username: 'upgraded_username' } });
		vi.spyOn(authApi, 'upgrade').mockResolvedValue();
		await renderSettings();
		await page.getByLabelText('Display name', { exact: true }).fill('Unsaved guest name');
		await page.getByLabelText('Username', { exact: true }).fill('upgraded_username');
		await page.getByLabelText('New password', { exact: true }).fill('new-password');
		await page.getByLabelText('Confirm new password', { exact: true }).fill('new-password');
		await page.getByRole('button', { name: 'Upgrade to a registered account' }).click();
		await expect
			.element(page.getByRole('heading', { name: 'Personal account', exact: true }))
			.toBeVisible();
		await expect
			.element(page.getByLabelText('Username', { exact: true }))
			.toHaveValue('upgraded_username');
		await expect
			.element(page.getByLabelText('Display name', { exact: true }))
			.toHaveValue('Unsaved guest name');
	});
});
