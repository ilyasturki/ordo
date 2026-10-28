import { registerPlugin } from '@capacitor/core';

type Feedback = {
	click: () => Promise<void>;
	haptic: (options: { kind: 'hold' | 'confirm' }) => Promise<void>;
};

const feedback = import.meta.env.APP_BUILD ? registerPlugin<Feedback>('Feedback') : null;

/** The system tap sound, on a discrete tap. */
export function tapClick(): void {
	void feedback?.click().catch(() => undefined);
}

/** The buzz of a long press taken. */
export function tapHold(): void {
	if (feedback === null) {
		navigator.vibrate?.(10);
		return;
	}
	void feedback.haptic({ kind: 'hold' }).catch(() => undefined);
}
