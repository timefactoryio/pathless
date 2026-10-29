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
		let request = this.#cache.get(path);
		if (!request) {
			request = fetch(`${circuit}/${path}`)
				.then((response) => response.bytes())
				.then(Pathless.decode)
				.catch((error) => {
					this.#cache.delete(path);
					throw error;
				});
			this.#cache.set(path, request);
		}
		if (!blobs) return request;
		return request.then((ones) =>
			ones.map((one) => {
				let url = this.#cache.get(one);
				if (!url) {
					const blob = new Blob([one.data], {
						type: one.type,
					});
					url = URL.createObjectURL(blob);
					this.#cache.set(one, url);
				}
				return url;
			}),
		);
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
