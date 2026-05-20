function buildLayoutSVGS() {
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
	return [
		[[v(), full()]],
		[
			[v(V), f(LF, V), f(RF, V)],
			[v(H), f(TF, H), f(BF, H)],
		],
		[
			[v(V, HR), f(LF, V, HR), f(TR, V, HR), f(BR, V, HR)],
			[v(H, VB), f(TF, H, VB), f(BL, H, VB), f(BR, H, VB)],
			[v(V, HL), f(RF, V, HL), f(TL, V, HL), f(BL, V, HL)],
			[v(H, VT), f(BF, H, VT), f(TL, H, VT), f(TR, H, VT)],
		],
	];
}
