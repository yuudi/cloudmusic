import { afterAll, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	ApiError,
	authApi,
	displayNameError,
	errorMessage,
	passwordError,
	type Session
} from './auth.js';

const { fetchMock } = vi.hoisted(() => {
	const fetchMock = vi.fn<(request: Request) => Promise<Response>>();
	// Node requires absolute URLs; browsers resolve these against the document origin.
	class BrowserRequest extends Request {
		constructor(input: RequestInfo | URL, init?: RequestInit) {
			super(typeof input === 'string' ? new URL(input, 'https://cloudmusic.test') : input, init);
		}
	}
	vi.stubGlobal('Request', BrowserRequest);
	vi.stubGlobal('fetch', fetchMock);
	return { fetchMock };
});

beforeEach(() => {
	fetchMock.mockReset();
});
afterAll(() => vi.unstubAllGlobals());

const session: Session = {
	user: { id: 'account-id', username: 'music', display_name: '音乐', is_guest: false },
	token: 'new-token',
	token_type: 'Bearer',
	expires_in: 2592000
};

describe('账号 API', () => {
	it('使用 Bearer 令牌且不缓存身份响应，接受无内容响应', async () => {
		fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
		await expect(authApi.profile('secret', { display_name: '' })).resolves.toBeUndefined();
		const [request] = fetchMock.mock.calls[0];
		expect(request.url).toBe('https://cloudmusic.test/api/v1/auth/me');
		expect(request.method).toBe('PATCH');
		expect(request.cache).toBe('no-store');
		expect(request.credentials).toBe('omit');
		expect(request.headers.get('Authorization')).toBe('Bearer secret');
		expect(request.headers.get('Content-Type')).toBe('application/json');
		expect(await request.json()).toEqual({ display_name: '' });
	});
	it.each(['guest', 'register', 'login'] as const)(
		'%s 不沿用上次认证请求的令牌，返回完整会话',
		async (mode) => {
			fetchMock
				.mockResolvedValueOnce(Response.json({ user: session.user }))
				.mockResolvedValueOnce(Response.json(session));
			await authApi.me('old-token');
			const body = { username: 'music', password: ' password ', display_name: '音乐' };
			const enter = {
				guest: () => authApi.guest(),
				register: () => authApi.register(body),
				login: () => authApi.login({ username: body.username, password: body.password })
			};
			await expect(enter[mode]()).resolves.toEqual(session);
			const [request] = fetchMock.mock.calls[1];
			expect(request.url).toBe(`https://cloudmusic.test/api/v1/auth/${mode}`);
			expect(request.method).toBe('POST');
			expect(request.headers.has('Authorization')).toBe(false);
			if (mode === 'guest') {
				expect(request.body).toBeNull();
				expect(request.headers.has('Content-Type')).toBe(false);
			} else {
				expect(await request.json()).toEqual(
					mode === 'register' ? body : { username: 'music', password: ' password ' }
				);
			}
		}
	);
	it('当前身份请求返回用户且不发送请求体', async () => {
		fetchMock.mockResolvedValue(Response.json({ user: session.user }));
		await expect(authApi.me('secret')).resolves.toEqual({ user: session.user });
		const [request] = fetchMock.mock.calls[0];
		expect(request.method).toBe('GET');
		expect(request.headers.get('Authorization')).toBe('Bearer secret');
		expect(request.headers.has('Content-Type')).toBe(false);
		expect(request.body).toBeNull();
	});
	it.each([
		{
			path: 'upgrade',
			body: { username: 'music', password: 'password' },
			action: () => authApi.upgrade('secret', { username: 'music', password: 'password' })
		},
		{
			path: 'password',
			body: { old_password: 'old password', new_password: 'new password' },
			action: () =>
				authApi.password('secret', { old_password: 'old password', new_password: 'new password' })
		},
		{ path: 'logout', body: undefined, action: () => authApi.logout('secret', false) },
		{ path: 'logout-all', body: undefined, action: () => authApi.logout('secret', true) }
	])('$path 使用正确请求并返回 void', async ({ path, body, action }) => {
		fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
		await expect(action()).resolves.toBeUndefined();
		const [request] = fetchMock.mock.calls[0];
		expect(request.url).toBe(`https://cloudmusic.test/api/v1/auth/${path}`);
		expect(request.method).toBe('POST');
		expect(request.headers.get('Authorization')).toBe('Bearer secret');
		if (body === undefined) {
			expect(request.body).toBeNull();
			expect(request.headers.has('Content-Type')).toBe(false);
		} else expect(await request.json()).toEqual(body);
	});
	it.each([
		[400, 'invalid_request'],
		[401, 'unauthorized'],
		[409, 'username_taken'],
		[429, 'rate_limited'],
		[500, 'internal_server_error']
	])('HTTP %i 保留状态和后端错误码', async (status, code) => {
		fetchMock.mockResolvedValue(Response.json({ error: code }, { status }));
		const request = authApi.guest();
		await expect(request).rejects.toBeInstanceOf(ApiError);
		await expect(request).rejects.toMatchObject({ status, code });
	});
	it.each(['gateway error', '', 'null', '{"error":123}'])(
		'网关或异常错误正文 %j 使用通用错误码',
		async (body) => {
			fetchMock.mockResolvedValue(new Response(body, { status: 502 }));
			await expect(authApi.guest()).rejects.toMatchObject({ status: 502, code: 'unknown' });
		}
	);
	it('网络异常原样传递，供调用方区分断网与 HTTP 错误', async () => {
		const error = new TypeError('fetch failed');
		fetchMock.mockRejectedValue(error);
		await expect(authApi.guest()).rejects.toBe(error);
		expect(errorMessage(error)).toBe('error.network');
	});
	it('需要 JSON 正文的成功请求不会将空响应当成有效身份', async () => {
		fetchMock.mockResolvedValue(new Response(null, { status: 200 }));
		await expect(authApi.me('secret')).rejects.toThrow('Missing API response body');
	});
	it('错误响应仍映射为原有消息键', () => {
		expect(errorMessage(new ApiError(409, 'username_taken'))).toBe('error.usernameTaken');
		expect(errorMessage(new ApiError(429, 'rate_limited'))).toBe('error.rateLimited');
		expect(errorMessage(new ApiError(401, 'unauthorized'))).toBe('error.unauthorized');
		expect(errorMessage(new ApiError(502, 'unknown'))).toBe('error.server');
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
