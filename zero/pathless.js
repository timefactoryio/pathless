class Pathless {
	#cache = new Map();
	#focused = null;
	static #decoder = new TextDecoder();
	static #range = new Range();

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
	}

	get observable() {
		return this.spaces.filter((space) => !space.hidden);
	}
	set observable(spaces) {
		for (const space of this.spaces)
			space.toggleAttribute('hidden', !spaces.includes(space));
		if (!this.#focused?.checkVisibility())
			this.focused = this.spaces.find((space) => space.checkVisibility());
	}
	get focused() {
		return this.#focused;
	}
	set focused(space) {
		if (space && (!this.spaces.includes(space) || !space.checkVisibility()))
			return;
		this.#focused = space ?? null;
		if (space && !space.contains(document.activeElement))
			space.focus({ preventScroll: true });
	}

	static decode(bytes) {
		let offset = 0;
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
			const start = offset;
			offset += length;
			return bytes.subarray(start, offset);
		};
		const ones = () =>
			Array.from({ length: uvarint() }, () => {
				const name = Pathless.#decoder.decode(field());
				const type = Pathless.#decoder.decode(field());
				const data = field();
				return {
					name,
					type,
					data: type.startsWith('text/html')
						? Pathless.#decoder.decode(data)
						: data,
					ones: ones(),
				};
			});
		return ones();
	}

	source(path = '', blobs = false) {
		let source = this.#cache.get(path);
		if (!source) this.#cache.set(path, (source = {}));
		const key = blobs ? 'blobs' : 'data';
		return (source[key] ??= (
			blobs
				? this.source(path).then((ones) =>
						ones.map((one) =>
							URL.createObjectURL(
								new Blob([one.data], { type: one.type }),
							),
						),
					)
				: fetch(`${circuit}/${path}`)
						.then((response) => response.bytes())
						.then(Pathless.decode)
		).catch((error) => {
			if (this.#cache.get(path) === source) this.#cache.delete(path);
			throw error;
		}));
	}

	exec(space = this.focused) {
		const frame = space.frames[space.index];
		if (!frame || space.mounted === frame) return this;
		space.replaceChildren(frame.cloneNode(true));
		space.mounted = frame;
		return this;
	}

	async init() {
		const [frames, panels] = await this.source();
		for (const [pool, group] of [
			[this.frames, frames],
			[this.panels, panels],
		]) {
			pool.length = 0;
			for (const { data } of group.ones)
				pool.push(Pathless.#range.createContextualFragment(data));
		}
		this.#cache.delete('');
		for (const space of this.spaces) this.exec(space);
	}
}
