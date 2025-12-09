class Keybinds {
	constructor() {
		this.keys = {};
		const handler = (event, pressedProp, callbackProp) => {
			const key = this.keys[event.key.toLowerCase()];
			if (!key) return;
			event.preventDefault();
			key.pressed = pressedProp;
			if (typeof key[callbackProp] === 'function')
				key[callbackProp](event);
		};
		window.addEventListener('keydown', (e) => handler(e, true, 'onDown'));
		window.addEventListener('keyup', (e) => handler(e, false, 'onUp'));
	}

	onKey(key, onDown, onUp) {
		this.keys[key.toLowerCase()] = { pressed: false, onDown, onUp };
		return this;
	}

	off(key) {
		delete this.keys[key.toLowerCase()];
		return this;
	}

	keyboard() {
		return Object.entries(this.keys).map(([key, { pressed }]) => ({
			key,
			pressed,
		}));
	}
}

class Source {
	constructor() {
		this.cache = new Map();
		this.totalFrames = null;
	}

	async fetch(url) {
		if (this.cache.has(url)) return this.cache.get(url);

		const promise = fetch(url).then(async (response) => {
			if (!response.ok) throw new Error(`HTTP ${response.status}`);
			const ct = response.headers.get('content-type') || '';
			let data;
			if (ct.includes('json')) {
				data = await response.json();
			} else if (ct.startsWith('text/')) {
				data = await response.text();
			} else {
				data = URL.createObjectURL(await response.blob());
			}
			return { data, headers: response.headers };
		});

		this.cache.set(url, promise);
		return promise;
	}

	async fetchFrame(index = 0) {
		const frameUrl =
			index === 0
				? `${window.apiUrl}/frame`
				: `${window.apiUrl}/frame/${index}`;

		const { data, headers } = await this.fetch(frameUrl);

		if (this.totalFrames === null && headers) {
			this.totalFrames =
				parseInt(headers.get('X-Frames') || '0', 10) || null;
		}

		return data;
	}

	async init() {
		const { data } = await this.fetch(`${window.apiUrl}/frames`);
		this.frames = data || [];
		await this.render();
	}
}

class Pathless {
	constructor() {
		this.layout = [0, 0];
		this.prev = null;
		this.focus = 0;
		this.universe = new Universe();
	}

	render(htmls = []) {
		this.universe.render(htmls, this.layout, this.focus);
	}

	fullscreen() {
		if (this.layout[0] === 0 && this.prev) {
			this.layout = [...this.prev];
			this.prev = null;
		} else if (this.layout[0] !== 0) {
			this.prev = [...this.layout];
			this.layout = [0, 0];
		}
	}

	layouts(i) {
		this.prev = null;
		this.focus = Math.min(this.focus, i);
		this.layout =
			this.layout[0] === i
				? [i, (this.layout[1] + 1) % [0, 2, 4][i]]
				: [i, 0];
	}

	setFocus(index) {
		this.focus = Math.max(
			0,
			Math.min(index, Universe.count(this.layout) - 1)
		);
	}
}

class Universe {
	constructor() {
		this.universe = Array.from({ length: 3 }, (_, i) => {
			const space = document.querySelector(`.s${i}`);
			return { space, frame: space ? space.firstElementChild : null };
		});
	}

	static count(layout) {
		return layout[0] + 1;
	}

	render(htmls = [], layout = [0, 0], focus = 0) {
		const count = Universe.count(layout);
		for (let i = 0; i < count; i++) {
			const u = this.universe[i];
			const html = htmls[i] || '';
			const tmp = document.createElement('div');
			tmp.innerHTML = html;
			['script', 'style'].forEach((tag) => {
				tmp.querySelectorAll(tag).forEach((old) => {
					const n = document.createElement(tag);
					[...old.attributes].forEach((a) =>
						n.setAttribute(a.name, a.value)
					);
					n.textContent = old.textContent;
					old.replaceWith(n);
				});
			});
			u.space.replaceChildren(...tmp.childNodes);
		}
		this.display(layout, focus);
	}

	display(layout = [0, 0], focus = 0) {
		const count = Universe.count(layout);
		requestAnimationFrame(() => {
			this.universe.forEach((u, i) => {
				if (layout[0] === 0) {
					u.space.style.display = i === focus ? 'flex' : 'none';
					u.space.style.flex = '1 1 100%';
				} else if (i < count) {
					u.space.style.display = 'flex';
					u.space.style.flex = '1 1 0';
				} else {
					u.space.style.display = 'none';
				}
			});
		});
	}
}
