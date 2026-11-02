import { addMessages, format, init, locale } from 'svelte-i18n';
import { derived } from 'svelte/store';
import en from './messages/en.json';
import zhCN from './messages/zh-CN.json';
import type { MessageKey } from './types.js';

export const localeKey = 'cloudmusic.locale';
export const supportedLocales = ['en', 'zh-CN'] as const;
export type Locale = (typeof supportedLocales)[number];

addMessages('en', en);
addMessages('zh-CN', zhCN);
init({ fallbackLocale: 'en', initialLocale: 'en' });

export const activeLocale = derived(locale, (value): Locale =>
	value === 'zh-CN' ? 'zh-CN' : 'en'
);
export const t = derived(
	format,
	(translate) => (key: MessageKey, values?: Record<string, string | number>) =>
		translate(key, { values })
);

function isLocale(value: string | null): value is Locale {
	return supportedLocales.some((supported) => supported === value);
}

export function resolveLocale(saved: string | null, languages: readonly string[]): Locale {
	if (isLocale(saved)) return saved;
	for (const language of languages) {
		if (/^zh(?:-|$)/i.test(language)) return 'zh-CN';
		if (/^en(?:-|$)/i.test(language)) return 'en';
	}
	return 'en';
}

function detectLocale(): Locale {
	let saved: string | null = null;
	try {
		saved = localStorage.getItem(localeKey);
	} catch {
		// Language switching remains usable when browser storage is blocked.
	}
	return resolveLocale(saved, navigator.languages);
}

export function initializeI18n() {
	if (typeof window !== 'undefined') locale.set(detectLocale());
}

export function setLanguage(value: string) {
	if (!isLocale(value)) return;
	locale.set(value);
	if (typeof window === 'undefined') return;
	try {
		localStorage.setItem(localeKey, value);
	} catch {
		// Keep the current document working even if the preference cannot be saved.
	}
}

export function mountI18n() {
	const unsubscribe = activeLocale.subscribe((value) => {
		document.documentElement.lang = value;
		document.documentElement.dir = 'ltr';
	});
	const sync = (event: StorageEvent) => {
		if (event.key === localeKey || event.key === null) initializeI18n();
	};
	const detect = () => initializeI18n();
	window.addEventListener('storage', sync);
	window.addEventListener('languagechange', detect);
	return () => {
		unsubscribe();
		window.removeEventListener('storage', sync);
		window.removeEventListener('languagechange', detect);
	};
}
