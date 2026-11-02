<script lang="ts">
	import { t } from '#lib/i18n/index.js';
	import type { MessageState } from '#lib/i18n/types.js';
	import { account } from '#lib/auth/session.svelte.js';
	import { displayNameError, errorMessage, passwordError, type User } from '#lib/api/auth.js';
	let { user }: { user: User } = $props();
	let displayName = $state('');
	let username = $state('');
	let upgradeName = $state('');
	let newPassword = $state('');
	let oldPassword = $state('');
	let confirmation = $state('');
	let busy = $state(false);
	let error = $state<MessageState>('');
	let message = $state<MessageState>('');
	let profileUserId: string | undefined;
	$effect(() => {
		// A session check can return a new user object without changing accounts.
		if (profileUserId === user.id) return;
		profileUserId = user.id;
		displayName = user.display_name;
		username = user.username ?? '';
	});

	async function run(action: () => Promise<void>, success: MessageState) {
		if (busy) return;
		busy = true;
		error = message = '';
		try {
			await action();
			message = success;
		} catch (cause) {
			error = errorMessage(cause);
		} finally {
			busy = false;
		}
	}

	function profile(event: SubmitEvent) {
		event.preventDefault();
		error = displayNameError(displayName);
		if (!error)
			void run(async () => {
				const id = user.id;
				await account.profile(displayName, user.is_guest ? undefined : username);
				if (account.status === 'ready' && account.user?.id === id) {
					displayName = account.user.display_name;
					username = account.user.username ?? '';
				}
			}, 'settings.profileSaved');
	}

	function password(event: SubmitEvent) {
		event.preventDefault();
		error = passwordError(newPassword);
		if (newPassword !== confirmation) error = 'error.passwordMismatch';
		if (error) return;
		void run(
			async () => {
				if (user.is_guest) {
					const id = user.id;
					await account.upgrade(upgradeName, newPassword);
					if (account.status === 'ready' && account.user?.id === id) {
						username = account.user.username ?? '';
					}
				} else await account.changePassword(oldPassword, newPassword);
				oldPassword = newPassword = confirmation = '';
			},
			user.is_guest ? 'settings.upgraded' : 'settings.passwordChanged'
		);
	}

	function logout(all: boolean) {
		if (user.is_guest && !window.confirm($t('settings.logoutGuestConfirm'))) return;
		if (all && !user.is_guest && !window.confirm($t('settings.logoutAllConfirm'))) return;
		void run(() => account.logout(all), '');
	}
</script>

<div class="grid max-w-2xl gap-6" aria-busy={busy}>
	<section class="account-panel">
		<h2 class="mb-3 text-xl font-semibold">
			{user.is_guest ? $t('settings.guestAccount') : $t('settings.personalAccount')}
		</h2>
		<p class="muted mb-5">
			{user.is_guest ? $t('settings.guestHelp') : $t('settings.personalHelp')}
		</p>
		<form aria-label={$t('settings.profile')} onsubmit={profile}>
			<fieldset class="grid gap-5" disabled={busy}>
				{#if !user.is_guest}
					<label class="label"
						><span>{$t('account.username')}</span><input
							class="input"
							name="username"
							bind:value={username}
							autocomplete="username"
							autocapitalize="none"
							spellcheck="false"
							required
							pattern={'[A-Za-z0-9_]{3,32}'}
							maxlength="32"
							aria-describedby="profile-username-help"
						/></label
					>
					<p class="muted text-sm" id="profile-username-help">{$t('settings.usernameHelp')}</p>
				{/if}
				<label class="label"
					><span>{$t('account.displayName')}</span><input
						class="input"
						name="display_name"
						bind:value={displayName}
						autocomplete="nickname"
						aria-describedby="display-help"
					/></label
				>
				<p class="muted text-sm" id="display-help">{$t('settings.displayHelp')}</p>
				<button class="btn justify-self-start" type="submit">{$t('settings.saveProfile')}</button>
			</fieldset>
		</form>
	</section>

	<section class="account-panel" id="password">
		<h2 class="mb-3 text-xl font-semibold">
			{user.is_guest ? $t('settings.upgradeHeading') : $t('settings.changePassword')}
		</h2>
		<p class="muted mb-5">
			{user.is_guest ? $t('settings.upgradeHelp') : $t('settings.passwordHelp')}
		</p>
		<form
			aria-label={user.is_guest ? $t('settings.upgradeForm') : $t('settings.changePassword')}
			onsubmit={password}
		>
			<fieldset class="grid gap-5" disabled={busy}>
				{#if user.is_guest}
					<label class="label"
						><span>{$t('account.username')}</span><input
							class="input"
							bind:value={upgradeName}
							autocomplete="username"
							autocapitalize="none"
							spellcheck="false"
							required
							pattern={'[A-Za-z0-9_]{3,32}'}
							maxlength="32"
							aria-describedby="upgrade-username-help"
						/></label
					>
					<p class="muted text-sm" id="upgrade-username-help">{$t('settings.upgradeNameHelp')}</p>
				{:else}
					<label class="label"
						><span>{$t('settings.currentPassword')}</span><input
							class="input"
							type="password"
							bind:value={oldPassword}
							autocomplete="current-password"
							required
						/></label
					>
				{/if}
				<label class="label"
					><span>{$t('settings.newPassword')}</span><input
						class="input"
						type="password"
						bind:value={newPassword}
						autocomplete="new-password"
						required
						aria-describedby="new-password-help"
					/></label
				>
				<p class="muted text-sm" id="new-password-help">{$t('settings.newPasswordHelp')}</p>
				<label class="label"
					><span>{$t('settings.confirmNewPassword')}</span><input
						class="input"
						type="password"
						bind:value={confirmation}
						autocomplete="new-password"
						required
					/></label
				>
				<button class="btn primary justify-self-start" type="submit"
					>{user.is_guest ? $t('settings.upgradeSubmit') : $t('settings.passwordSubmit')}</button
				>
			</fieldset>
		</form>
	</section>

	<section class="account-panel">
		<h2 class="mb-3 text-xl font-semibold">{$t('settings.storage')}</h2>
		<p class="muted">
			{#if user.is_guest}{$t('settings.guestStorage')}
				<a href="#password">{$t('account.upgradeLink')}</a>
			{:else}{$t('settings.memberStorage')}{/if}
		</p>
	</section>
	<section class="account-panel">
		<h2 class="mb-3 text-xl font-semibold">{$t('settings.sessions')}</h2>
		<div class="flex flex-wrap gap-3">
			<button class="btn" type="button" disabled={busy} onclick={() => logout(false)}
				>{$t('settings.logout')}</button
			>
			{#if !user.is_guest}<button
					class="btn"
					type="button"
					disabled={busy}
					onclick={() => logout(true)}>{$t('settings.logoutAll')}</button
				>{/if}
		</div>
	</section>
	{#if busy}<p role="status">{$t('settings.processing')}</p>{/if}
	{#if error}<p role="alert">{$t(error)}</p>{/if}
	{#if message}<p role="status">{$t(message)}</p>{/if}
</div>
