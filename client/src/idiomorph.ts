/**
 * Idiomorph-based DOM morpher for GoSPA navigation.
 *
 * Uses ID-set matching to correctly handle element reordering, insertion,
 * and deletion — unlike the previous index-based approach.
 *
 * Adapted from https://github.com/bigskysoftware/idiomorph (MIT License)
 * with GoSPA-specific simplifications:
 *   - No head merging (GoSPA handles head via updateHead())
 *   - innerHTML-only morph style
 *   - beforeNodeRemoved hook for island cleanup
 *   - data-gospa-permanent skip support
 */

type NodeCallback = (node: Node) => boolean | void;
type AttributeCallback = (
  attr: string,
  el: Element,
  type: "update" | "remove",
) => boolean | void;

export interface MorphConfig {
  callbacks?: {
    beforeNodeAdded?: NodeCallback;
    afterNodeAdded?: (node: Node) => void;
    beforeNodeMorphed?: (oldNode: Node, newNode: Node) => boolean | void;
    afterNodeMorphed?: (oldNode: Node, newNode: Node) => void;
    beforeNodeRemoved?: NodeCallback;
    afterNodeRemoved?: (node: Node) => void;
    beforeAttributeUpdated?: AttributeCallback;
  };
  ignoreActiveValue?: boolean;
  restoreFocus?: boolean;
}

interface InternalConfig {
  callbacks: {
    beforeNodeAdded: NodeCallback;
    afterNodeAdded: (node: Node) => void;
    beforeNodeMorphed: (oldNode: Node, newNode: Node) => boolean | void;
    afterNodeMorphed: (oldNode: Node, newNode: Node) => void;
    beforeNodeRemoved: NodeCallback;
    afterNodeRemoved: (node: Node) => void;
    beforeAttributeUpdated: AttributeCallback;
  };
  ignoreActiveValue: boolean;
  restoreFocus: boolean;
}

interface MorphContext {
  target: Element;
  config: InternalConfig;
  idMap: Map<Node, Set<string>>;
  persistentIds: Set<string>;
  pantry: HTMLDivElement;
  activeElementAndParents: Element[];
}

const noOp = () => {};

function mergeConfig(cfg: MorphConfig): InternalConfig {
  return {
    callbacks: {
      beforeNodeAdded: cfg.callbacks?.beforeNodeAdded ?? noOp,
      afterNodeAdded: cfg.callbacks?.afterNodeAdded ?? noOp,
      beforeNodeMorphed: cfg.callbacks?.beforeNodeMorphed ?? noOp,
      afterNodeMorphed: cfg.callbacks?.afterNodeMorphed ?? noOp,
      beforeNodeRemoved: cfg.callbacks?.beforeNodeRemoved ?? noOp,
      afterNodeRemoved: cfg.callbacks?.afterNodeRemoved ?? noOp,
      beforeAttributeUpdated: cfg.callbacks?.beforeAttributeUpdated ?? noOp,
    },
    ignoreActiveValue: cfg.ignoreActiveValue ?? false,
    restoreFocus: cfg.restoreFocus ?? true,
  };
}

function createPantry(): HTMLDivElement {
  const pantry = document.createElement("div");
  pantry.hidden = true;
  document.body.insertAdjacentElement("afterend", pantry);
  return pantry;
}

function findIdElements(root: Element): Element[] {
  const elements = Array.from(root.querySelectorAll("[id]"));
  if (root.getAttribute?.("id")) {
    elements.push(root);
  }
  return elements;
}

function findIdElementsInNode(root: Node): Element[] {
  if (root instanceof Element) {
    return findIdElements(root);
  }
  if ("querySelectorAll" in root) {
    return Array.from((root as ParentNode).querySelectorAll("[id]"));
  }
  return [];
}

function createPersistentIds(
  oldElements: Element[],
  newElements: Element[],
): Set<string> {
  const duplicates = new Set<string>();
  const oldMap = new Map<string, string>();

  for (const { id, tagName } of oldElements) {
    if (oldMap.has(id)) {
      duplicates.add(id);
    } else {
      oldMap.set(id, tagName);
    }
  }

  const persistent = new Set<string>();
  for (const { id, tagName } of newElements) {
    if (persistent.has(id)) {
      duplicates.add(id);
    } else if (oldMap.get(id) === tagName) {
      persistent.add(id);
    }
  }

  for (const id of duplicates) {
    persistent.delete(id);
  }
  return persistent;
}

function populateIdMap(
  idMap: Map<Node, Set<string>>,
  persistentIds: Set<string>,
  root: Node,
  elements: Element[],
): void {
  for (const elt of elements) {
    const id = elt.getAttribute("id");
    if (!id || !persistentIds.has(id)) continue;

    let current: Node | null = elt;
    while (current) {
      let idSet = idMap.get(current);
      if (!idSet) {
        idSet = new Set();
        idMap.set(current, idSet);
      }
      idSet.add(id);
      if (current === root) break;
      current = current.parentElement;
    }
  }
}

function createIdMaps(
  oldContent: Element,
  newContent: Node,
): { persistentIds: Set<string>; idMap: Map<Node, Set<string>> } {
  const oldIdElements = findIdElements(oldContent);
  const newIdElements = findIdElementsInNode(newContent);
  const persistentIds = createPersistentIds(oldIdElements, newIdElements);

  const idMap = new Map<Node, Set<string>>();
  populateIdMap(idMap, persistentIds, oldContent, oldIdElements);
  populateIdMap(idMap, persistentIds, newContent, newIdElements);

  return { persistentIds, idMap };
}

function createActiveElementAndParents(root: Element): Element[] {
  const result: Element[] = [];
  let elt: Element | null = document.activeElement as Element | null;
  if (elt && elt.tagName !== "BODY" && root.contains(elt)) {
    while (elt) {
      result.push(elt);
      if (elt === root) break;
      elt = elt.parentElement;
    }
  }
  return result;
}

function createMorphContext(
  target: Element,
  newContent: Node,
  config: MorphConfig,
): MorphContext {
  const mergedConfig = mergeConfig(config);
  const { persistentIds, idMap } = createIdMaps(target, newContent);

  return {
    target,
    config: mergedConfig,
    idMap,
    persistentIds,
    pantry: createPantry(),
    activeElementAndParents: createActiveElementAndParents(target),
  };
}

function isSoftMatch(oldNode: Node, newNode: Node): boolean {
  const oldElt = oldNode as Element;
  const newElt = newNode as Element;
  return (
    oldElt.nodeType === newElt.nodeType &&
    oldElt.tagName === newElt.tagName &&
    (!oldElt.getAttribute?.("id") ||
      oldElt.getAttribute?.("id") === newElt.getAttribute?.("id"))
  );
}

function isIdSetMatch(ctx: MorphContext, oldNode: Node, newNode: Node): boolean {
  const oldSet = ctx.idMap.get(oldNode);
  const newSet = ctx.idMap.get(newNode);
  if (!oldSet || !newSet) return false;
  for (const id of oldSet) {
    if (newSet.has(id)) return true;
  }
  return false;
}

function findBestMatch(
  ctx: MorphContext,
  node: Node,
  startPoint: Node | null,
  endPoint: Node | null,
): Node | null {
  let softMatch: Node | null | undefined = null;
  let nextSibling = node.nextSibling;
  let siblingSoftMatchCount = 0;

  let cursor = startPoint;
  while (cursor && cursor !== endPoint) {
    if (isSoftMatch(cursor, node)) {
      if (isIdSetMatch(ctx, cursor, node)) {
        return cursor;
      }
      if (softMatch === null && !ctx.idMap.has(cursor)) {
        softMatch = cursor;
      }
    }
    if (
      softMatch === null &&
      nextSibling &&
      isSoftMatch(cursor, nextSibling)
    ) {
      siblingSoftMatchCount++;
      nextSibling = nextSibling.nextSibling;
      if (siblingSoftMatchCount >= 2) {
        softMatch = undefined;
      }
    }
    if (ctx.activeElementAndParents.includes(cursor as Element)) break;
    cursor = cursor.nextSibling;
  }

  return softMatch || null;
}

function moveBefore(
  parentNode: Node & { moveBefore?: (element: Node | null, after: Node | null) => void },
  element: Node,
  after: Node | null,
): void {
  if (parentNode.moveBefore) {
    try {
      parentNode.moveBefore(element, after);
    } catch {
      parentNode.insertBefore(element, after);
    }
  } else {
    parentNode.insertBefore(element, after);
  }
}

function removeElementFromAncestorsIdMaps(
  element: Element,
  ctx: MorphContext,
): void {
  const id = element.getAttribute("id");
  let current: Node | null = element.parentNode;
  while (current) {
    const idSet = ctx.idMap.get(current);
    if (idSet) {
      idSet.delete(id!);
      if (!idSet.size) {
        ctx.idMap.delete(current);
      }
    }
    current = current.parentNode;
  }
}

function moveBeforeById(
  parentNode: Element,
  id: string,
  after: Node | null,
  ctx: MorphContext,
): Element {
  const target =
    (ctx.target.getAttribute?.("id") === id && ctx.target) ||
    ctx.target.querySelector(`[id="${id}"]`) ||
    ctx.pantry.querySelector(`[id="${id}"]`);

  if (!target) {
    throw new Error(`Could not find element with id "${id}"`);
  }

  removeElementFromAncestorsIdMaps(target as Element, ctx);
  moveBefore(parentNode, target, after);
  return target as Element;
}

function removeNode(ctx: MorphContext, node: Node): void {
  if (ctx.idMap.has(node)) {
    moveBefore(ctx.pantry, node, null);
  } else {
    if (ctx.config.callbacks.beforeNodeRemoved(node) === false) return;
    node.parentNode?.removeChild(node);
    ctx.config.callbacks.afterNodeRemoved(node);
  }
}

function removeNodesBetween(
  ctx: MorphContext,
  startInclusive: Node,
  endExclusive: Node,
): Node | null {
  let cursor: Node | null = startInclusive;
  while (cursor && cursor !== endExclusive) {
    const temp = cursor;
    cursor = cursor.nextSibling;
    removeNode(ctx, temp);
  }
  return cursor;
}

function ignoreAttribute(
  attr: string,
  el: Element,
  type: "update" | "remove",
  ctx: MorphContext,
): boolean {
  if (
    attr === "value" &&
    ctx.config.ignoreActiveValue &&
    el === document.activeElement
  ) {
    return true;
  }
  return ctx.config.callbacks.beforeAttributeUpdated(attr, el, type) === false;
}

function syncBooleanAttribute(
  oldEl: Element,
  newEl: Element,
  attr: string,
  ctx: MorphContext,
): void {
  const newVal = (newEl as any)[attr];
  const oldVal = (oldEl as any)[attr];
  if (newVal !== oldVal) {
    if (!ignoreAttribute(attr, oldEl, "update", ctx)) {
      (oldEl as any)[attr] = newVal;
    }
    if (newVal) {
      if (!ignoreAttribute(attr, oldEl, "update", ctx)) {
        oldEl.setAttribute(attr, "");
      }
    } else {
      if (!ignoreAttribute(attr, oldEl, "remove", ctx)) {
        oldEl.removeAttribute(attr);
      }
    }
  }
}

function syncInputValue(
  oldEl: Element,
  newEl: Element,
  ctx: MorphContext,
): void {
  if (
    oldEl instanceof HTMLInputElement &&
    newEl instanceof HTMLInputElement &&
    newEl.type !== "file"
  ) {
    const newValue = newEl.value;
    const oldValue = oldEl.value;

    syncBooleanAttribute(oldEl, newEl, "checked", ctx);
    syncBooleanAttribute(oldEl, newEl, "disabled", ctx);

    if (!newEl.hasAttribute("value")) {
      if (!ignoreAttribute("value", oldEl, "remove", ctx)) {
        oldEl.value = "";
        oldEl.removeAttribute("value");
      }
    } else if (oldValue !== newValue) {
      if (!ignoreAttribute("value", oldEl, "update", ctx)) {
        oldEl.setAttribute("value", newValue);
        oldEl.value = newValue;
      }
    }
  } else if (
    oldEl instanceof HTMLOptionElement &&
    newEl instanceof HTMLOptionElement
  ) {
    syncBooleanAttribute(oldEl, newEl, "selected", ctx);
  } else if (
    oldEl instanceof HTMLTextAreaElement &&
    newEl instanceof HTMLTextAreaElement
  ) {
    const newValue = newEl.value;
    const oldValue = oldEl.value;
    if (ignoreAttribute("value", oldEl, "update", ctx)) return;
    if (newValue !== oldValue) {
      oldEl.value = newValue;
    }
    if (oldEl.firstChild && oldEl.firstChild.nodeValue !== newValue) {
      oldEl.firstChild.nodeValue = newValue;
    }
  }
}

function morphAttributes(oldNode: Node, newNode: Node, ctx: MorphContext): void {
  if (oldNode.nodeType !== newNode.nodeType) return;

  if (oldNode.nodeType === 1) {
    const oldEl = oldNode as Element;
    const newEl = newNode as Element;
    const oldAttrs = oldEl.attributes;
    const newAttrs = newEl.attributes;

    for (const attr of Array.from(newAttrs)) {
      if (ignoreAttribute(attr.name, oldEl, "update", ctx)) continue;
      if (oldEl.getAttribute(attr.name) !== attr.value) {
        oldEl.setAttribute(attr.name, attr.value);
      }
    }

    for (let i = oldAttrs.length - 1; i >= 0; i--) {
      const oldAttr = oldAttrs[i];
      if (!oldAttr) continue;
      if (!newEl.hasAttribute(oldAttr.name)) {
        if (ignoreAttribute(oldAttr.name, oldEl, "remove", ctx)) continue;
        oldEl.removeAttribute(oldAttr.name);
      }
    }

    syncInputValue(oldEl, newEl, ctx);
  }

  if (oldNode.nodeType === 3 || oldNode.nodeType === 8) {
    if (oldNode.nodeValue !== newNode.nodeValue) {
      oldNode.nodeValue = newNode.nodeValue;
    }
  }
}

function morphNode(oldNode: Node, newNode: Node, ctx: MorphContext): Node | null {
  if (ctx.config.callbacks.beforeNodeMorphed(oldNode, newNode) === false) {
    return oldNode;
  }

  morphAttributes(oldNode, newNode, ctx);

  if (oldNode.nodeType === 1) {
    const oldEl = oldNode as Element;
    if (oldEl.hasAttribute("data-gospa-permanent")) {
      ctx.config.callbacks.afterNodeMorphed(oldNode, newNode);
      return oldNode;
    }
  }

  morphChildren(ctx, oldNode, newNode);
  ctx.config.callbacks.afterNodeMorphed(oldNode, newNode);
  return oldNode;
}

function createNode(
  oldParent: Element,
  newChild: Node,
  insertionPoint: Node | null,
  ctx: MorphContext,
): Node | null {
  if (ctx.config.callbacks.beforeNodeAdded(newChild) === false) return null;

  if (ctx.idMap.has(newChild)) {
    const newEmptyChild = document.createElement(
      (newChild as Element).tagName,
    );
    oldParent.insertBefore(newEmptyChild, insertionPoint);
    morphNode(newEmptyChild, newChild, ctx);
    ctx.config.callbacks.afterNodeAdded(newEmptyChild);
    return newEmptyChild;
  } else {
    const clonedChild = document.importNode(newChild, true);
    oldParent.insertBefore(clonedChild, insertionPoint);
    ctx.config.callbacks.afterNodeAdded(clonedChild);
    return clonedChild;
  }
}

function morphChildren(
  ctx: MorphContext,
  oldParent: Node,
  newParent: Node,
  insertionPoint: Node | null = null,
  endPoint: Node | null = null,
): void {
  if (
    oldParent instanceof HTMLTemplateElement &&
    newParent instanceof HTMLTemplateElement
  ) {
    oldParent = oldParent.content;
    newParent = newParent.content;
  }

  insertionPoint = insertionPoint || oldParent.firstChild;

  for (const newChild of Array.from(newParent.childNodes)) {
    if (insertionPoint && insertionPoint !== endPoint) {
      const bestMatch = findBestMatch(ctx, newChild, insertionPoint, endPoint);
      if (bestMatch) {
        if (bestMatch !== insertionPoint) {
          removeNodesBetween(ctx, insertionPoint, bestMatch);
        }
        morphNode(bestMatch, newChild, ctx);
        insertionPoint = bestMatch.nextSibling;
        continue;
      }
    }

    if (newChild instanceof Element) {
      const newChildId = newChild.getAttribute("id");
      if (newChildId && ctx.persistentIds.has(newChildId)) {
        const movedChild = moveBeforeById(
          oldParent as Element,
          newChildId,
          insertionPoint,
          ctx,
        );
        morphNode(movedChild, newChild, ctx);
        insertionPoint = movedChild.nextSibling;
        continue;
      }
    }

    const insertedNode = createNode(
      oldParent as Element,
      newChild,
      insertionPoint,
      ctx,
    );
    if (insertedNode) {
      insertionPoint = insertedNode.nextSibling;
    }
  }

  while (insertionPoint && insertionPoint !== endPoint) {
    const temp = insertionPoint;
    insertionPoint = insertionPoint.nextSibling;
    removeNode(ctx, temp);
  }
}

function saveAndRestoreFocus<T>(ctx: MorphContext, fn: () => T): T {
  if (!ctx.config.restoreFocus) return fn();

  const activeEl = document.activeElement as
    | HTMLInputElement
    | HTMLTextAreaElement
    | null;

  if (
    !(
      activeEl instanceof HTMLInputElement ||
      activeEl instanceof HTMLTextAreaElement
    )
  ) {
    return fn();
  }

  const { id: activeId, selectionStart, selectionEnd } = activeEl;
  const results = fn();

  if (activeId && activeId !== document.activeElement?.getAttribute("id")) {
    const restored = ctx.target.querySelector(`[id="${activeId}"]`);
    if (restored) (restored as HTMLElement).focus();
  }
  if (activeEl && selectionEnd) {
    activeEl.setSelectionRange(selectionStart, selectionEnd);
  }

  return results;
}

/**
 * Morph the innerHTML of a target element with new HTML content.
 * Uses ID-set matching for correct handling of element reordering.
 */
export function morphInnerHTML(
  target: Element,
  newHTML: string,
  config: MorphConfig = {},
): void {
  const template = document.createElement("template");
  template.innerHTML = newHTML;

  const ctx = createMorphContext(target, template.content, {
    ...config,
  });

  saveAndRestoreFocus(ctx, () => {
    morphChildren(ctx, target, template.content);
  });

  ctx.pantry.remove();
}
