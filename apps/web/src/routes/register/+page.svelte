<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import AccountForm from '#lib/components/AccountForm.svelte';
</script>

<svelte:head><title>{$t('register.title')}</title></svelte:head>
<h1>{$t('account.create')}</h1>
<p class="muted mb-8">{$t('register.description')}</p>
<AccountForm mode="register" />
