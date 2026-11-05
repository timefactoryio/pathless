class Input {
	#pointer = null;
	#bindings = new WeakMap();

	constructor() {
		const { el } = pathless;
		this.#register(['1'], () => pathless.layout(0));
		this.#register(['2'], () => pathless.layout(1));
		this.#register(['3'], () => pathless.layout(2));
		this.#register(['tab'], () => pathless.focus(1));
		this.#register(['q', 'swipe left'], () => pathless.navigate(-1));
		this.#register(['e', 'swipe right'], () => pathless.navigate(1));
		this.#register(['z', 'bottomCorner'], () => pathless.panel());

		pathless.click = (el, callback) =>
			this.#register(['click'], callback, el);
		pathless.x = this.x.bind(this);
		pathless.y = this.y.bind(this);

		const up = this.up.bind(this);
		for (const [type, handler] of [
			['click', this.click.bind(this)],
			['pointerdown', this.down.bind(this)],
			['pointerup', up],
			['pointercancel', up],
			['lostpointercapture', up],
		]) {
			el.addEventListener(type, handler);
		}
		document.addEventListener('keydown', this.keydown.bind(this));
	}

	x(left, right) {
		const { el } = pathless.frame();
		this.#register(['a', 'left'], left, el);
		return this.#register(['d', 'right'], right, el);
	}
	y(up, down) {
		const { el } = pathless.frame();
		this.#register(['w', 'up'], up, el);
		return this.#register(['s', 'down'], down, el);
	}

	#register(triggers, callback, el = pathless.el) {
		let callbacks = this.#bindings.get(el);
		if (!callbacks) this.#bindings.set(el, (callbacks = new Map()));
		for (const trigger of triggers)
			callbacks.set(trigger.toLowerCase(), callback);
		return pathless;
	}
	#callback(trigger, space) {
		trigger = trigger.toLowerCase();
		const el = space?.checkVisibility() ? space.firstElementChild : null;
		return (
			this.#bindings.get(el)?.get(trigger) ??
			this.#bindings.get(pathless.el)?.get(trigger)
		);
	}

	down(event) {
		if (this.#blocked(event) || !event.isPrimary) return;
		if (event.button !== 0 || this.#pointer) return;
		const space = pathless.spaces.find(
			(space) => space.checkVisibility() && space.contains(event.target),
		);
		if (!space) return;
		const start = this.#coordinate(event);
		if (!this.#corner(start)) pathless.focus(0, space);
		if (this.#button(event.target)) return;
		this.#pointer = {
			space,
			el: space.firstElementChild,
			id: event.pointerId,
			start,
		};
		pathless.el.setPointerCapture(event.pointerId);
	}
	up(event) {
		const pointer = this.#pointer;
		if (!pointer || pointer.id !== event.pointerId) return;
		this.#pointer = null;
		const { start, space, el } = pointer;
		if (event.type !== 'pointerup' || this.#blocked(event)) return;
		if (!space.checkVisibility() || el !== space.firstElementChild) return;
		pointer.end = this.#coordinate(event);
		const gesture = this.#gesture(pointer);
		if (gesture)
			this.#callback(gesture, space)?.(start, pointer.end, space);
	}
	click(event) {
		if (event.defaultPrevented) return;
		const el = this.#button(event.target);
		if (el) this.#bindings.get(el).get('click').call(el, event);
	}
	keydown(event) {
		if (this.#blocked(event) || event.isComposing) return;
		if (event.ctrlKey || event.metaKey || event.altKey) return;
		const { space } = pathless.frame();
		const callback = this.#callback(event.key, space);
		if (!callback) return;
		event.preventDefault();
		callback(undefined, undefined, space);
	}

	#gesture({ start, end, space }) {
		const deltaX = Math.abs(end[0] - start[0]);
		const deltaY = Math.abs(end[1] - start[1]);
		if (deltaY >= 0.1 && deltaY > deltaX) return null;
		if (deltaX >= 0.1)
			return end[0] < start[0] ? 'swipe left' : 'swipe right';
		if (this.#corner(start) && this.#corner(end)) return 'bottomCorner';

		const { origins } = pathless.universe;
		const [horizontal, vertical] =
			origins[pathless.spaces.indexOf(space)] ?? origins[0];
		const offsetX = end[0] - horizontal;
		const offsetY = end[1] - vertical;
		if (Math.abs(offsetX) >= Math.abs(offsetY))
			return offsetX < 0 ? 'left' : 'right';
		return offsetY < 0 ? 'down' : 'up';
	}
	#corner([horizontal, vertical]) {
		return vertical <= -0.8 && Math.abs(horizontal) >= 0.8;
	}
	#blocked(event) {
		return (
			event.defaultPrevented ||
			event.target.isContentEditable ||
			event.target.closest?.('input, textarea, select')
		);
	}
	#button(el) {
		while (el && el !== pathless.el) {
			if (this.#bindings.get(el)?.has('click')) return el;
			el = el.parentElement;
		}
	}
	#coordinate({ clientX, clientY }) {
		const { width, height } = pathless.el;
		return [
			Math.max(-1, Math.min(1, (2 * clientX) / width - 1)),
			Math.max(-1, Math.min(1, 1 - (2 * clientY) / height)),
		];
	}
}
