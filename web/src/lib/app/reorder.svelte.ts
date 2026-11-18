export type Press = {
	/** The lifted item's key, as its element's data-line names it. */
	key: string;
	/** In the order shown, its own among them. */
	items: HTMLElement[];
	allowed?: (slot: number) => boolean;
	/** The keys in the order the release leaves them, and as the lift found them. */
	drop: (order: string[], lifted: string[]) => void;
	/** Where the scroller's visible top starts, below a sticky bar. */
	top?: () => number;
};

const mouseSlop = 4;
const touchSlop = 8;
const longPress = 450;
const edgeZone = 56;

function scroller(el: HTMLElement): HTMLElement | undefined {
	for (let at = el.parentElement; at; at = at.parentElement) {
		const y = getComputedStyle(at).overflowY;
		if ((y === 'auto' || y === 'scroll') && at.scrollHeight > at.clientHeight) return at;
	}
	return undefined;
}

/** A mouse moved or a touch held lifts an item; the others make room for it and the release hands over their order. */
export class Reorder {
	key: string | undefined = $state();
	slot = $state(0);
	dy = $state(0);
	#press: Press | undefined;
	#pointer = -1;
	#touch = false;
	#x0 = 0;
	#y0 = 0;
	#y = 0;
	#low = 0;
	#high = 0;
	#keys: string[] = [];
	#others: string[] = [];
	#index = new Map<string, number>();
	#from = 0;
	#height = 0;
	#tops: number[] = [];
	#spans: number[] = [];
	#scroller: HTMLElement | undefined;
	#scroll0 = 0;
	#timer: ReturnType<typeof setTimeout> | undefined;
	#frame = 0;
	#last = 0;
	#carry = 0;
	#done: (() => void)[] = [];

	/** Touch scroll stops once lifted, and no link drag or long-press menu takes the press. */
	surface = (node: HTMLElement) => {
		const touchmove = (e: TouchEvent) => {
			if (this.key) e.preventDefault();
		};
		const contextmenu = (e: Event) => {
			if (!this.#press || !this.#touch) return;
			e.preventDefault();
			if (!this.key) this.#lift();
		};
		const dragstart = (e: Event) => e.preventDefault();
		node.addEventListener('touchmove', touchmove, { passive: false });
		node.addEventListener('contextmenu', contextmenu);
		node.addEventListener('dragstart', dragstart);
		return () => {
			node.removeEventListener('touchmove', touchmove);
			node.removeEventListener('contextmenu', contextmenu);
			node.removeEventListener('dragstart', dragstart);
		};
	};

	look(key: string): string[] {
		if (!this.key) return ['flow-root'];
		return ['flow-root', key === this.key ? 'relative z-10' : 'motion-safe:transition-transform motion-safe:duration-150'];
	}

	transform(key: string): string | undefined {
		return this.key ? `translateY(${this.#offset(key)}px)` : undefined;
	}

	#offset(key: string): number {
		if (key === this.key) return this.dy;
		const j = this.#index.get(key);
		if (j === undefined) return 0;
		return (j >= this.slot ? this.#height : 0) - (j >= this.#from ? this.#height : 0);
	}

	order(): string[] {
		if (!this.key) return [];
		return [...this.#others.slice(0, this.slot), this.key, ...this.#others.slice(this.slot)];
	}

	press(e: PointerEvent, press: Press): void {
		if (this.#press || !e.isPrimary || e.button !== 0 || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return;
		this.#press = press;
		this.#pointer = e.pointerId;
		this.#touch = e.pointerType !== 'mouse';
		this.#x0 = e.clientX;
		this.#y0 = this.#y = e.clientY;
		this.#listen(window, 'pointermove', this.#move);
		this.#listen(window, 'pointerup', this.#up);
		this.#listen(window, 'pointercancel', this.#cancel);
		this.#listen(window, 'blur', this.#cancel);
		this.#listen(window, 'keydown', this.#key, true);
		if (this.#touch) this.#timer = setTimeout(() => this.#lift(), longPress);
	}

	#listen<E extends Event>(target: Window, type: string, fn: (e: E) => void, capture = false) {
		const handler = fn as EventListener;
		target.addEventListener(type, handler, capture);
		this.#done.push(() => target.removeEventListener(type, handler, capture));
	}

	#lift() {
		const press = this.#press;
		if (!press || this.key) return;
		clearTimeout(this.#timer);
		if (press.items.some((el) => !el.isConnected)) return this.#end();
		const rects = press.items.map((el) => el.getBoundingClientRect());
		this.#keys = press.items.map((el) => el.dataset.line ?? '');
		this.#from = this.#keys.indexOf(press.key);
		if (this.#from < 0) return this.#end();
		// From one item's top to the next: what an item takes, gaps included.
		const spans = rects.map((r, i) => (i + 1 < rects.length ? rects[i + 1].top - r.top : r.height));
		this.#height = spans[this.#from];
		this.#others = this.#keys.filter((_, i) => i !== this.#from);
		this.#index = new Map(this.#others.map((k, j) => [k, j]));
		this.#spans = spans.filter((_, i) => i !== this.#from);
		this.#tops = [0];
		for (const s of this.#spans) this.#tops.push(this.#tops.at(-1)! + s);
		this.#scroller = scroller(press.items[0]);
		this.#scroll0 = this.#scrollTop();
		this.#low = this.#high = this.#y;
		this.slot = this.#from;
		this.dy = this.#y - this.#y0;
		this.key = press.key;
		this.#last = performance.now();
		this.#frame = requestAnimationFrame(this.#tick);
	}

	#scrollTop(): number {
		return this.#scroller ? this.#scroller.scrollTop : scrollY;
	}

	#move = (e: PointerEvent) => {
		if (e.pointerId !== this.#pointer) return;
		this.#y = e.clientY;
		if (!this.key) {
			const far = Math.hypot(e.clientX - this.#x0, e.clientY - this.#y0);
			if (!this.#touch && far > mouseSlop) this.#lift();
			else if (this.#touch && far > touchSlop) this.#end();
			return;
		}
		e.preventDefault();
		this.#follow();
	};

	#follow() {
		this.#low = Math.min(this.#low, this.#y);
		this.#high = Math.max(this.#high, this.#y);
		this.dy = this.#y - this.#y0 + this.#scrollTop() - this.#scroll0;
		const want = this.#tops[this.#from] + this.dy;
		// A slot is taken once the item's top passes the middle of the line just above it, as the phone's swaps.
		let best = -1;
		for (let s = 0; s < this.#tops.length; s++) {
			if (this.#press?.allowed && !this.#press.allowed(s)) continue;
			if (best >= 0 && want <= this.#tops[s] - this.#spans[s - 1] / 2) break;
			best = s;
		}
		if (best < 0) return;
		this.slot = best;
	}

	// An edge scrolls only once the pointer has moved toward it, so an item lifted near one stays put.
	#tick = (now: number) => {
		if (!this.key) return;
		const box = this.#scroller?.getBoundingClientRect();
		const top = Math.max(box?.top ?? 0, this.#press?.top?.() ?? 0);
		const bottom = Math.min(box?.bottom ?? innerHeight, innerHeight);
		const zone = Math.min(edgeZone, (bottom - top) / 4);
		const slop = this.#touch ? touchSlop : mouseSlop;
		let depth = 0;
		if (this.#y < top + zone && this.#y < this.#high - slop) depth = -Math.min(1, (top + zone - this.#y) / zone);
		else if (this.#y > bottom - zone && this.#y > this.#low + slop) depth = Math.min(1, (this.#y - bottom + zone) / zone);
		if (depth !== 0) {
			// The phone's pace at the edge: a twentieth of the view each tenth of a second.
			this.#carry += depth * (bottom - top) * 0.05 * ((now - this.#last) / 100);
			const step = Math.trunc(this.#carry);
			if (step !== 0) {
				const before = this.#scrollTop();
				if (this.#scroller) this.#scroller.scrollTop += step;
				else scrollBy(0, step);
				this.#carry = this.#scrollTop() === before ? 0 : this.#carry - step;
				this.#follow();
			}
		} else this.#carry = 0;
		this.#last = now;
		this.#frame = requestAnimationFrame(this.#tick);
	};

	#up = (e: PointerEvent) => {
		if (e.pointerId !== this.#pointer) return;
		const press = this.#press;
		if (this.key && press) {
			this.#y = e.clientY;
			this.#follow();
			const order = this.order();
			const lifted = this.#keys;
			this.#swallowClick();
			this.#end();
			if (order.join('\n') !== lifted.join('\n')) press.drop(order, lifted);
			return;
		}
		this.#end();
	};

	#cancel = (e: Event) => {
		if (e instanceof PointerEvent && e.pointerId !== this.#pointer) return;
		if (this.key) this.#swallowClick();
		this.#end();
	};

	#key = (e: KeyboardEvent) => {
		if (!this.key) {
			this.#end();
			return;
		}
		e.preventDefault();
		e.stopPropagation();
		if (e.key === 'Escape') this.#end();
	};

	// The release also reaches the item as a click: that one neither opens it nor steps its status.
	#swallowClick() {
		const stop = (e: Event) => {
			e.preventDefault();
			e.stopPropagation();
			removeEventListener('click', stop, true);
		};
		addEventListener('click', stop, true);
		setTimeout(() => removeEventListener('click', stop, true), 400);
	}

	#end() {
		clearTimeout(this.#timer);
		cancelAnimationFrame(this.#frame);
		for (const off of this.#done.splice(0)) off();
		this.#press = undefined;
		this.#pointer = -1;
		this.#carry = 0;
		this.key = undefined;
	}
}
