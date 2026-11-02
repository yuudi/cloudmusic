<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import { account } from '#lib/auth/session.svelte.js';
	import AccountSettings from '#lib/components/AccountSettings.svelte';
	import AccountEntry from '#lib/components/AccountEntry.svelte';
</script>

<svelte:head>
	<title>{$t('settings.title')}</title>
</svelte:head>

<h1>{$t('nav.settings')}</h1>
<p class="muted mb-8">{$t('settings.description')}</p>
{#if account.status === 'ready'}
	{#if account.user}
		{#key account.user.id}<AccountSettings user={account.user} />{/key}
	{:else}
		<AccountEntry />
	{/if}
{/if}
