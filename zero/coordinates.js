class Coordinates {
	static zero(x = null, y = null, z = null) {
		return [x, y, z];
	}
	constructor() {
		this.origin = Coordinates.zero(0, 0);
		this.threshold = 0.05;
	}
	measure(el) {
		const r = el.getBoundingClientRect();
		this.L = r.left;
		this.T = r.top;
		this.W = r.width || 1;
		this.H = r.height || 1;
	}
	at(clientX, clientY) {
		const u = (clientX - this.L) / this.W,
			v = (clientY - this.T) / this.H;
		return Coordinates.zero(2 * u - 1, 1 - 2 * v, Date.now());
	}
	track(clientX, clientY) {
		const pair = {
			start: this.at(clientX, clientY),
			end: Coordinates.zero(),
			path: [],
			_dir: null,
		};
		pair.move = (x, y) => {
			const point = this.at(x, y);
			const dir = this.direction(
				this._last(pair.path, pair.start),
				point,
			);
			if (dir && dir !== pair._dir) {
				pair.path.push({ point, direction: dir });
				pair._dir = dir;
			}
		};
		pair.close = (endX, endY) => {
			pair.end = this.at(endX, endY);
			const direction = this.direction(pair.start, pair.end);
			return {
				start: pair.start,
				end: pair.end,
				delta: this.delta(pair.start, pair.end),
				distance: this.distance(pair.start, pair.end),
				duration: pair.end[2] - pair.start[2],
				velocity: this.velocity(pair.start, pair.end),
				terminal: this.terminal(pair.path, pair.start, pair.end),
				axis: this.axis(pair.start, pair.end),
				direction,
				type: this.type(direction, pair.path),
				from: this.fromOrigin(pair.start),
				path: pair.path,
			};
		};
		return pair;
	}
	delta(a, b) {
		return Coordinates.zero(b[0] - a[0], b[1] - a[1], b[2] - a[2]);
	}
	distance(a, b) {
		return Math.abs(b[0] - a[0]) + Math.abs(b[1] - a[1]);
	}
	axis(a, b) {
		return Math.abs(b[0] - a[0]) >= Math.abs(b[1] - a[1]) ? 'x' : 'y';
	}
	direction(a, b) {
		const dx = b[0] - a[0],
			dy = b[1] - a[1];
		if (Math.abs(dx) + Math.abs(dy) < this.threshold) return null;
		return Math.abs(dx) >= Math.abs(dy)
			? dx >= 0
				? 'right'
				: 'left'
			: dy >= 0
				? 'up'
				: 'down';
	}
	velocity(a, b) {
		const dt = b[2] - a[2];
		return dt > 0 ? this.distance(a, b) / dt : 0;
	}
	terminal(path, start, end) {
		return this.velocity(this._last(path, start), end);
	}
	type(direction, path) {
		return direction === null
			? 'tap'
			: path.length > 1
				? 'gesture'
				: 'swipe';
	}
	fromOrigin(coord) {
		return this.delta(this.origin, coord);
	}
	_last(path, fallback) {
		return path.length ? path[path.length - 1].point : fallback;
	}
}
