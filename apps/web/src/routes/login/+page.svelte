<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import AccountForm from '#lib/components/AccountForm.svelte';
</script>

<svelte:head><title>{$t('login.title')}</title></svelte:head>
<h1>{$t('account.login')}</h1>
<p class="muted mb-8">{$t('login.description')}</p>
<AccountForm mode="login" />
