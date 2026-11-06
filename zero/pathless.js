class Pathless {
	#cache = new Map();
	#focused = null;
	#rendering = null;
	#previous = null;
	static #decoder = new TextDecoder();
	static #range = new Range();
	static #variants = [[0], [0, 1], [0, 1, 2, 3]];

	constructor() {
		this.el = document.getElementById('universe');
		this.frames = [];
		this.panels = [];
		this.spaces = Array.from(this.el.children, (space, index) =>
			Object.assign(space, {
				frames: index === 3 ? this.panels : this.frames,
				index: 0,
				mounted: null,
				states: new Map(),
			}),
		);
		this.#focused = this.spaces[0];
		this.#measure();
		new ResizeObserver(() => this.#measure()).observe(this.el, {
			box: 'border-box',
		});
	}
	#measure() {
		const { width, height } = this.el.getBoundingClientRect();
		Object.assign(this.el, { width, height });
	}

	async init() {
		const [frames, panels] = await fetch(circuit).then((response) =>
			response.json(),
		);
		for (const [pool, group] of [
			[this.frames, frames],
			[this.panels, panels],
		]) {
			pool.length = 0;
			for (const html of group)
				pool.push(Pathless.#range.createContextualFragment(html));
		}
		this.#arrange();
		for (const space of this.spaces) this.#mount(space);
		return this;
	}

	get observable() {
		return this.spaces.filter((space) => !space.hidden);
	}
	get focused() {
		return this.#focused;
	}
	set focused(space) {
		if (this.spaces.includes(space) && !space.hidden) this.#focused = space;
	}

	layout(layout, variant = 0) {
		if (
			!Number.isInteger(layout) ||
			!Pathless.#variants[layout]?.includes(variant)
		)
			return this;
		const { dataset } = this.el;
		const current = Number(dataset.layout);
		if (layout === 0 && current !== 0)
			this.#previous = [current, Number(dataset.variant)];
		else if (layout === 0) {
			if (!this.#previous) return this;
			[layout, variant] = this.#previous;
		}
		Object.assign(dataset, { layout, variant });
		this.#arrange();
		return this;
	}
	focus(direction = 1, space = this.focused) {
		const spaces = this.observable;
		const index = spaces.indexOf(space);
		if (index >= 0)
			this.focused = spaces.at((index + direction) % spaces.length);
		return this;
	}
	panel(visible = this.spaces[3].hidden) {
		const panel = this.spaces[3];
		panel.toggleAttribute('hidden', !visible);
		this.#arrange(
			this.focused === panel && !visible ? this.spaces[0] : this.focused,
		);
		return this;
	}
	navigate(direction = 1, space = this.focused) {
		if (this.#rendering)
			throw new Error('Cannot navigate during frame initialization');
		if (!this.spaces.includes(space) || !space.frames.length) return this;
		const { length } = space.frames;
		space.index = (((space.index + direction) % length) + length) % length;
		this.#mount(space);
		return this;
	}

	frame(space = this.#rendering ?? this.focused) {
		const { index, states } = space;
		let state = states.get(index);
		if (!state) states.set(index, (state = new Map()));
		return { space, el: space.firstElementChild, state };
	}

	source(path, bytes = false) {
		if (!path) throw new TypeError('source() requires a path');
		const cached = !bytes && this.#cache.get(path);
		if (cached) return cached;
		const request = fetch(`${circuit}/${path}`)
			.then((response) => response.bytes())
			.then((data) => this.#decode(data, bytes));
		if (bytes) return request;
		const source = request.catch((error) => {
			this.#cache.delete(path);
			throw error;
		});
		this.#cache.set(path, source);
		return source;
	}

	#arrange(target = this.focused) {
		const layout = Number(this.el.dataset.layout);
		for (let index = 0; index < 3; index++) {
			const space = this.spaces[index];
			space.toggleAttribute(
				'hidden',
				layout === 0 ? space !== target : index > layout,
			);
		}
		if (layout === 0 && target === this.spaces[3]) target.hidden = false;
		this.focused = target.hidden ? this.observable[0] : target;
	}

	#mount(space) {
		const frame = space.frames[space.index];
		if (!frame || space.mounted === frame) return;
		const fragment = frame.cloneNode(true);
		const rendering = this.#rendering;
		this.#rendering = space;
		space.mounted = frame;
		try {
			space.replaceChildren(fragment);
		} catch (error) {
			space.mounted = null;
			throw error;
		} finally {
			this.#rendering = rendering;
		}
	}

	#decode(bytes, raw = true) {
		let offset = 0;
		const key = raw ? 'data' : 'url';
		const uvarint = () => {
			let value = 0;
			for (let shift = 0; ; shift += 7) {
				const byte = bytes[offset++];
				value += (byte & 0x7f) * 2 ** shift;
				if (!(byte & 0x80)) return value >>> 0;
			}
		};
		const field = () => {
			const length = uvarint();
			offset += length;
			return bytes.subarray(offset - length, offset);
		};
		const ones = () =>
			Array.from({ length: uvarint() }, () => {
				const name = Pathless.#decoder.decode(field());
				const type = Pathless.#decoder.decode(field());
				let data = field();
				if (!raw)
					data =
						type || data.length
							? URL.createObjectURL(new Blob([data], { type }))
							: null;
				return { name, type, [key]: data, ones: ones() };
			});
		return ones();
	}
}
