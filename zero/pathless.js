class Pathless {
	#cache = new Map();
	#rendering = null;
	static #decoder = new TextDecoder();
	static #range = new Range();

	constructor() {
		this.el = document.getElementById('universe');
		this.spaces = Array.from(this.el.children, (el) => ({
			el,
			index: 0,
			states: new Map(),
		}));
		this.focused = this.spaces[0];
		this.observable = [this.focused];
	}

	async init() {
		const [frames, panels] = await fetch(circuit).then((response) =>
			response.json(),
		);
		const fragment = (html) =>
			Pathless.#range.createContextualFragment(html);
		this.frames = frames.map(fragment);
		this.panels = panels.map(fragment);
		for (const space of this.spaces)
			space.frames = space === this.spaces[3] ? this.panels : this.frames;
		this.#mount(this.spaces);
		return this;
	}

	set show(spaces) {
		for (const space of [spaces].flat()) {
			if (!this.observable.includes(space)) this.observable.push(space);
			space.el.hidden = false;
		}
	}
	set hide(spaces) {
		for (const space of [spaces].flat()) {
			if (space === this.focused) continue;
			const index = this.observable.indexOf(space);
			if (index >= 0) this.observable.splice(index, 1);
			space.el.hidden = true;
		}
	}

	focus(direction = 1, space = this.focused) {
		const spaces = this.observable;
		const index = spaces.indexOf(space);
		if (index >= 0)
			this.focused = spaces.at((index + direction) % spaces.length);
		return this;
	}
	panel(visible) {
		const space = this.spaces[3];
		if (visible === undefined ? space.el.hidden : visible)
			this.show = space;
		else this.hide = space;
		return this;
	}
	navigate(direction = 1, space = this.focused) {
		const { length } = space.frames;
		if (!length) return this;
		const index = (((space.index + direction) % length) + length) % length;
		if (index === space.index) return this;
		space.index = index;
		this.#mount(space);
		return this;
	}
	frame(space = this.#rendering ?? this.focused) {
		const { index, states } = space;
		let state = states.get(index);
		if (!state) states.set(index, (state = new Map()));
		return { space, el: space.el.firstElementChild, state };
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

	#mount(spaces) {
		for (const space of [spaces].flat()) {
			const frame = space.frames[space.index];
			if (!frame) continue;
			const rendering = this.#rendering;
			this.#rendering = space;
			try {
				space.el.replaceChildren(frame.cloneNode(true));
			} finally {
				this.#rendering = rendering;
			}
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
		const ones = () => {
			const nodes = new Array(uvarint());
			for (let index = 0; index < nodes.length; index++) {
				const name = Pathless.#decoder.decode(field());
				const type = Pathless.#decoder.decode(field());
				let data = field();
				if (!raw)
					data =
						type || data.length
							? URL.createObjectURL(new Blob([data], { type }))
							: null;
				nodes[index] = { name, type, [key]: data, ones: ones() };
			}
			return nodes;
		};
		return ones();
	}
}
