import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, authApi, displayNameError, errorMessage, passwordError } from './auth.js';

afterEach(() => vi.unstubAllGlobals());

describe('账号 API', () => {
	it('使用 Bearer 令牌且不缓存身份响应，接受无内容响应', async () => {
		const fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
		vi.stubGlobal('fetch', fetch);
		await expect(authApi.profile('secret', { display_name: '' })).resolves.toBeUndefined();
		expect(fetch).toHaveBeenCalledWith(
			'/api/v1/auth/me',
			expect.objectContaining({
				method: 'PATCH',
				cache: 'no-store',
				credentials: 'omit',
				headers: { Authorization: 'Bearer secret', 'Content-Type': 'application/json' },
				body: '{"display_name":""}'
			})
		);
	});
	it('登录请求不发送已有账号令牌，保留密码中的空格', async () => {
		const fetch = vi.fn().mockResolvedValue(Response.json({ token: 'new' }));
		vi.stubGlobal('fetch', fetch);
		await authApi.login({ username: 'music', password: ' password ' });
		expect(fetch.mock.calls[0][1].headers).not.toHaveProperty('Authorization');
		expect(JSON.parse(fetch.mock.calls[0][1].body).password).toBe(' password ');
	});
	it('错误响应保留状态，非 JSON 故障也有可读的错误提示', async () => {
		vi.stubGlobal(
			'fetch',
			vi.fn().mockResolvedValue(new Response('gateway error', { status: 502 }))
		);
		await expect(authApi.guest()).rejects.toMatchObject({ status: 502 });
		expect(errorMessage(new ApiError(409, 'username_taken'))).toBe('error.usernameTaken');
		expect(errorMessage(new ApiError(429, 'rate_limited'))).toBe('error.rateLimited');
		expect(errorMessage(new TypeError('fetch failed'))).toBe('error.network');
	});
});

describe('输入规则', () => {
	it('密码按 UTF-8 字节长度检查，不按字符数检查', () => {
		expect(passwordError('中文密码')).toBe('');
		expect(passwordError('1234567')).not.toBe('');
		expect(passwordError('a'.repeat(1024))).toBe('');
		expect(passwordError('中'.repeat(342))).not.toBe('');
	});
	it('显示名按 Unicode 码点计数且允许清空', () => {
		expect(displayNameError('')).toBe('');
		expect(displayNameError('🎵'.repeat(100))).toBe('');
		expect(displayNameError('🎵'.repeat(101))).not.toBe('');
		expect(displayNameError('a\0b')).not.toBe('');
	});
});
