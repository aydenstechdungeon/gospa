// Minified reactive primitives - property names shortened for size reduction
// _value -> _v, _subscribers -> _s, _dependencies -> _d, _dirty -> _y, etc.

export type Unsubscribe = () => void;
export type Subscriber<T> = (value: T, oldValue: T) => void;
export type EffectFn = () => void | (() => void);
export type ComputeFn<T> = () => T;

let _id = 0;
let _eid = 0;
let _batch = 0;
let _pending: Set<Notifier> = new Set();

interface Notifier { notify(): void; }

let _cur: Effect | null = null;
const _stack: Effect[] = [];

export function batch(fn: () => void): void {
	_batch++;
	try { fn(); }
	finally {
		_batch--;
		if (_batch === 0) {
			const p = [..._pending];
			_pending.clear();
			p.forEach(n => n.notify());
		}
	}
}

// Rune - core reactive primitive (minified properties)
export class Rune<T> implements Notifier {
	private _v: T;
	private readonly _id: number;
	private readonly _s: Set<Subscriber<T>> = new Set();
	private _y: boolean = false;

	constructor(initialValue: T) {
		this._v = initialValue;
		this._id = ++_id;
	}

	get value(): T {
		this._track();
		return this._v;
	}

	set value(n: T) {
		if (this._eq(this._v, n)) return;
		const o = this._v;
		this._v = n;
		this._y = true;
		this._notify(o);
	}

	get(): T { this._track(); return this._v; }
	set(n: T): void { this.value = n; }
	update(fn: (c: T) => T): void { this.value = fn(this._v); }

	subscribe(fn: Subscriber<T>): Unsubscribe {
		this._s.add(fn);
		return () => this._s.delete(fn);
	}

	private _notify(o: T): void {
		if (_batch > 0) { _pending.add(this); return; }
		this.notify();
	}

	notify(): void {
		const v = this._v;
		this._s.forEach(fn => fn(v, this._v));
	}

	private _eq(a: T, b: T): boolean {
		return Object.is(a, b) || JSON.stringify(a) === JSON.stringify(b);
	}

	private _track(): void {
		if (_cur) _cur.addDep(this as unknown as Rune<unknown>);
	}

	toJSON(): { id: number; value: T } {
		return { id: this._id, value: this._v };
	}
}

export function rune<T>(initialValue: T): Rune<T> {
	return new Rune(initialValue);
}

// Derived - computed state (minified)
export class Derived<T> implements Notifier {
	private _v: T;
	private readonly _c: ComputeFn<T>;
	private readonly _d: Set<Rune<unknown>> = new Set();
	private readonly _s: Set<Subscriber<T>> = new Set();
	private _y: boolean = true;
	private _z: boolean = false;

	constructor(compute: ComputeFn<T>) {
		this._c = compute;
		this._v = undefined as T;
		this._recompute();
	}

	get value(): T {
		if (this._y) this._recompute();
		this._track();
		return this._v;
	}

	get(): T { return this.value; }

	subscribe(fn: Subscriber<T>): Unsubscribe {
		this._s.add(fn);
		return () => this._s.delete(fn);
	}

	private _recompute(): void {
		const old = new Set(this._d);
		this._d.clear();

		const prev = _cur;
		const col = { addDep: (r: Rune<unknown>) => { this._d.add(r); } } as Effect;
		_cur = col;
		try {
			this._v = this._c();
			this._y = false;
		} finally {
			_cur = prev;
		}

		this._d.forEach(dep => {
			if (!old.has(dep)) {
				dep.subscribe(() => {
					this._y = true;
					this._notify();
				});
			}
		});
	}

	private _notify(): void {
		if (_batch > 0) { _pending.add(this); return; }
		this.notify();
	}

	notify(): void {
		if (this._y) this._recompute();
		const v = this._v;
		this._s.forEach(fn => fn(v, this._v));
	}

	private _track(): void {
		if (_cur) _cur.addDep(this as unknown as Rune<unknown>);
	}

	dispose(): void {
		this._z = true;
		this._d.clear();
		this._s.clear();
	}
}

export function derived<T>(compute: ComputeFn<T>): Derived<T> {
	return new Derived(compute);
}

// Effect - side effects (minified)
export class Effect implements Notifier {
	private readonly _fn: EffectFn;
	private _cl: (() => void) | void;
	private readonly _d: Set<Rune<unknown>> = new Set();
	private readonly _id: number;
	private _a: boolean = true;
	private _z: boolean = false;

	constructor(fn: EffectFn) {
		this._fn = fn;
		this._id = ++_eid;
		this._cl = undefined;
		this._run();
	}

	private _run(): void {
		if (!this._a || this._z) return;

		if (this._cl) { this._cl(); this._cl = undefined; }
		this._d.clear();

		_stack.push(this);
		_cur = this;

		try {
			this._cl = this._fn();
		} finally {
			_stack.pop();
			_cur = _stack[_stack.length - 1] || null;
		}
	}

	addDep(rune: Rune<unknown>): void { this._d.add(rune); }
	notify(): void { this._run(); }
	pause(): void { this._a = false; }
	resume(): void { this._a = true; this._run(); }

	dispose(): void {
		if (this._cl) this._cl();
		this._z = true;
		this._d.clear();
	}
}

export function effect(fn: EffectFn): Effect {
	return new Effect(fn);
}

// Watch multiple runes
export function watch<T>(
	sources: Rune<T> | Rune<T>[],
	callback: (values: T | T[], oldValues: T | T[]) => void
): Unsubscribe {
	const arr = Array.isArray(sources) ? sources : [sources];
	const unsubs: Unsubscribe[] = [];

	arr.forEach(src => {
		unsubs.push(src.subscribe((v, o) => {
			const vs = arr.map(s => s.get());
			const os = arr.map(s => o);
			callback(Array.isArray(sources) ? vs : vs[0], Array.isArray(sources) ? os : os[0]);
		}));
	});

	return () => unsubs.forEach(u => u());
}

// StateMap - collection of named runes (minified)
export class StateMap {
	private readonly _r: Map<string, Rune<unknown>> = new Map();

	set<T>(key: string, value: T): Rune<T> {
		const ex = this._r.get(key);
		if (ex) {
			ex.set(value);
			return ex as Rune<T>;
		}
		const r = new Rune(value);
		this._r.set(key, r as unknown as Rune<unknown>);
		return r;
	}

	get<T>(key: string): Rune<T> | undefined {
		return this._r.get(key) as Rune<T> | undefined;
	}

	has(key: string): boolean { return this._r.has(key); }
	delete(key: string): boolean { return this._r.delete(key); }
	clear(): void { this._r.clear(); }

	toJSON(): Record<string, unknown> {
		const r: Record<string, unknown> = {};
		this._r.forEach((rune, key) => { r[key] = rune.get(); });
		return r;
	}

	fromJSON(data: Record<string, unknown>): void {
		Object.entries(data).forEach(([key, value]) => {
			if (this._r.has(key)) {
				(this._r.get(key) as Rune<unknown>).set(value);
			} else {
				this.set(key, value);
			}
		});
	}
}

export function stateMap(): StateMap {
	return new StateMap();
}

// untrack - execute without tracking
export function untrack<T>(fn: () => T): T {
	const prev = _cur;
	_cur = null;
	try { return fn(); }
	finally { _cur = prev; }
}

// tracking - check if in reactive context
export function tracking(): boolean {
	return _cur !== null;
}
