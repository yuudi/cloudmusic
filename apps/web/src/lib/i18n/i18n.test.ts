import { afterEach, describe, expect, it } from 'vitest';
import { get } from 'svelte/store';
import { addMessages, dictionary, getMessageFormatter } from 'svelte-i18n';
import { activeLocale, resolveLocale, setLanguage, t } from './index.js';
import en from './messages/en.json';
import zhCN from './messages/zh-CN.json';

afterEach(() => {
	addMessages('zh-CN', zhCN);
	setLanguage('en');
});

type Ast = ReturnType<ReturnType<typeof getMessageFormatter>['getAst']>;
function parameters(ast: Ast): string[] {
	const result = new Set<string>();
	for (const element of ast) {
		if ('value' in element && element.type !== 0 && element.type !== 8) result.add(element.value);
		if (element.type === 5 || element.type === 6) {
			for (const option of Object.values(element.options)) {
				for (const name of parameters(option.value)) result.add(name);
			}
		}
	}
	return [...result].sort();
}

describe('语言选择和英文回退', () => {
	it('先使用手动选择，再匹配浏览器语言，最后回退到英文', () => {
		expect(resolveLocale('en', ['zh-CN'])).toBe('en');
		expect(resolveLocale('zh-CN', ['en-US'])).toBe('zh-CN');
		expect(resolveLocale(null, ['fr-FR', 'zh-SG', 'en-US'])).toBe('zh-CN');
		expect(resolveLocale(null, ['en-GB', 'zh-CN'])).toBe('en');
		expect(resolveLocale('unsupported', ['zh-Hans'])).toBe('zh-CN');
		expect(resolveLocale(null, ['fr-FR'])).toBe('en');
		expect(resolveLocale(null, [])).toBe('en');
	});
	it('中文缺少单条翻译时使用英文，不显示消息键', () => {
		const partial: Record<string, string> = { ...zhCN };
		delete partial['nav.home'];
		dictionary.update((value) => ({ ...value, 'zh-CN': partial }));
		setLanguage('zh-CN');
		expect(get(t)('nav.home')).toBe('Home');
		expect(get(t)('nav.settings')).toBe('设置');
	});
	it('切换语言时重新格式化带参数的消息', () => {
		setLanguage('en');
		expect(get(t)('home.welcome', { name: '听众' })).toBe('Welcome, 听众.');
		setLanguage('zh-CN');
		expect(get(t)('home.welcome', { name: '听众' })).toBe('欢迎，听众。');
		setLanguage('invalid');
		expect(get(activeLocale)).toBe('zh-CN');
	});
});

describe('字典完整性', () => {
	it('所有语言具有相同的消息键、合法 ICU 语法和相同插值参数', () => {
		expect(Object.keys(zhCN).sort()).toEqual(Object.keys(en).sort());
		for (const key of Object.keys(en) as (keyof typeof en)[]) {
			expect(zhCN[key].trim(), key).not.toBe('');
			const source = getMessageFormatter(en[key], 'en');
			const translation = getMessageFormatter(zhCN[key], 'zh-CN');
			expect(parameters(translation.getAst()), key).toEqual(parameters(source.getAst()));
		}
	});
});
