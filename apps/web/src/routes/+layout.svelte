<script lang="ts">
	import { t, mountI18n } from '#lib/i18n/index.js';
	import { onMount } from 'svelte';
	import { account, tokenKey } from '#lib/auth/session.svelte.js';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { pwaInfo } from 'virtual:pwa-info';
	import PwaUpdate from '#lib/components/PwaUpdate.svelte';
	import LanguagePicker from '#lib/components/LanguagePicker.svelte';
	import '#lib/styles/global.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
	onMount(() => {
		const unmountI18n = mountI18n();
		void account.restore();
		const sync = (event: StorageEvent) => {
			if (event.key === tokenKey || event.key === null) void account.restore();
		};
		window.addEventListener('storage', sync);
		return () => {
			unmountI18n();
			window.removeEventListener('storage', sync);
		};
	});
</script>

<svelte:head>
	<meta name="description" content={$t('app.description')} />
	{#if pwaInfo}
		<link rel="manifest" href={pwaInfo.webManifest.href} />
	{/if}
</svelte:head>

<a class="skip-link" href="#main">{$t('nav.skip')}</a>
<div class="app-shell">
	<header>
		<a class="brand" href={resolve('/')}>{$t('app.name')}</a>
		<nav aria-label={$t('nav.main')}>
			<a href={resolve('/')} aria-current={page.url.pathname === '/' ? 'page' : undefined}
				>{$t('nav.home')}</a
			>
			<a
				href={resolve('/settings')}
				aria-current={page.url.pathname === '/settings' ? 'page' : undefined}
				>{$t('nav.settings')}</a
			>
			{#if account.user}
				<span class="muted max-w-40 truncate self-center text-sm"
					>{account.user.display_name || account.user.username || $t('account.guest')}</span
				>
			{:else if account.status === 'ready'}
				<a href={resolve('/login')}>{$t('account.login')}</a>
			{/if}
		</nav>
		<LanguagePicker />
	</header>
	<main id="main" tabindex="-1">
		{#if account.notice}<p class="mb-5" role="status">{$t(account.notice)}</p>{/if}
		{#if account.persistenceWarning}<p class="mb-5" role="alert">
				{$t(account.persistenceWarning)}
			</p>{/if}
		{#if account.status === 'loading'}
			<p class="muted mb-5" role="status">{$t('session.restoring')}</p>
		{:else if account.status === 'error'}
			<div class="account-panel mb-5" role="alert">
				<p class="mb-3">{$t('session.retryHelp')}</p>
				<button type="button" onclick={() => account.refresh()}>{$t('session.retry')}</button>
			</div>
		{/if}
		{@render children()}
	</main>
</div>
<PwaUpdate />

<style>
	.app-shell {
		width: min(100% - 2rem, 64rem);
		margin-inline: auto;
	}

	header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding-block: 1.5rem;
		border-bottom: 1px solid var(--border);
	}

	.brand {
		font-weight: 700;
		text-decoration: none;
	}

	nav {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	nav a {
		padding: 0.75rem 1rem;
		border-radius: 0.75rem;
		text-decoration: none;
	}

	nav a[aria-current='page'] {
		background: var(--accent-soft);
	}

	main {
		padding-block: clamp(2rem, 6vw, 5rem);
	}
</style>
