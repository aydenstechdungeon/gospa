import { type Rune, type Derived } from "../state.ts";

/**
 * Conditional rendering helper.
 */
export function renderIf<T>(
  condition: Rune<boolean> | Derived<boolean>,
  trueRender: () => T,
  falseRender?: () => T,
): { element: T | null; cleanup: () => void } {
  let current: T | null = null;

  const update = (value: boolean) => {
    if (value) {
      if (!current) {
        current = trueRender();
      }
    } else {
      if (current && falseRender) {
        current = falseRender();
      } else {
        current = null;
      }
    }
  };

  const unsubscribe = condition.subscribe(update);
  update(condition.get());

  return {
    element: current,
    cleanup: unsubscribe,
  };
}

/**
 * List rendering helper with key tracking and reconciliation.
 */
export function renderList<T, K>(
  items: Rune<T[]> | Derived<T[]>,
  render: (item: T, index: number) => Element,
  getKey: (item: T, index: number) => K,
): { container: Element; cleanup: () => void } {
  const containerElement = document.createElement("div");
  const itemMap = new Map<K, { element: Element; index: number }>();

  const update = (newItems: T[]) => {
    const newKeys = new Set<K>();

    newItems.forEach((item, index) => {
      const key = getKey(item, index);
      newKeys.add(key);

      if (!itemMap.has(key)) {
        const element = render(item, index);
        itemMap.set(key, { element, index });
        const refNode = containerElement.children[index] || null;
        containerElement.insertBefore(element, refNode);
      } else {
        const existing = itemMap.get(key)!;
        existing.index = index;
        if (containerElement.children[index] !== existing.element) {
          containerElement.insertBefore(
            existing.element,
            containerElement.children[index] || null,
          );
        }
      }
    });

    itemMap.forEach((value, key) => {
      if (!newKeys.has(key)) {
        value.element.remove();
        itemMap.delete(key);
      }
    });
  };

  const unsubscribe = items.subscribe(update);
  update(items.get());

  return {
    container: containerElement,
    cleanup: () => {
      unsubscribe();
      itemMap.clear();
    },
  };
}
