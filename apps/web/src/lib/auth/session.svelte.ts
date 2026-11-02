import { ApiError, authApi, type Session, type User } from '#lib/api/auth.js';
import { MessageError, type MessageState } from '#lib/i18n/types.js';

export const tokenKey = 'cloudmusic.session';

export class AccountSession {
	user = $state<User | null>(null);
	status = $state<'loading' | 'ready' | 'error'>('loading');
	notice = $state<MessageState>('');
	persistenceWarning = $state<MessageState>('');
	private token: string | null = null;
	private revision = 0;

	private persist(token: string | null) {
		try {
			if (token) localStorage.setItem(tokenKey, token);
			else localStorage.removeItem(tokenKey);
			this.persistenceWarning = '';
		} catch {
			this.persistenceWarning = 'session.storageWrite';
		}
	}

	async restore() {
		this.status = 'loading';
		this.user = null;
		this.notice = '';
		try {
			this.token = localStorage.getItem(tokenKey);
		} catch {
			this.persistenceWarning = 'session.storageRead';
		}
		await this.refresh();
	}

	async refresh() {
		const revision = ++this.revision;
		const token = this.token;
		if (!token) {
			this.status = 'ready';
			return;
		}
		try {
			const { user } = await authApi.me(token);
			if (revision !== this.revision) return;
			this.user = user;
			this.status = 'ready';
		} catch (error) {
			if (revision !== this.revision) return;
			if (error instanceof ApiError && error.status === 401) {
				this.clear('session.expiredGuest');
			} else {
				this.status = 'error';
			}
		}
	}

	private clear(notice: MessageState = '') {
		this.revision++;
		this.token = null;
		this.user = null;
		this.status = 'ready';
		this.notice = notice;
		this.persist(null);
	}

	async enter(
		mode: 'login' | 'register' | 'guest',
		username = '',
		password = '',
		displayName = ''
	) {
		const revision = this.revision;
		let session: Session;
		if (mode === 'guest') session = await authApi.guest();
		else if (mode === 'login') session = await authApi.login({ username, password });
		else session = await authApi.register({ username, password, display_name: displayName });
		if (revision !== this.revision) throw new MessageError('session.changedElsewhere');
		this.revision++;
		this.token = session.token;
		this.user = session.user;
		this.status = 'ready';
		this.notice = '';
		this.persist(session.token);
	}

	private async authenticated(action: (token: string) => Promise<void>, ambiguous401 = false) {
		const token = this.token;
		const revision = this.revision;
		if (!token) throw new MessageError('session.loginRequired');
		try {
			await action(token);
		} catch (error) {
			if (revision === this.revision && error instanceof ApiError && error.status === 401) {
				if (ambiguous401) await this.refresh();
				else this.clear('session.expired');
			}
			throw error;
		}
		if (revision !== this.revision) throw new MessageError('session.changed');
	}

	async profile(displayName: string, username?: string) {
		await this.authenticated((token) =>
			authApi.profile(token, { display_name: displayName, username })
		);
		await this.refresh();
	}

	async upgrade(username: string, password: string) {
		await this.authenticated((token) => authApi.upgrade(token, { username, password }), true);
		await this.refresh();
	}

	async changePassword(oldPassword: string, newPassword: string) {
		await this.authenticated(
			(token) => authApi.password(token, { old_password: oldPassword, new_password: newPassword }),
			true
		);
		this.clear('session.passwordChanged');
	}

	async logout(all = false) {
		await this.authenticated((token) => authApi.logout(token, all));
		this.clear(all ? 'session.loggedOutAll' : 'session.loggedOut');
	}
}

export const account = new AccountSession();
