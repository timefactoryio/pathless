class Layouts {
	static buildLayouts() {
		const el = (inner, fill) =>
			`<svg viewBox="0 0 880 596" style="width:100%;height:100%"><rect width="860" height="576" x="10" y="10" fill="${fill || 'none'}" stroke="#00f" stroke-width="30" rx="16"/>${inner}</svg>`;
		const div = (...d) =>
			d.length
				? `<path stroke="#00f" stroke-width="30" d="${d.join('')}"/>`
				: '';
		const pane = (x, y, w, h) =>
			`<rect width="${w}" height="${h}" x="${x}" y="${y}" fill="#00f"/>`;
		const v = (...d) => el(div(...d));
		const f = (p, ...d) => el(pane(...p) + div(...d));
		const full = () => el('', '#00f');

		const V = 'M440 10v576',
			H = 'M10 298h860',
			HR = 'M440 298h430',
			HL = 'M10 298h430',
			VB = 'M440 298v288',
			VT = 'M440 10v288';
		const LF = [25, 25, 415, 546],
			RF = [455, 25, 415, 546],
			TF = [25, 25, 830, 258],
			BF = [25, 313, 830, 258];
		const TL = [25, 25, 415, 258],
			TR = [455, 25, 415, 258],
			BL = [25, 313, 415, 258],
			BR = [455, 313, 415, 258];
		return {
			0: { v: [v()], f: [[full()]] },
			1: {
				v: [v(V), v(H)],
				f: [
					[f(LF, V), f(RF, V)],
					[f(TF, H), f(BF, H)],
				],
			},
			2: {
				v: [v(V, HR), v(H, VB), v(V, HL), v(H, VT)],
				f: [
					[f(LF, V, HR), f(TR, V, HR), f(BR, V, HR)],
					[f(TF, H, VB), f(BL, H, VB), f(BR, H, VB)],
					[f(RF, V, HL), f(TL, V, HL), f(BL, V, HL)],
					[f(BF, H, VT), f(TL, H, VT), f(TR, H, VT)],
				],
			},
		};
	}
	static buildPanel() {
		const p = Object.assign(document.createElement('div'), {
			id: 'panel',
			innerHTML:
				'<div id="panel-left"></div><div id="panel-right"></div>',
		});
		p.dataset.panel = '1';
		return p;
	}
	constructor() {
		this.prev = null;
		this.panel = Layouts.buildPanel();
		this.layouts = Layouts.buildLayouts();
		this.sync();

		new MutationObserver(() => {
			pathless.spaces.forEach((el) => {
				el.innerHTML = pathless.frames[el.frame];
			});
			pathless.exec(pathless.space);
		}).observe(pathless.universe, { attributeFilter: ['data-layout'] });
	}

	get layout() {
		return +pathless.universe.dataset.layout;
	}
	set layout(v) {
		pathless.universe.dataset.layout = v;
	}
	get variant() {
		return +pathless.universe.dataset.variant;
	}
	set variant(v) {
		pathless.universe.dataset.variant = v;
	}
	get entry() {
		return this.layouts[this.layout];
	}
	get variantSVG() {
		return this.entry.v[this.variant];
	}
	get focusSVG() {
		return this.entry.f[this.variant][pathless.universe.focused];
	}

	#movePanel() {
		if (this.panel.isConnected) pathless.space.appendChild(this.panel);
	}

	sync() {
		const l = this.layout;
		pathless.spaces[0].visible = true;
		pathless.spaces[1].visible = l >= 1;
		pathless.spaces[2].visible = l >= 2;
		if (!pathless.space.visible) {
			pathless.universe.focused = 0;
			pathless.render(0);
		}
		this.#movePanel();
	}

	cycle(layout = this.layout) {
		if (layout === 0 && this.layout === 0) {
			this.layout = this.prev ?? 2;
			this.variant = 0;
		} else if (layout !== this.layout) {
			this.prev = this.layout;
			this.layout = layout;
			this.variant = 0;
		} else {
			this.variant = (this.variant + 1) % this.entry.v.length;
		}
		this.sync();
	}

	focus() {
		do {
			pathless.universe.focused =
				(pathless.universe.focused + 1) % pathless.spaces.length;
		} while (!pathless.space.visible);
		pathless.render(pathless.universe.focused);
		this.sync();
	}

	togglePanel() {
		pathless.space.contains(this.panel)
			? this.panel.remove()
			: pathless.space.appendChild(this.panel);
	}

	get right() {
		return this.panel.querySelector('#panel-right');
	}
	get left() {
		return this.panel.querySelector('#panel-left');
	}
	set left(v) {
		if (!this.left.innerHTML) this.left.innerHTML = v;
	}

	switchPanel(rightContent) {
		this.right.innerHTML = rightContent ?? '';
		this.panel.dataset.panel = rightContent ? '2' : '1';
	}
}
return new Layouts();
