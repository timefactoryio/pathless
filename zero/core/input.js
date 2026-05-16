class Input {
	constructor() {
		this.binds = new Map();
		this.listen();
	}

	bind(key, { down, up, label } = {}) {
		this.binds.set(key, { down, up, label: label ?? '' });
	}

	unbind(key) {
		this.binds.delete(key);
	}

	release() {
		this.binds.clear();
	}

	emit(key, phase) {
		this.binds.get(key)?.[phase]?.();
	}

	listen() {
		window.addEventListener('keydown', (e) => {
			if (e.key === 'Tab') e.preventDefault();
			if (!e.repeat) this.emit(e.key, 'down');
		});
		window.addEventListener('keyup', (e) => this.emit(e.key, 'up'));

		const up = (e) => {
			const key = e.target.closest('[data-key]')?.dataset.key;
			if (key) this.emit(key, 'up');
		};
		document.addEventListener('pointerdown', (e) => {
			const key = e.target.closest('[data-key]')?.dataset.key;
			if (!key) return;
			this.emit(key, 'down');
		});
		document.addEventListener('pointerup', up);
		document.addEventListener('pointercancel', up);
	}
}
return new Input();
