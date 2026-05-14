class Input {
	constructor() {
		this.el = pathless.universe;
		this.listen();
	}

	emit(key, phase) {
		this.el.dispatchEvent(
			new CustomEvent('input', { detail: { key, phase } }),
		);
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
		this.el.addEventListener('pointerdown', (e) => {
			const key = e.target.closest('[data-key]')?.dataset.key;
			if (!key) return;
			this.el.setPointerCapture(e.pointerId);
			this.emit(key, 'down');
		});
		this.el.addEventListener('pointerup', up);
		this.el.addEventListener('pointercancel', up);
	}
}
return new Input();
