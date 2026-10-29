class Universe {
	spaces = pathless.spaces;
	#layouts = [[0], [1, 2], [3, 4, 5, 6]];
	#visibility = [1, 2, 3].map((count) => this.spaces.slice(0, count));
	#origins = [
		[[0, 0]],
		[
			[0, 0.5],
			[0, -0.5],
		],
		[
			[-0.5, 0],
			[0.5, 0],
		],
		[
			[0, 0.5],
			[0.5, -0.5],
			[-0.5, -0.5],
		],
		[
			[-0.5, 0],
			[0.5, 0.5],
			[0.5, -0.5],
		],
		[
			[0.5, 0],
			[-0.5, -0.5],
			[-0.5, 0.5],
		],
		[
			[0, -0.5],
			[-0.5, 0.5],
			[0.5, 0.5],
		],
	];
	#layout = [0, 0, null];
	#visible = this.#visibility[0];
	#focused = this.spaces[0];
	#rendering = null;

	constructor() {
		this.#measure();
		new ResizeObserver(() => this.#measure()).observe(pathless.el, {
			box: 'border-box',
		});
		this.#show(this.#visible);
	}

	universeState(
		layout = this.#layout[0],
		variant = this.#layout[1],
		focused = this.#focused,
		visible = this.#visible,
	) {
		return { layout, variant, focused, visible };
	}
	frame(space = this.#rendering ?? this.#focused) {
		const { index, states } = space;
		if (!states.has(index)) states.set(index, new Map());
		return {
			space,
			el: space.firstElementChild,
			state: states.get(index),
		};
	}
	layout(layout, space) {
		let [current, variant, previous] = this.#layout;
		const cycling = layout === current && layout !== 0;
		if (layout === 0 && current === 0 && space === undefined) {
			if (!previous) return pathless;
			[layout, variant] = previous;
		} else {
			if (layout === 0 && current !== 0) previous = [current, variant];
			variant = cycling
				? (variant + 1) % this.#layouts[layout].length
				: 0;
		}
		this.#layout = [layout, variant, previous];
		Object.assign(pathless.el.dataset, { layout, variant });
		if (cycling) {
			this.#updateOrigins();
			return pathless;
		}
		return this.#show(
			layout === 0 ? [space ?? this.#focused] : this.#visibility[layout],
		);
	}
	focus(direction = 1, target = this.#focused) {
		const spaces = this.#visible.filter(
			(space) => space !== this.spaces[3],
		);
		const index = spaces.indexOf(target);
		if (index < 0) return pathless;
		const length = spaces.length;
		this.#focused =
			spaces[(((index + direction) % length) + length) % length];
		return pathless;
	}
	panel(visible = this.spaces[3].hidden) {
		const panel = this.spaces[3];
		if (panel.hidden === !visible) return pathless;
		panel.hidden = !visible;
		return this.#show(this.#visible);
	}
	#measure() {
		const { width, height } = pathless.el.getBoundingClientRect();
		Object.assign(pathless.el, { width, height });
	}
	#updateOrigins() {
		const [layout, variant] = this.#layout;
		const origins = this.#origins[this.#layouts[layout][variant]];
		const panel = this.spaces[3];
		const inset = panel.hidden ? 0 : 0.15;
		for (const [index, space] of this.#visible.entries()) {
			const [horizontal, vertical] =
				space === panel ? [0, -0.85] : origins[index];
			const origin = (space.origin ??= {});
			origin.x = horizontal;
			origin.y =
				space === panel ? vertical : inset + (1 - inset) * vertical;
		}
	}
	navigate(direction = 1, space = this.#focused) {
		const { length } = space.frames;
		if (!length) return pathless;
		const index = space.index + direction;
		space.index = ((index % length) + length) % length;
		return this.#exec(space, space.frames[space.index].data);
	}
	#exec(space, data) {
		const rendering = this.#rendering;
		this.#rendering = space;
		try {
			pathless.exec(data, space);
		} finally {
			this.#rendering = rendering;
		}
		return pathless;
	}
	#show(spaces) {
		const panel = this.spaces[3];
		spaces = spaces.filter((space) => space !== panel);
		if (!spaces.length) spaces.push(this.spaces[0]);
		if (!panel.hidden) spaces.push(panel);
		this.#visible = spaces;
		this.spaces.forEach((space) => {
			space.hidden = !spaces.includes(space);
		});
		if (this.#focused.hidden) this.#focused = spaces[0];
		this.#updateOrigins();
		for (const space of spaces) {
			if (space.firstElementChild) continue;
			const frame = space.frames[space.index];
			if (frame) this.#exec(space, frame.data);
		}
		return pathless;
	}
}
