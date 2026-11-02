<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import type { MessageState } from '#lib/i18n/types.js';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { account } from '#lib/auth/session.svelte.js';
	import { errorMessage } from '#lib/api/auth.js';
	let busy = $state(false);
	let error = $state<MessageState>('');
	async function guest() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			await account.enter('guest');
			await goto(resolve('/settings'));
		} catch (cause) {
			error = errorMessage(cause);
		} finally {
			busy = false;
		}
	}
</script>

<div class="grid gap-4">
	<div class="flex flex-wrap items-center gap-3">
		<a class="btn primary" href={resolve('/login')}>{$t('account.login')}</a>
		<a class="btn preset-outlined-surface-500" href={resolve('/register')}>{$t('account.create')}</a
		>
		<button class="btn" type="button" onclick={guest} disabled={busy}>
			{busy ? $t('account.creatingGuest') : $t('account.guestContinue')}
		</button>
	</div>
	<p class="muted text-sm">{$t('account.guestHelp')}</p>
	{#if error}<p role="alert">{$t(error)}</p>{/if}
</div>
