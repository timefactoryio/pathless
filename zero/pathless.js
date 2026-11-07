class Pathless {
	#cache = new Map();
	#rendering = null;
	#view = { selection: [0, 0] };
	static #decoder = new TextDecoder();
	static #range = new Range();
	static #areas = { full: [1, 1, 3, 3], dock: [3, 1, 4, 3] };
	static #layouts = {
		'0,0': [Pathless.#areas.full],
		'1,0': [
			[1, 1, 2, 3],
			[2, 1, 3, 3],
		],
		'1,1': [
			[1, 1, 3, 2],
			[1, 2, 3, 3],
		],
		'2,0': [
			[1, 1, 2, 3],
			[2, 2, 3, 3],
			[2, 1, 3, 2],
		],
		'2,1': [
			[1, 1, 3, 2],
			[1, 2, 2, 3],
			[2, 2, 3, 3],
		],
		'2,2': [
			[1, 2, 3, 3],
			[2, 1, 3, 2],
			[1, 1, 2, 2],
		],
		'2,3': [
			[2, 1, 3, 3],
			[1, 1, 2, 2],
			[1, 2, 2, 3],
		],
	};

	constructor() {
		const view = this.#view;
		this.el = document.getElementById('universe');
		view.panel = this.el.querySelector('#panel');
		this.frames = [];
		this.panels = [];
		this.spaces = Array.from(this.el.children, (space) =>
			Object.assign(space, {
				frames: space === view.panel ? this.panels : this.frames,
				index: 0,
				mounted: null,
				states: new Map(),
			}),
		);
		view.main = this.spaces.filter((space) => space !== view.panel);
		view.focused = view.main[0];
		this.#solo(view.focused);
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
		const groups = await fetch(circuit).then((response) => response.json());
		for (const [index, pool] of [this.frames, this.panels].entries()) {
			pool.length = 0;
			for (const html of groups[index])
				pool.push(Pathless.#range.createContextualFragment(html));
		}
		for (const space of this.spaces) this.#mount(space);
		return this;
	}

	get observable() {
		return this.spaces.filter((space) => !space.hidden);
	}
	get focused() {
		return this.#view.focused;
	}
	set focused(space) {
		if (this.spaces.includes(space) && !space.hidden)
			this.#view.focused = space;
	}

	layout(layout, variant = 0) {
		if (!Pathless.#layouts[`${layout},${variant}`]) return this;
		const view = this.#view;
		if (layout === 0) {
			const current = view.selection;
			if (current[0] !== 0) view.previous = current;
			else if (view.previous) [layout, variant] = view.previous;
			else return this;
		}
		view.selection = [layout, variant];
		if (layout === 0) this.#solo(view.focused);
		else {
			const areas = Pathless.#layouts[`${layout},${variant}`];
			for (const [index, space] of view.main.entries())
				this.#place(space, areas[index]);
			if (view.focused.hidden) view.focused = view.main[0];
			this.#place(
				view.panel,
				view.panel.hidden ? undefined : Pathless.#areas.dock,
			);
		}
		return this;
	}
	focus(direction = 1, space = this.focused) {
		const spaces = this.observable;
		const index = spaces.indexOf(space);
		if (index >= 0)
			this.focused = spaces.at((index + direction) % spaces.length);
		return this;
	}
	panel(visible = this.#view.panel.hidden) {
		const view = this.#view;
		const active = view.focused === view.panel;
		const solo = view.selection[0] === 0;
		if (active && !visible) view.focused = view.main[0];
		if (active && solo) this.#solo(view.focused, visible);
		else
			this.#place(view.panel, visible ? Pathless.#areas.dock : undefined);
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

	#solo(target, panelVisible = !this.#view.panel.hidden) {
		const { full, dock } = Pathless.#areas;
		for (const space of this.spaces) {
			let area;
			if (space === target) area = full;
			else if (space === this.#view.panel && panelVisible) area = dock;
			this.#place(space, area);
		}
	}

	#place(space, area) {
		space.hidden = !area;
		space.style.gridArea = area?.join(' / ') ?? '';
		if (space !== this.#view.panel) return;
		const size = area === Pathless.#areas.dock ? '15%' : '0px';
		this.el.style.setProperty('--dock-size', size);
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
