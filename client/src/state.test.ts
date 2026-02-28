import { describe, it, expect, beforeEach } from 'bun:test';
import {
  Rune,
  Derived,
  Effect,
  StateMap,
  batch,
  effect,
  watch,
  rune,
  derived,
  runeRaw,
  snapshot,
  untrack,
  tracking,
  Resource,
  resource,
  enableDisposalTracking,
  disposeAll,
} from './state';

describe('Rune', () => {
  beforeEach(() => {
    // Clean up any tracked disposables
    disposeAll();
  });

  it('should create a rune with initial value', () => {
    const r = new Rune(42);
    expect(r.get()).toBe(42);
  });

  it('should update value', () => {
    const r = new Rune(0);
    r.set(10);
    expect(r.get()).toBe(10);
  });

  it('should use value getter/setter', () => {
    const r = new Rune(5);
    r.value = 20;
    expect(r.value).toBe(20);
  });

  it('should update with function', () => {
    const r = new Rune(5);
    r.update(v => v * 2);
    expect(r.get()).toBe(10);
  });

  it('should notify subscribers', async () => {
    const r = new Rune(0);
    const values: number[] = [];

    const unsub = r.subscribe((value) => {
      values.push(value);
    });

    // Initial subscription doesn't push current value, only changes
    r.set(1);
    await new Promise(resolve => setTimeout(resolve, 10));
    expect(values).toContain(1);

    r.set(2);
    await new Promise(resolve => setTimeout(resolve, 10));
    expect(values).toContain(2);

    r.set(3);
    await new Promise(resolve => setTimeout(resolve, 10));
    expect(values).toContain(3);

    unsub();
  });

  it('should unsubscribe correctly', async () => {
    const r = new Rune(0);
    const values: number[] = [];

    const unsub = r.subscribe((value) => {
      values.push(value);
    });

    r.set(1);
    await new Promise(resolve => setTimeout(resolve, 10));
    expect(values.length).toBeGreaterThanOrEqual(0); // May or may not have fired

    unsub();
    const countBefore = values.length;
    r.set(2);
    await new Promise(resolve => setTimeout(resolve, 10));

    // Should not receive updates after unsubscribe
    expect(values.length).toBe(countBefore);
  });

  it('should not notify on same value', async () => {
    const r = new Rune(5);
    let count = 0;

    const unsub = r.subscribe(() => {
      count++;
    });

    r.set(5);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(count).toBe(0);
    unsub();
  });

  it('should handle object equality', async () => {
    const r = new Rune({ a: 1 });
    let count = 0;

    const unsub = r.subscribe(() => {
      count++;
    });

    r.set({ a: 1 });
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(count).toBe(0);

    r.set({ a: 2 });
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(count).toBe(1);
    unsub();
  });

  it('should be disposable', () => {
    const r = new Rune(5);
    expect(r.isDisposed()).toBe(false);
    r.dispose();
    expect(r.isDisposed()).toBe(true);
  });

  it('should convert to JSON', () => {
    const r = new Rune(42);
    const json = r.toJSON();
    expect(json.value).toBe(42);
    expect(json.id).toBeDefined();
  });
});

describe('rune factory', () => {
  it('should create rune using factory', () => {
    const r = rune(42);
    expect(r.get()).toBe(42);
  });
});

describe('Derived', () => {
  it('should compute initial value', () => {
    const a = new Rune(5);
    const d = new Derived(() => a.get() * 2);
    expect(d.get()).toBe(10);
  });

  it('should recompute when dependencies change', async () => {
    const a = new Rune(5);
    const d = new Derived(() => a.get() * 2);

    expect(d.get()).toBe(10);

    a.set(10);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(d.get()).toBe(20);
  });

  it('should notify subscribers on change', async () => {
    const a = new Rune(5);
    const d = new Derived(() => a.get() * 2);
    const values: number[] = [];

    const unsub = d.subscribe((value) => {
      values.push(value);
    });

    a.set(10);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(values).toContain(20);
    unsub();
  });

  it('should be disposable', () => {
    const a = new Rune(5);
    const d = new Derived(() => a.get() * 2);
    d.dispose();
    expect(d.isDisposed()).toBe(true);
  });
});

describe('derived factory', () => {
  it('should create derived using factory', () => {
    const a = rune(5);
    const d = derived(() => a.get() * 2);
    expect(d.get()).toBe(10);
  });
});

describe('Effect', () => {
  it('should run immediately', () => {
    let ran = false;
    const e = new Effect(() => {
      ran = true;
      return undefined;
    });
    expect(ran).toBe(true);
    e.dispose();
  });

  it('should re-run on dependency change', async () => {
    const a = new Rune(0);
    let count = 0;

    const e = new Effect(() => {
      a.get();
      count++;
      return undefined;
    });

    expect(count).toBe(1);

    a.set(1);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(count).toBe(2);
    e.dispose();
  });

  it('should call cleanup before re-run', async () => {
    const a = new Rune(0);
    let cleanupCount = 0;

    const e = new Effect(() => {
      a.get();
      return () => {
        cleanupCount++;
      };
    });

    a.set(1);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(cleanupCount).toBe(1);
    e.dispose();
  });

  it('should pause and resume', async () => {
    const a = new Rune(0);
    let count = 0;

    const e = new Effect(() => {
      a.get();
      count++;
      return undefined;
    });

    e.pause();
    a.set(1);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(count).toBe(1);

    e.resume();
    expect(count).toBe(2);
    e.dispose();
  });

  it('should be disposable', () => {
    const e = new Effect(() => undefined);
    expect(e.isDisposed()).toBe(false);
    e.dispose();
    expect(e.isDisposed()).toBe(true);
  });
});

describe('effect factory', () => {
  it('should create effect using factory', () => {
    let ran = false;
    const e = effect(() => {
      ran = true;
      return undefined;
    });
    expect(ran).toBe(true);
    e.dispose();
  });
});

describe('batch', () => {
  it('should batch updates', async () => {
    const a = new Rune(0);
    const values: number[] = [];

    const unsub = a.subscribe((value) => {
      values.push(value);
    });

    batch(() => {
      a.set(1);
      a.set(2);
      a.set(3);
    });

    await new Promise(resolve => setTimeout(resolve, 10));

    // Should have at least the final value
    expect(values[values.length - 1]).toBe(3);
    unsub();
  });
});

describe('watch', () => {
  it('should watch a single rune', async () => {
    const a = new Rune(0);
    const values: number[] = [];

    const unsub = watch(a, (value) => {
      values.push(value as number);
    });

    a.set(1);
    a.set(2);

    await new Promise(resolve => setTimeout(resolve, 10));

    expect(values).toContain(1);
    expect(values).toContain(2);
    unsub();
  });

  it('should watch multiple runes', async () => {
    const a = new Rune(0);
    const b = new Rune(0);
    const calls: number[][] = [];

    const unsub = watch([a, b], (values) => {
      calls.push(values as number[]);
    });

    a.set(1);
    await new Promise(resolve => setTimeout(resolve, 10));

    b.set(2);
    await new Promise(resolve => setTimeout(resolve, 10));

    expect(calls.length).toBeGreaterThan(0);
    unsub();
  });
});

describe('StateMap', () => {
  it('should set and get values', () => {
    const sm = new StateMap();
    sm.set('count', 42);

    const r = sm.get('count');
    expect(r?.get()).toBe(42);
  });

  it('should check if key exists', () => {
    const sm = new StateMap();
    sm.set('key', 'value');

    expect(sm.has('key')).toBe(true);
    expect(sm.has('nonexistent')).toBe(false);
  });

  it('should delete keys', () => {
    const sm = new StateMap();
    sm.set('key', 'value');
    sm.delete('key');

    expect(sm.has('key')).toBe(false);
  });

  it('should convert to JSON', () => {
    const sm = new StateMap();
    sm.set('count', 42);
    sm.set('name', 'test');

    const json = sm.toJSON();
    expect(json.count).toBe(42);
    expect(json.name).toBe('test');
  });

  it('should load from JSON', () => {
    const sm = new StateMap();
    sm.fromJSON({ count: 42, name: 'test' });

    expect(sm.get('count')?.get()).toBe(42);
    expect(sm.get('name')?.get()).toBe('test');
  });

  it('should update existing rune when setting same key', () => {
    const sm = new StateMap();
    sm.set('count', 0);

    const r1 = sm.get('count');
    sm.set('count', 42);
    const r2 = sm.get('count');

    expect(r1).toBe(r2);
    expect(r2?.get()).toBe(42);
  });
});

describe('runeRaw', () => {
  it('should create raw rune', () => {
    const r = runeRaw({ a: 1 });
    expect(r.get()).toEqual({ a: 1 });
  });

  it('should not do deep equality', async () => {
    const r = runeRaw({ a: 1 });
    let count = 0;

    const unsub = r.subscribe(() => {
      count++;
    });

    r.set({ a: 1 });
    await new Promise(resolve => setTimeout(resolve, 10));

    // Raw rune uses Object.is, so objects won't be equal
    expect(count).toBe(1);
    unsub();
  });

  it('should create snapshot', () => {
    const r = runeRaw([1, 2, 3]);
    const snap = r.snapshot();
    expect(snap).toEqual([1, 2, 3]);

    // Snapshot should be a copy
    snap.push(4);
    expect(r.get()).toEqual([1, 2, 3]);
  });
});

describe('snapshot', () => {
  it('should snapshot a rune', () => {
    const r = new Rune({ a: 1 });
    const s = snapshot(r);
    expect(s).toEqual({ a: 1 });
  });

  it('should snapshot a raw rune', () => {
    const r = runeRaw({ a: 1 });
    const s = snapshot(r);
    expect(s).toEqual({ a: 1 });
  });

  it('should snapshot a plain object', () => {
    const obj = { a: 1 };
    const s = snapshot(obj);
    expect(s).toEqual({ a: 1 });
  });

  it('should snapshot a plain array', () => {
    const arr = [1, 2, 3];
    const s = snapshot(arr);
    expect(s).toEqual([1, 2, 3]);
  });
});

describe('untrack', () => {
  it('should not track dependencies', async () => {
    const a = new Rune(5);
    let effectRan = 0;

    const e = new Effect(() => {
      untrack(() => a.get());
      effectRan++;
      return undefined;
    });

    a.set(10);
    await new Promise(resolve => setTimeout(resolve, 10));

    // Effect should not have re-run
    expect(effectRan).toBe(1);
    e.dispose();
  });
});

describe('tracking', () => {
  it('should return false outside effect', () => {
    expect(tracking()).toBe(false);
  });

  it('should return true inside effect', () => {
    let wasTracking = false;

    const e = new Effect(() => {
      wasTracking = tracking();
      return undefined;
    });

    expect(wasTracking).toBe(true);
    e.dispose();
  });
});

describe('Resource', () => {
  it('should start in idle state', () => {
    const r = new Resource(async () => 'data');
    expect(r.isIdle).toBe(true);
    expect(r.isPending).toBe(false);
    expect(r.isSuccess).toBe(false);
    expect(r.isError).toBe(false);
  });

  it('should fetch data', async () => {
    const r = new Resource(async () => 'data');
    await r.refetch();

    expect(r.isSuccess).toBe(true);
    expect(r.data).toBe('data');
  });

  it('should handle errors', async () => {
    const r = new Resource(async () => {
      throw new Error('failed');
    });

    await r.refetch();

    expect(r.isError).toBe(true);
    expect(r.error).toBeDefined();
  });

  it('should reset to idle', async () => {
    const r = new Resource(async () => 'data');
    await r.refetch();

    r.reset();

    expect(r.isIdle).toBe(true);
    expect(r.data).toBeUndefined();
  });
});

describe('resource factory', () => {
  it('should create resource using factory', async () => {
    const r = resource(async () => 'data');
    await r.refetch();

    expect(r.data).toBe('data');
  });
});

describe('disposal tracking', () => {
  it('should enable and disable tracking', () => {
    enableDisposalTracking(true);
    enableDisposalTracking(false);
    // Just verify it doesn't throw
    expect(true).toBe(true);
  });
});
