<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import type { MessageState } from '#lib/i18n/types.js';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { account } from '#lib/auth/session.svelte.js';
	import { displayNameError, errorMessage, passwordError } from '#lib/api/auth.js';

	let { mode }: { mode: 'login' | 'register' } = $props();
	let username = $state('');
	let password = $state('');
	let confirmation = $state('');
	let displayName = $state('');
	let busy = $state(false);
	let error = $state<MessageState>('');
	let abandonGuest = $state(false);
	const registering = $derived(mode === 'register');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		error = registering ? passwordError(password) || displayNameError(displayName) : '';
		if (registering && password !== confirmation) error = 'error.passwordMismatch';
		if (error) return;
		busy = true;
		try {
			await account.enter(mode, username, password, displayName);
			password = confirmation = '';
			await goto(resolve('/settings'));
		} catch (cause) {
			error = cause instanceof Error ? errorMessage(cause) : 'error.unexpected';
		} finally {
			busy = false;
		}
	}
</script>

{#if (account.status === 'ready' && !account.user) || account.user?.is_guest}
	<section class="account-panel max-w-lg">
		{#if account.user?.is_guest}
			<p class="mb-4">
				{$t('account.guestSwitchHelp')}
				<a href={resolve('/settings')}>{$t('account.upgradeLink')}</a>
			</p>
		{/if}
		<form
			class="grid gap-5"
			onsubmit={submit}
			aria-label={registering ? $t('account.registerForm') : $t('account.loginForm')}
		>
			<fieldset class="grid gap-5" disabled={busy}>
				<label class="label">
					<span>{$t('account.username')}</span>
					<input
						class="input"
						name="username"
						bind:value={username}
						autocomplete="username"
						autocapitalize="none"
						spellcheck="false"
						required
						pattern={registering ? '[A-Za-z0-9_]{3,32}' : undefined}
						maxlength="32"
						aria-describedby={registering ? 'username-help' : undefined}
					/>
				</label>
				{#if registering}
					<p class="muted text-sm" id="username-help">{$t('account.usernameHelp')}</p>
					<label class="label"
						><span>{$t('account.displayNameOptional')}</span><input
							class="input"
							name="display_name"
							bind:value={displayName}
							autocomplete="nickname"
						/></label
					>
				{/if}
				<label class="label">
					<span>{$t('account.password')}</span>
					<input
						class="input"
						type="password"
						name="password"
						bind:value={password}
						autocomplete={registering ? 'new-password' : 'current-password'}
						required
						aria-describedby={registering ? 'password-help' : undefined}
					/>
				</label>
				{#if registering}
					<p class="muted text-sm" id="password-help">{$t('account.passwordHelp')}</p>
					<label class="label"
						><span>{$t('account.confirmPassword')}</span><input
							class="input"
							type="password"
							bind:value={confirmation}
							autocomplete="new-password"
							required
						/></label
					>
				{/if}
				{#if account.user?.is_guest}
					<label class="flex items-start gap-3"
						><input
							class="checkbox mt-1"
							type="checkbox"
							bind:checked={abandonGuest}
							required
						/><span>{$t('account.switchConfirm')}</span></label
					>
				{/if}
				{#if error}<p role="alert">{$t(error)}</p>{/if}
				<button class="btn primary" type="submit"
					>{busy
						? $t('account.submitting')
						: registering
							? $t('account.registerSubmit')
							: $t('account.login')}</button
				>
			</fieldset>
		</form>
		<p class="mt-5 text-sm">
			{#if registering}{$t('account.haveAccount')}
				<a href={resolve('/login')}>{$t('account.goLogin')}</a>{:else}{$t('account.noAccount')}
				<a href={resolve('/register')}>{$t('account.create')}</a>{/if}
		</p>
	</section>
{:else if account.user}
	<p>
		{$t('account.signedIn', {
			name: account.user.display_name || account.user.username || $t('account.guest')
		})} <a href={resolve('/settings')}>{$t('account.manageOrLogout')}</a>
	</p>
{/if}
