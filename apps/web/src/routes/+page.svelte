<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import { resolve } from '$app/paths';
	import { account } from '#lib/auth/session.svelte.js';
	import AccountEntry from '#lib/components/AccountEntry.svelte';
</script>

<svelte:head>
	<title>{$t('app.name')}</title>
</svelte:head>

<h1>{$t('home.heading')}</h1>
<p>{$t('home.description')}</p>
<p class="muted">{$t('home.comingSoon')}</p>

<div class="mt-8 max-w-xl">
	{#if account.status === 'ready'}
		{#if account.user}
			<p class="mb-4">
				{$t('home.welcome', {
					name: account.user.display_name || account.user.username || $t('account.guest')
				})}
			</p>
			<a class="btn preset-outlined-surface-500" href={resolve('/settings')}
				>{$t('account.manage')}</a
			>
		{:else}
			<AccountEntry />
		{/if}
	{/if}
</div>
