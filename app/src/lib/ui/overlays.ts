type Close = () => void;

const stack: Close[] = [];

/** Puts an open sheet, menu, dialog or mode on the stack the back button closes from the top. */
export function registerOverlay(close: Close): () => void {
	stack.push(close);
	return () => {
		const i = stack.lastIndexOf(close);
		if (i !== -1) {
			stack.splice(i, 1);
		}
	};
}

export function closeTopOverlay(): boolean {
	const top = stack.pop();
	if (top === undefined) {
		return false;
	}
	top();
	return true;
}
