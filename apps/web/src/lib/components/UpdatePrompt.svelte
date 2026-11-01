<script lang="ts">
	let {
		updating = false,
		error = '',
		onUpdate,
		onDismiss
	}: {
		updating?: boolean;
		error?: string;
		onUpdate: () => void;
		onDismiss: () => void;
	} = $props();
</script>

<aside aria-label="应用更新" aria-live="polite">
	<p>新版本已就绪，刷新后即可使用。</p>
	<p class="muted">刷新会中断当前播放，你可以稍后再更新。</p>
	{#if error}<p role="alert">{error}</p>{/if}
	<div class="actions">
		<button type="button" onclick={onUpdate} disabled={updating}>
			{updating ? '正在更新…' : '刷新更新'}
		</button>
		<button type="button" onclick={onDismiss} disabled={updating}>稍后</button>
	</div>
</aside>

<style lang="scss">
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
		box-shadow: 0 0.5rem 2rem #172d2720;
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
