class Universe {
	static layouts = [
		[[[0, 0]]],
		[
			[
				[0, 0.5],
				[0, -0.5],
			],
			[
				[-0.5, 0],
				[0.5, 0],
			],
		],
		[
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
		],
	];
	#rendering = null;
	#focused = null;
	#mounted = new WeakMap();

	constructor() {
		this.prevLayout = null;
		this.#focused = pathless.spaces[0];
		pathless.universe = this;
		for (const name of [
			'frame',
			'layout',
			'focus',
			'panel',
			'navigate',
			'sync',
		])
			pathless[name] = this[name].bind(this);
		this.#measure();
		new ResizeObserver(() => this.#measure()).observe(pathless.el, {
			box: 'border-box',
		});
		this.sync();
	}

	get origins() {
		const { layout, variant } = pathless.el.dataset;
		return this.layouts[layout][variant];
	}
	get space() {
		return this.#focused;
	}
	frame(space = this.#rendering ?? this.space) {
		const { index, states } = space;
		if (!states.has(index)) states.set(index, new Map());
		return {
			space,
			el: space.firstElementChild,
			state: states.get(index),
		};
	}
	layout(layout, space) {
		const { dataset } = pathless.el;
		const current = Number(dataset.layout);
		let variant = Number(dataset.variant);
		if (layout === 0 && current === 0 && space === undefined) {
			if (!this.prevLayout) return pathless;
			[layout, variant] = this.prevLayout;
		} else {
			if (layout === 0 && current !== 0)
				this.prevLayout = [current, variant];
			variant =
				layout === current && layout !== 0
					? (variant + 1) % this.layouts[layout].length
					: 0;
		}
		if (dataset.layout !== String(layout)) dataset.layout = layout;
		if (dataset.variant !== String(variant)) dataset.variant = variant;
		const single = layout === 0;
		const target = single ? (space ?? this.space) : this.space;
		for (const space of pathless.spaces.slice(0, 3)) {
			const hidden = single && space !== target;
			if (space.hidden !== hidden) space.hidden = hidden;
		}
		if (single && target.hidden) target.hidden = false;
		this.#focused = target.checkVisibility()
			? target
			: pathless.observable[0];
		return this.sync();
	}
	focus(direction = 1, target = this.space) {
		const spaces = pathless.observable;
		const index = spaces.indexOf(target);
		if (index < 0) return pathless;
		this.#focused = spaces.at((index + direction) % spaces.length);
		return pathless;
	}
	panel(visible = pathless.spaces[3].hidden) {
		const panel = pathless.spaces[3];
		if (panel.hidden === !visible) return pathless;
		panel.hidden = !visible;
		if (!pathless.observable.length) pathless.spaces[0].hidden = false;
		if (!this.space.checkVisibility())
			this.#focused = pathless.observable[0];
		return this.sync();
	}
	#measure() {
		const { width, height } = pathless.el.getBoundingClientRect();
		Object.assign(pathless.el, { width, height });
	}
	navigate(direction = 1, space = this.space) {
		space.index += direction;
		return this.sync(space);
	}
	sync(target) {
		for (const space of target ? [target] : pathless.observable) {
			if (!space.checkVisibility()) continue;
			const { frames } = space;
			const { length } = frames;
			if (!length) continue;
			space.index = ((space.index % length) + length) % length;
			const frame = frames[space.index];
			const { data } = frame;
			const mounted = this.#mounted.get(space.firstElementChild);
			if (mounted?.frame === frame && mounted.data === data) continue;
			const rendering = this.#rendering;
			this.#rendering = space;
			try {
				pathless.exec(data, space);
				if (space.firstElementChild)
					this.#mounted.set(space.firstElementChild, { frame, data });
			} finally {
				this.#rendering = rendering;
			}
		}
		return pathless;
	}
}
