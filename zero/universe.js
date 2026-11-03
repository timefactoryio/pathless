class Universe {
	layouts = [
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

	constructor() {
		this.currentLayout = [0, 0, null];
		this.spaces = [pathless.spaces[0]];
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

	universeState() {
		const [layout, variant] = this.currentLayout;
		return {
			layout: this.currentLayout,
			focused: this.spaces[0],
			visible: this.spaces,
			origins: this.layouts[layout][variant],
		};
	}
	frame(space = this.#rendering ?? this.spaces[0]) {
		const { index, states } = space;
		if (!states.has(index)) states.set(index, new Map());
		return {
			space,
			el: space.firstElementChild,
			state: states.get(index),
		};
	}
	layout(layout, space) {
		let [current, variant, previous] = this.currentLayout;
		if (layout === 0 && current === 0 && space === undefined) {
			if (!previous) return pathless;
			[layout, variant] = previous;
		} else {
			if (layout === 0 && current !== 0) previous = [current, variant];
			variant =
				layout === current && layout !== 0
					? (variant + 1) % this.layouts[layout].length
					: 0;
		}
		this.currentLayout = [layout, variant, previous];
		const focused =
			layout === 0 ? (space ?? this.spaces[0]) : this.spaces[0];
		const spaces =
			layout === 0 ? [focused] : pathless.spaces.slice(0, layout + 1);
		const anchored = this.spaces.filter(
			(space) => space.origin && !spaces.includes(space),
		);
		this.spaces.splice(0, this.spaces.length, ...spaces, ...anchored);
		this.focus(0, this.spaces.includes(focused) ? focused : this.spaces[0]);
		return this.sync();
	}
	focus(direction = 1, target = this.spaces[0]) {
		const index = this.spaces.indexOf(target);
		if (index < 0) return pathless;
		const length = this.spaces.length;
		const offset = (((index + direction) % length) + length) % length;
		this.spaces.push(...this.spaces.splice(0, offset));
		return pathless;
	}
	panel(visible = !this.spaces.includes(pathless.spaces[3])) {
		const panel = pathless.spaces[3];
		const index = this.spaces.indexOf(panel);
		if (visible === index >= 0) return pathless;
		if (visible) this.spaces.push(panel);
		else this.spaces.splice(index, 1);
		if (!this.spaces.length) this.spaces.push(pathless.spaces[0]);
		return this.sync();
	}
	#measure() {
		const { width, height } = pathless.el.getBoundingClientRect();
		Object.assign(pathless.el, { width, height });
	}
	navigate(direction = 1, space = this.spaces[0]) {
		space.index += direction;
		return this.sync();
	}
	sync() {
		const {
			layout: [layout, variant],
			visible,
		} = this.universeState();
		Object.assign(pathless.el.dataset, { layout, variant });
		pathless.spaces.forEach((space) => {
			space.hidden = !visible.includes(space);
		});
		for (const space of visible) {
			const { frames } = space;
			const { length } = frames;
			if (!length) continue;
			space.index = ((space.index % length) + length) % length;
			const { data } = frames[space.index];
			const rendering = this.#rendering;
			this.#rendering = space;
			try {
				pathless.exec(data, space);
			} finally {
				this.#rendering = rendering;
			}
		}
		return pathless;
	}
}
