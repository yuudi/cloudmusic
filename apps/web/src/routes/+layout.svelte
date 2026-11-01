<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { pwaInfo } from 'virtual:pwa-info';
	import PwaUpdate from '#lib/components/PwaUpdate.svelte';
	import '#lib/styles/global.scss';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();
</script>

<svelte:head>
	<meta name="description" content="连接自己的存储空间，随时聆听你的音乐。" />
	{#if pwaInfo}
		<link rel="manifest" href={pwaInfo.webManifest.href} />
	{/if}
</svelte:head>

<a class="skip-link" href="#main">跳到内容</a>
<div class="app-shell">
	<header>
		<a class="brand" href={resolve('/')}>Cloud Music</a>
		<nav aria-label="主导航">
			<a href={resolve('/')} aria-current={page.url.pathname === '/' ? 'page' : undefined}>首页</a>
			<a
				href={resolve('/settings')}
				aria-current={page.url.pathname === '/settings' ? 'page' : undefined}>设置</a
			>
		</nav>
	</header>
	<main id="main" tabindex="-1">{@render children()}</main>
</div>
<PwaUpdate />

<style lang="scss">
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
		gap: 0.5rem;

		a {
			padding: 0.75rem 1rem;
			border-radius: 0.75rem;
			text-decoration: none;

			&[aria-current='page'] {
				background: var(--accent-soft);
			}
		}
	}

	main {
		padding-block: clamp(2rem, 6vw, 5rem);
	}
</style>
