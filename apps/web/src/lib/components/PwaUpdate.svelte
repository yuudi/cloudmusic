<script lang="ts">
	import { onMount } from 'svelte';
	import UpdatePrompt from './UpdatePrompt.svelte';
	import type { MessageState } from '#lib/i18n/types.js';

	let needsUpdate = $state(false);
	let updating = $state(false);
	let error = $state<MessageState>('');
	let applyUpdate: (() => Promise<void>) | undefined;

	onMount(() => {
		let disposed = false;
		let registration: ServiceWorkerRegistration | undefined;
		const checkForUpdate = () => {
			if (document.visibilityState === 'visible' && navigator.onLine) {
				void registration?.update().catch((cause: unknown) => {
					console.error('检查应用更新失败', cause);
				});
			}
		};

		void import('virtual:pwa-register')
			.then(({ registerSW }) => {
				if (disposed) return;
				applyUpdate = registerSW({
					immediate: true,
					onNeedRefresh: () => (needsUpdate = true),
					onRegisteredSW: (_url, value) => (registration = value),
					onRegisterError: (cause: unknown) => console.error('注册离线应用失败', cause)
				});
			})
			.catch((cause: unknown) => console.error('加载离线应用失败', cause));

		document.addEventListener('visibilitychange', checkForUpdate);
		window.addEventListener('online', checkForUpdate);
		return () => {
			disposed = true;
			document.removeEventListener('visibilitychange', checkForUpdate);
			window.removeEventListener('online', checkForUpdate);
		};
	});

	async function update() {
		if (!applyUpdate) return;
		updating = true;
		error = '';
		try {
			await applyUpdate();
		} catch {
			error = 'pwa.failed';
		} finally {
			updating = false;
		}
	}
</script>

{#if needsUpdate}
	<UpdatePrompt {updating} {error} onUpdate={update} onDismiss={() => (needsUpdate = false)} />
{/if}
