import createClient from 'openapi-fetch';
import type { paths } from './schema';

export class ApiError extends Error {
	constructor(
		public status: number,
		public code: string
	) {
		super(code);
	}
}

export const client = createClient<paths>({
	baseUrl: '/api/v1',
	cache: 'no-store',
	credentials: 'omit'
});

client.use({
	async onResponse({ response }) {
		if (response.ok) return;
		const body: unknown = await response.json().catch(() => null);
		const code =
			body !== null && typeof body === 'object' && 'error' in body && typeof body.error === 'string'
				? body.error
				: 'unknown';
		throw new ApiError(response.status, code);
	}
});
