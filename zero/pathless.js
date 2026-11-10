class Pathless {
	#cache = new Map();
	#rendering = null;
	static #range = new Range();
	static #layouts = ['x', 'y', 'up', 'right', 'down', 'left'];

	constructor() {
		this.el = document.getElementById('universe');
		this.layout = [null, null];
		this.spaces = Array.from(this.el.children, (el) => ({
			el,
			index: 0,
			states: new Map(),
		}));
		this.focused = this.spaces[0];
	}

	async init() {
		const [frames, panels] = await (await fetch(circuit)).json();
		const fragment = (html) =>
			Pathless.#range.createContextualFragment(html);
		this.frames = frames.map(fragment);
		this.panels = panels.map(fragment);
		for (const space of this.spaces)
			space.frames = space === this.spaces[3] ? this.panels : this.frames;
		this.#mount(this.spaces);
		return this;
	}

	get observable() {
		return this.spaces.filter((space) => !space.el.hidden);
	}
	set show(spaces) {
		for (const space of Array.isArray(spaces) ? spaces : [spaces])
			space.el.hidden = false;
	}
	set hide(spaces) {
		for (const space of Array.isArray(spaces) ? spaces : [spaces])
			if (space !== this.focused) space.el.hidden = true;
	}

	focus(direction = 1, space = this.focused) {
		const spaces = this.observable;
		const index = spaces.indexOf(space);
		if (index >= 0)
			this.focused = spaces.at((index + direction) % spaces.length);
		return this;
	}
	panel(visible = this.spaces[3].el.hidden) {
		const space = this.spaces[3];
		if (visible) this.show = space;
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
	source(path, raw = false) {
		if (!path) throw new TypeError('source() requires a path');
		if (!raw && this.#cache.has(path)) return this.#cache.get(path);
		const request = fetch(`${circuit}/${path}`)
			.then((response) => {
				if (!response.ok)
					throw new Error(`HTTP ${response.status}: ${response.url}`);
				return response.arrayBuffer();
			})
			.catch((error) => {
				if (!raw) this.#cache.delete(path);
				throw error;
			});
		if (!raw) this.#cache.set(path, request);
		return request;
	}
	layoutZero(space = this.focused) {
		this.focused = space;
		return this.#layout(this.layout[0] === null ? this.layout[1] : null);
	}
	layoutOne() {
		return this.#layout(this.layout[0] === 0 ? 1 : 0);
	}
	layoutTwo() {
		const layout = this.layout[0];
		return this.#layout(layout >= 2 ? ((layout - 1) % 4) + 2 : 2);
	}

	#mount(spaces) {
		for (const space of Array.isArray(spaces) ? spaces : [spaces]) {
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
	#layout(layout) {
		const count = layout === null ? 0 : layout < 2 ? 2 : 3;
		if (count === 2 && this.focused === this.spaces[2])
			this.focused = this.spaces[0];
		this.show = count ? this.spaces.slice(0, count) : this.focused;
		this.hide = count ? this.spaces.slice(count, 3) : this.spaces;
		this.layout = [layout, this.layout[0]];
		this.el.className = Pathless.#layouts[layout] ?? '';
		return this;
	}
}
