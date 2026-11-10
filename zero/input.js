class Input {
	#bindings = new WeakMap();
	#active = null;

	constructor() {
		pathless.register = this.register.bind(this);
		pathless.x = this.#x.bind(this);
		pathless.y = this.#y.bind(this);
		pathless.click = (el, callback) =>
			this.register(['click'], callback, el);

		const { el } = pathless;
		const complete = this.#complete.bind(this);
		el.addEventListener('pointerdown', this.#begin.bind(this));
		for (const type of ['pointerup', 'pointercancel', 'lostpointercapture'])
			el.addEventListener(type, complete);
		el.addEventListener('click', this.#activate.bind(this));
		document.addEventListener('keydown', this.#keydown.bind(this));
	}

	#x(left, right) {
		const { el } = pathless.frame();
		this.register(['a', 'left'], left, el);
		return this.register(['d', 'right'], right, el);
	}
	#y(up, down) {
		const { el } = pathless.frame();
		this.register(['w', 'up'], up, el);
		return this.register(['s', 'down'], down, el);
	}
	register(triggers, callback, el = pathless.el) {
		if (typeof callback !== 'function') return pathless;
		let callbacks = this.#bindings.get(el);
		if (!callbacks) this.#bindings.set(el, (callbacks = new Map()));
		for (const trigger of triggers)
			callbacks.set(trigger.toLowerCase(), callback);
		return pathless;
	}
	#dispatch(trigger, event, { space, el, start = null, end = null }) {
		trigger = trigger.toLowerCase();
		const frame = pathless.observable.includes(space)
			? space.el.firstElementChild
			: null;
		for (const owner of el ? [el] : [frame, pathless.el]) {
			const callback = this.#bindings.get(owner)?.get(trigger);
			if (!callback) continue;
			event.preventDefault();
			callback.call(owner, { trigger, space, el: owner, start, end });
			return;
		}
	}

	#keydown(event) {
		if (this.#blocked(event) || event.isComposing) return;
		if (event.ctrlKey || event.metaKey || event.altKey) return;
		this.#dispatch(event.key, event, { space: pathless.focused });
	}
	#begin(event) {
		if (this.#active || this.#blocked(event) || !event.isPrimary) return;
		if (event.button !== 0) return;
		const { space, el } = this.#target(event.target);
		if (!space || el) return;
		const bounds = pathless.el.getBoundingClientRect();
		if (!bounds.width || !bounds.height) return;
		this.#active = {
			id: event.pointerId,
			space,
			frame: space.el.firstElementChild,
			start: this.#coordinate(event.clientX, event.clientY, bounds),
			bounds,
		};
		pathless.el.setPointerCapture(event.pointerId);
	}
	#complete(event) {
		const active = this.#active;
		if (!active || active.id !== event.pointerId) return;
		this.#active = null;
		if (event.type !== 'pointerup' || this.#blocked(event)) return;
		const { space, frame, bounds } = active;
		if (!pathless.observable.includes(space)) return;
		if (space.el.firstElementChild !== frame) return;
		const current = pathless.el.getBoundingClientRect();
		const keys = ['left', 'top', 'width', 'height'];
		if (keys.some((key) => bounds[key] !== current[key])) return;
		active.end = this.#coordinate(event.clientX, event.clientY, bounds);
		const trigger = this.#gesture(active);
		if (!trigger) return;
		if (trigger !== 'bottomcorner') pathless.focus(0, space);
		this.#dispatch(trigger, event, active);
	}
	#activate(event) {
		if (this.#blocked(event)) return;
		const context = this.#target(event.target);
		if (!context.space || !context.el) return;
		pathless.focus(0, context.space);
		this.#dispatch('click', event, context);
	}

	#target(el) {
		const space = pathless.observable.find((space) =>
			space.el.contains(el),
		);
		for (; el && el !== pathless.el; el = el.parentElement)
			if (this.#bindings.get(el)?.has('click')) return { space, el };
		return { space };
	}
	#gesture({ start, end, space, bounds }) {
		const deltaX = Math.abs(end[0] - start[0]);
		const deltaY = Math.abs(end[1] - start[1]);
		if (Math.max(deltaX, deltaY) >= 0.1) {
			if (deltaY > deltaX) return null;
			return end[0] < start[0] ? 'swipe left' : 'swipe right';
		}
		if (
			Math.max(start[1], end[1]) <= -0.8 &&
			Math.min(Math.abs(start[0]), Math.abs(end[0])) >= 0.8
		)
			return 'bottomcorner';

		const { left, top, width, height } = space.el.getBoundingClientRect();
		const [horizontal, vertical] = this.#coordinate(
			left + width / 2,
			top + height / 2,
			bounds,
		);
		const offsetX = end[0] - horizontal;
		const offsetY = end[1] - vertical;
		if (Math.abs(offsetX) >= Math.abs(offsetY))
			return offsetX < 0 ? 'left' : 'right';
		return offsetY < 0 ? 'down' : 'up';
	}
	#blocked(event) {
		return (
			event.defaultPrevented ||
			event.target?.isContentEditable ||
			event.target?.closest?.('input, textarea, select')
		);
	}
	#coordinate(clientX, clientY, { left, top, width, height }) {
		return [
			Math.max(-1, Math.min(1, (2 * (clientX - left)) / width - 1)),
			Math.max(-1, Math.min(1, 1 - (2 * (clientY - top)) / height)),
		];
	}
}
