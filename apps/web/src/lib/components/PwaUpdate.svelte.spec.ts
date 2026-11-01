import type { RegisterSWOptions } from '@unplugin-pwa/core/types';
import { page } from 'vitest/browser';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import PwaUpdate from './PwaUpdate.svelte';

const { state, registerSW, applyUpdate } = vi.hoisted(() => {
	const state = { options: undefined as RegisterSWOptions | undefined };
	const applyUpdate = vi.fn<() => Promise<void>>();
	return {
		state,
		applyUpdate,
		registerSW: vi.fn((options: RegisterSWOptions) => {
			state.options = options;
			return applyUpdate;
		})
	};
});

vi.mock('virtual:pwa-register', () => ({ registerSW }));

beforeEach(() => {
	state.options = undefined;
	registerSW.mockClear();
	applyUpdate.mockReset().mockResolvedValue(undefined);
});

describe('PWA 更新生命周期', () => {
	it('发现新版本后等待用户确认，选择稍后不会激活', async () => {
		render(PwaUpdate);
		await expect.poll(() => registerSW.mock.calls.length).toBe(1);
		state.options?.onNeedRefresh?.();

		await expect.element(page.getByRole('complementary', { name: '应用更新' })).toBeVisible();
		expect(applyUpdate).not.toHaveBeenCalled();
		await page.getByRole('button', { name: '稍后' }).click();
		await expect
			.element(page.getByRole('complementary', { name: '应用更新' }))
			.not.toBeInTheDocument();
		expect(applyUpdate).not.toHaveBeenCalled();
	});

	it('确认后才激活，更新失败时保留页面并允许重试', async () => {
		applyUpdate.mockRejectedValueOnce(new Error('更新失败'));
		render(PwaUpdate);
		await expect.poll(() => registerSW.mock.calls.length).toBe(1);
		state.options?.onNeedRefresh?.();
		await page.getByRole('button', { name: '刷新更新' }).click();
		await expect.element(page.getByRole('alert')).toHaveTextContent('更新失败，请稍后重试。');
		expect(applyUpdate).toHaveBeenCalledOnce();

		await page.getByRole('button', { name: '刷新更新' }).click();
		expect(applyUpdate).toHaveBeenCalledTimes(2);
		await expect.element(page.getByRole('alert')).not.toBeInTheDocument();
	});
});
