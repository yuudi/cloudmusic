import type { components } from './schema';
import { MessageError, type MessageKey, type MessageState } from '#lib/i18n/types.js';

type Schemas = components['schemas'];
export type User = Schemas['User'];
export type Session = Schemas['Session'];

export class ApiError extends Error {
	constructor(
		public status: number,
		public code: string
	) {
		super(code);
	}
}

async function request<T>(
	path: string,
	method = 'GET',
	body?: unknown,
	token?: string
): Promise<T> {
	const response = await fetch(`/api/v1/auth/${path}`, {
		method,
		cache: 'no-store',
		credentials: 'omit',
		headers: {
			...(body === undefined ? {} : { 'Content-Type': 'application/json' }),
			...(token ? { Authorization: `Bearer ${token}` } : {})
		},
		body: body === undefined ? undefined : JSON.stringify(body)
	});
	if (!response.ok) {
		const error = await response.json().catch(() => ({}));
		throw new ApiError(response.status, error.error ?? 'unknown');
	}
	return response.status === 204 ? (undefined as T) : response.json();
}

export const authApi = {
	guest: () => request<Session>('guest', 'POST'),
	register: (body: Schemas['RegisterRequest']) => request<Session>('register', 'POST', body),
	login: (body: Schemas['LoginRequest']) => request<Session>('login', 'POST', body),
	me: (token: string) => request<Schemas['CurrentUser']>('me', 'GET', undefined, token),
	profile: (token: string, body: Schemas['UpdateProfileRequest']) =>
		request<void>('me', 'PATCH', body, token),
	upgrade: (token: string, body: Schemas['UpgradeRequest']) =>
		request<void>('upgrade', 'POST', body, token),
	password: (token: string, body: Schemas['ChangePasswordRequest']) =>
		request<void>('password', 'POST', body, token),
	logout: (token: string, all: boolean) =>
		request<void>(all ? 'logout-all' : 'logout', 'POST', undefined, token)
};

export function errorMessage(error: unknown): MessageKey {
	if (error instanceof ApiError) {
		if (error.status === 429) return 'error.rateLimited';
		if (error.code === 'username_taken') return 'error.usernameTaken';
		if (error.status === 401) return 'error.unauthorized';
		if (error.status === 400) return 'error.invalid';
		return 'error.server';
	}
	if (error instanceof TypeError) return 'error.network';
	return error instanceof MessageError ? error.key : 'error.unexpected';
}

export function passwordError(password: string): MessageState {
	const bytes = new TextEncoder().encode(password).length;
	return bytes >= 8 && bytes <= 1024 ? '' : 'error.passwordLength';
}

export function displayNameError(name: string): MessageState {
	return [...name].length <= 100 && !name.includes('\0') ? '' : 'error.displayName';
}
