import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError, authApi, type User } from '#lib/api/auth.js';
import { AccountSession, tokenKey } from './session.svelte.js';

const guest: User = { id: 'guest-id', username: null, display_name: '我的昵称', is_guest: true };
const member: User = { ...guest, username: 'music', is_guest: false };

beforeEach(() => {
	vi.restoreAllMocks();
	localStorage.clear();
});

describe('浏览器身份与会话', () => {
	it('从本地令牌恢复游客身份，升级后保留同一个令牌和 UUID', async () => {
		localStorage.setItem(tokenKey, 'guest-token');
		const me = vi
			.spyOn(authApi, 'me')
			.mockResolvedValueOnce({ user: guest })
			.mockResolvedValueOnce({ user: member });
		vi.spyOn(authApi, 'upgrade').mockResolvedValue();
		const session = new AccountSession();
		await session.restore();
		expect(session.user).toEqual(guest);
		await session.upgrade('music', 'password');
		expect(session.user?.id).toBe(guest.id);
		expect(session.user?.is_guest).toBe(false);
		expect(localStorage.getItem(tokenKey)).toBe('guest-token');
		expect(me).toHaveBeenLastCalledWith('guest-token');
	});
	it('断网时保留令牌，重试成功后恢复，不创建新游客', async () => {
		localStorage.setItem(tokenKey, 'saved-token');
		vi.spyOn(authApi, 'me')
			.mockRejectedValueOnce(new TypeError('offline'))
			.mockResolvedValueOnce({ user: guest });
		const create = vi.spyOn(authApi, 'guest');
		const session = new AccountSession();
		await session.restore();
		expect(session.status).toBe('error');
		expect(localStorage.getItem(tokenKey)).toBe('saved-token');
		await session.refresh();
		expect(session.user?.id).toBe(guest.id);
		expect(create).not.toHaveBeenCalled();
	});
	it('验证会话过期后移除本地身份', async () => {
		localStorage.setItem(tokenKey, 'expired');
		vi.spyOn(authApi, 'me').mockRejectedValue(new ApiError(401, 'unauthorized'));
		const session = new AccountSession();
		await session.restore();
		expect(session.user).toBeNull();
		expect(localStorage.getItem(tokenKey)).toBeNull();
		expect(session.notice).toBe('session.expiredGuest');
	});
	it('当前密码错误时重新检查会话，仍有效则保留登录', async () => {
		localStorage.setItem(tokenKey, 'valid');
		vi.spyOn(authApi, 'me').mockResolvedValue({ user: member });
		vi.spyOn(authApi, 'password').mockRejectedValue(new ApiError(401, 'unauthorized'));
		const session = new AccountSession();
		await session.restore();
		await expect(session.changePassword('wrong', 'newpassword')).rejects.toMatchObject({
			status: 401
		});
		expect(session.user?.username).toBe('music');
		expect(localStorage.getItem(tokenKey)).toBe('valid');
	});
	it('改密成功后清理当前身份并提示重新登录', async () => {
		localStorage.setItem(tokenKey, 'valid');
		vi.spyOn(authApi, 'me').mockResolvedValue({ user: member });
		vi.spyOn(authApi, 'password').mockResolvedValue();
		const session = new AccountSession();
		await session.restore();
		await session.changePassword('password', 'newpassword');
		expect(session.user).toBeNull();
		expect(localStorage.getItem(tokenKey)).toBeNull();
		expect(session.notice).toBe('session.passwordChanged');
	});
	it('其他标签页切换账号后，旧请求的 401 不得清除新账号令牌', async () => {
		localStorage.setItem(tokenKey, 'old');
		let rejectOld!: (error: Error) => void;
		vi.spyOn(authApi, 'me')
			.mockImplementationOnce(
				() =>
					new Promise((_, reject) => {
						rejectOld = reject;
					})
			)
			.mockResolvedValueOnce({ user: member });
		const session = new AccountSession();
		const oldRequest = session.restore();
		localStorage.setItem(tokenKey, 'new');
		await session.restore();
		rejectOld(new ApiError(401, 'unauthorized'));
		await oldRequest;
		expect(session.user?.username).toBe('music');
		expect(localStorage.getItem(tokenKey)).toBe('new');
	});
});
