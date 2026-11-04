class Pathless {
	#cache = new Map();
	static #decoder = new TextDecoder();
	static #range = new Range();

	constructor() {
		this.el = document.getElementById('universe');
		this.frames = [];
		this.panels = [];
		this.spaces = Array.from(
			this.el.querySelectorAll(':scope > .space'),
		).map((space, index) =>
			Object.assign(space, {
				frames: index === 3 ? this.panels : this.frames,
				index: 0,
				states: new Map(),
			}),
		);
	}

	get observable() {
		return this.spaces.filter((space) => !space.hidden);
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

	exec(data, space) {
		const fragment = Pathless.#range.createContextualFragment(data);
		space?.replaceChildren(fragment);
		return space ? this : fragment;
	}

	async init() {
		const [frames, panels] = await this.source();
		this.frames.splice(0, this.frames.length, ...frames.ones);
		this.panels.splice(0, this.panels.length, ...panels.ones);
		new Input();
		new Universe();
	}
}
