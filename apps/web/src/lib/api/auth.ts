import type { components } from './schema';
import { ApiError, client } from './client.js';
import { MessageError, type MessageKey, type MessageState } from '#lib/i18n/types.js';

export { ApiError } from './client.js';

type Schemas = components['schemas'];
export type User = Schemas['User'];
export type Session = Schemas['Session'];

async function responseData<T>(request: Promise<{ data?: T }>): Promise<T> {
	const { data } = await request;
	if (data === undefined) throw new Error('Missing API response body');
	return data;
}

const bearer = (token: string) => ({ Authorization: `Bearer ${token}` });

export const authApi = {
	guest: () => responseData(client.POST('/auth/guest')),
	register: (body: Schemas['RegisterRequest']) =>
		responseData(client.POST('/auth/register', { body })),
	login: (body: Schemas['LoginRequest']) => responseData(client.POST('/auth/login', { body })),
	me: (token: string) => responseData(client.GET('/auth/me', { headers: bearer(token) })),
	profile: async (token: string, body: Schemas['UpdateProfileRequest']) => {
		await client.PATCH('/auth/me', { body, headers: bearer(token) });
	},
	upgrade: async (token: string, body: Schemas['UpgradeRequest']) => {
		await client.POST('/auth/upgrade', { body, headers: bearer(token) });
	},
	password: async (token: string, body: Schemas['ChangePasswordRequest']) => {
		await client.POST('/auth/password', { body, headers: bearer(token) });
	},
	logout: async (token: string, all: boolean) => {
		await client.POST(all ? '/auth/logout-all' : '/auth/logout', { headers: bearer(token) });
	}
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
