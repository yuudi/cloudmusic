import type en from './messages/en.json';

export type MessageKey = keyof typeof en;
export type MessageState = MessageKey | '';

export class MessageError extends Error {
	constructor(public key: MessageKey) {
		super(key);
	}
}
