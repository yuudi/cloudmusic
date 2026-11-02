<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import type { MessageState } from '#lib/i18n/types.js';
	let {
		updating = false,
		error = '',
		onUpdate,
		onDismiss
	}: {
		updating?: boolean;
		error?: MessageState;
		onUpdate: () => void;
		onDismiss: () => void;
	} = $props();
</script>

<aside aria-label={$t('pwa.label')} aria-live="polite">
	<p>{$t('pwa.ready')}</p>
	<p class="muted">{$t('pwa.interrupt')}</p>
	{#if error}<p role="alert">{$t(error)}</p>{/if}
	<div class="actions">
		<button type="button" onclick={onUpdate} disabled={updating}>
			{updating ? $t('pwa.updating') : $t('pwa.refresh')}
		</button>
		<button type="button" onclick={onDismiss} disabled={updating}>{$t('pwa.later')}</button>
	</div>
</aside>

<style>
	aside {
		position: fixed;
		right: 1rem;
		bottom: calc(1rem + env(safe-area-inset-bottom));
		left: 1rem;
		max-width: 28rem;
		margin-left: auto;
		padding: 1.25rem;
		border: 1px solid var(--border);
		border-radius: 1rem;
		background: var(--surface);
		box-shadow: 0 0.5rem 2rem #00000020;
	}

	p {
		margin: 0 0 0.75rem;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
	}
</style>
