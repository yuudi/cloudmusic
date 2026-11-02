import { initializeI18n } from '#lib/i18n/index.js';

export const ssr = false;

export const load = () => {
	initializeI18n();
	return {};
};
