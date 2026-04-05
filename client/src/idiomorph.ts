/*
 * Idiomorph - A small, but capable, DOM morphing library.
 * https://github.com/bigskysoftware/idiomorph
 *
 * Modified for GoSPA TypeScript environment.
 * Fixed outerHTML normalization to prevent layout-container-loss bug.
 */

export interface IdiomorphConfig {
    morphStyle?: 'outerHTML' | 'innerHTML';
    ignoreActive?: boolean;
    ignoreActiveValue?: boolean;
    restoreFocus?: boolean;
    callbacks?: {
        beforeNodeAdded?: (node: Node) => boolean;
        afterNodeAdded?: (node: Node) => void;
        beforeNodeMorphed?: (oldNode: Node, newNode: Node) => boolean;
        afterNodeMorphed?: (oldNode: Node, newNode: Node) => void;
        beforeNodeRemoved?: (node: Node) => boolean;
        afterNodeRemoved?: (node: Node) => void;
        beforeAttributeUpdated?: (name: string, el: Element, action: 'update' | 'remove') => boolean;
    };
    head?: {
        style?: 'merge' | 'append' | 'morph' | 'none';
        shouldPreserve?: (el: Element) => boolean;
        shouldReAppend?: (el: Element) => boolean;
        shouldRemove?: (el: Element) => boolean;
        afterHeadMorphed?: (el: Element, info: { added: Node[], kept: Element[], removed: Element[] }) => void;
    };
}

const noOp = () => { };

const defaults: Required<IdiomorphConfig> = {
    morphStyle: "outerHTML",
    callbacks: {
        beforeNodeAdded: noOp as any,
        afterNodeAdded: noOp as any,
        beforeNodeMorphed: noOp as any,
        afterNodeMorphed: noOp as any,
        beforeNodeRemoved: noOp as any,
        afterNodeRemoved: noOp as any,
        beforeAttributeUpdated: noOp as any,
    },
    head: {
        style: "merge",
        shouldPreserve: (elt) => elt.getAttribute("im-preserve") === "true",
        shouldReAppend: (elt) => elt.getAttribute("im-re-append") === "true",
        shouldRemove: noOp as any,
        afterHeadMorphed: noOp as any,
    },
    restoreFocus: true,
    ignoreActive: false,
    ignoreActiveValue: false
};

export const Idiomorph = (function () {

    class SlicedParentNode {
        originalNode: Node;
        realParentNode: Element;
        previousSibling: Node | null;
        nextSibling: Node | null;

        constructor(node: Node) {
            this.originalNode = node;
            this.realParentNode = node.parentNode as Element;
            this.previousSibling = node.previousSibling;
            this.nextSibling = node.nextSibling;
        }

        get childNodes(): Node[] {
            const nodes = [];
            let cursor = this.previousSibling ? this.previousSibling.nextSibling : this.realParentNode.firstChild;
            while (cursor && cursor !== this.nextSibling) {
                nodes.push(cursor);
                cursor = cursor.nextSibling;
            }
            return nodes;
        }

        get firstChild(): Node | null {
            return this.previousSibling ? this.previousSibling.nextSibling : this.realParentNode.firstChild;
        }

        querySelectorAll(selector: string): Element[] {
            return this.childNodes.reduce((results, node) => {
                if (node instanceof Element) {
                    if (node.matches(selector)) results.push(node);
                    const nodeList = node.querySelectorAll(selector);
                    for (let i = 0; i < nodeList.length; i++) results.push(nodeList[i]);
                }
                return results;
            }, [] as Element[]);
        }

        insertBefore(node: Node, referenceNode: Node | null): Node {
            return this.realParentNode.insertBefore(node, referenceNode);
        }

        append(node: Node): void {
            this.realParentNode.appendChild(node);
        }
        
        removeChild(node: Node): Node {
            return this.realParentNode.removeChild(node);
        }
    }

    function morph(oldNode: any, newContent: any, config: IdiomorphConfig = {}): Node[] {
        oldNode = normalizeElement(oldNode);
        const newNode = normalizeParent(newContent);
        const ctx = createMorphContext(oldNode, newNode, config);

        const morphedNodes = saveAndRestoreFocus(ctx, () => {
            if (ctx.morphStyle === "innerHTML") {
                morphChildren(ctx, oldNode, newNode);
                return Array.from(oldNode.childNodes);
            } else {
                return morphOuterHTML(ctx, oldNode, newNode);
            }
        });

        if (ctx.pantry.parentNode) ctx.pantry.remove();
        return morphedNodes as Node[];
    }

    function normalizeElement(elt: any) {
        if (elt instanceof Document) return elt.documentElement;
        return elt;
    }

    function normalizeParent(newContent: any): any {
        if (newContent instanceof Document) return newContent.documentElement;
        if (typeof newContent === 'string') {
            const template = document.createElement('template');
            template.innerHTML = newContent;
            return template.content;
        }
        if (newContent instanceof Node) {
            if (newContent.parentNode) {
                return new SlicedParentNode(newContent);
            } else {
                const dummy = document.createElement('div');
                dummy.appendChild(newContent);
                return dummy;
            }
        }
        if (newContent instanceof HTMLCollection || Array.isArray(newContent)) {
            const dummy = document.createElement('div');
            for (const node of Array.from(newContent)) dummy.appendChild(node);
            return dummy;
        }
        return newContent;
    }

    function createMorphContext(oldNode: any, newNode: any, config: IdiomorphConfig): any {
        const mergedConfig = { ...defaults, ...config };
        mergedConfig.callbacks = { ...defaults.callbacks, ...(config.callbacks || {}) };
        mergedConfig.head = { ...defaults.head, ...(config.head || {}) };

        const idMap: Map<Node, Set<string>> = new Map();
        const persistentIds: Set<string> = new Set();
        
        // We need to use the "real" nodes for ID mapping
        const oldRoot = (oldNode instanceof SlicedParentNode) ? oldNode.originalNode : oldNode;
        const newRoot = (newNode instanceof SlicedParentNode) ? newNode.originalNode : newNode;

        populateIdMap(oldRoot, idMap, persistentIds);
        populateIdMap(newRoot, idMap, persistentIds);

        return {
            target: oldRoot,
            newContent: newRoot,
            config: mergedConfig,
            morphStyle: mergedConfig.morphStyle,
            ignoreActive: mergedConfig.ignoreActive,
            ignoreActiveValue: mergedConfig.ignoreActiveValue,
            restoreFocus: mergedConfig.restoreFocus,
            idMap,
            persistentIds,
            callbacks: mergedConfig.callbacks,
            head: mergedConfig.head,
            pantry: document.createElement('div'),
            activeElementAndParents: getActiveElementAndParents()
        };
    }

    function getActiveElementAndParents() {
        const active = document.activeElement;
        const parents = [];
        let cursor = active;
        while (cursor) {
            parents.push(cursor);
            cursor = cursor.parentElement;
        }
        return parents;
    }

    function populateIdMap(node: any, idMap: Map<any, Set<string>>, persistentIds: Set<string>) {
        if (node instanceof Element || node instanceof DocumentFragment) {
            const set = new Set<string>();
            if (node instanceof Element && node.id) {
                set.add(node.id);
                persistentIds.add(node.id);
            }
            const childrenWithIds = (node instanceof Element ? node : (node as any)).querySelectorAll?.('[id]') || [];
            for (const child of childrenWithIds) {
                set.add(child.id);
                persistentIds.add(child.id);
            }
            idMap.set(node, set);
        }
    }

    function saveAndRestoreFocus(ctx: any, fn: any) {
        if (!ctx.restoreFocus) return fn();
        const activeElement = document.activeElement as any;
        const selectionStart = activeElement?.selectionStart;
        const selectionEnd = activeElement?.selectionEnd;
        const activeId = activeElement?.id;

        const results = fn();

        if (activeId) {
            const found = ctx.target.querySelector(`[id="${CSS.escape(activeId)}"]`);
            if (found && found !== document.activeElement) {
                found.focus();
                if (selectionStart !== undefined && found.setSelectionRange) {
                    found.setSelectionRange(selectionStart, selectionEnd);
                }
            }
        }

        return results;
    }

    function morphOuterHTML(ctx: any, oldNode: any, newNode: any) {
        const oldParent = normalizeParent(oldNode);
        morphChildren(ctx, oldParent, newNode, oldNode, oldNode.nextSibling);
        return Array.from(oldParent.childNodes);
    }

    function morphChildren(ctx: any, oldParent: any, newParent: any, insertionPoint: any = null, endPoint: any = null) {
        if (oldParent instanceof HTMLTemplateElement && newParent instanceof HTMLTemplateElement) {
            oldParent = (oldParent as any).content;
            newParent = (newParent as any).content;
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

            if (newChild instanceof Element && newChild.id && ctx.persistentIds.has(newChild.id)) {
                const movedChild = moveBeforeById(oldParent, newChild.id, insertionPoint, ctx);
                morphNode(movedChild, newChild, ctx);
                insertionPoint = movedChild.nextSibling;
                continue;
            }

            const insertedNode = createNode(oldParent, newChild, insertionPoint, ctx);
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

    function findBestMatch(ctx: any, node: any, startPoint: any, endPoint: any) {
        let softMatch = null;
        let cursor = startPoint;
        while (cursor && cursor !== endPoint) {
            if (isSoftMatch(cursor, node)) {
                if (isIdSetMatch(ctx, cursor, node)) return cursor;
                if (softMatch === null && !ctx.idMap.has(cursor)) softMatch = cursor;
            }
            if (ctx.activeElementAndParents.includes(cursor)) break;
            cursor = cursor.nextSibling;
        }
        return softMatch;
    }

    function isSoftMatch(oldNode: any, newNode: any) {
        return oldNode.nodeType === newNode.nodeType &&
            oldNode.tagName === newNode.tagName &&
            (!oldNode.id || oldNode.id === newNode.id);
    }

    function isIdSetMatch(ctx: any, oldNode: any, newNode: any) {
        const oldSet = ctx.idMap.get(oldNode);
        const newSet = ctx.idMap.get(newNode);
        if (!oldSet || !newSet) return false;
        for (const id of oldSet) if (newSet.has(id)) return true;
        return false;
    }

    function removeNode(ctx: any, node: any) {
        if (ctx.idMap.has(node)) {
            ctx.pantry.appendChild(node);
        } else {
            if (ctx.callbacks.beforeNodeRemoved(node) === false) return;
            node.parentNode?.removeChild(node);
            ctx.callbacks.afterNodeRemoved(node);
        }
    }

    function removeNodesBetween(ctx: any, start: any, end: any) {
        let cursor = start;
        while (cursor && cursor !== end) {
            const temp = cursor;
            cursor = cursor.nextSibling;
            removeNode(ctx, temp);
        }
    }

    function createNode(parent: any, newNode: any, insertionPoint: any, ctx: any) {
        if (ctx.callbacks.beforeNodeAdded(newNode) === false) return null;
        const clone = newNode.cloneNode(true);
        parent.insertBefore(clone, insertionPoint);
        ctx.callbacks.afterNodeAdded(clone);
        return clone;
    }

    function moveBeforeById(parent: any, id: string, after: any, ctx: any) {
        const target = ctx.target.querySelector?.(`[id="${CSS.escape(id)}"]`) || ctx.pantry.querySelector?.(`[id="${CSS.escape(id)}"]`);
        if (target) {
            parent.insertBefore(target, after);
        }
        return target;
    }

    function morphNode(oldNode: any, newNode: any, ctx: any) {
        if (ctx.ignoreActive && oldNode === document.activeElement) return;
        if (ctx.callbacks.beforeNodeMorphed(oldNode, newNode) === false) return;

        if (oldNode.nodeType === Node.TEXT_NODE || oldNode.nodeType === Node.COMMENT_NODE) {
            if (oldNode.nodeValue !== newNode.nodeValue) oldNode.nodeValue = newNode.nodeValue;
        } else if (oldNode instanceof Element && newNode instanceof Element) {
            morphAttributes(oldNode, newNode, ctx);
            morphChildren(ctx, oldNode, newNode);
        }

        ctx.callbacks.afterNodeMorphed(oldNode, newNode);
    }

    function morphAttributes(oldNode: Element, newNode: Element, ctx: any) {
        for (const attr of Array.from(newNode.attributes)) {
            if (ctx.callbacks.beforeAttributeUpdated(attr.name, oldNode, 'update') !== false) {
                if (oldNode.getAttribute(attr.name) !== attr.value) {
                    oldNode.setAttribute(attr.name, attr.value);
                }
            }
        }
        for (const attr of Array.from(oldNode.attributes)) {
            if (!newNode.hasAttribute(attr.name)) {
                if (ctx.callbacks.beforeAttributeUpdated(attr.name, oldNode, 'remove') !== false) {
                    oldNode.removeAttribute(attr.name);
                }
            }
        }
        
        // Sync input values
        if (oldNode instanceof HTMLInputElement && newNode instanceof HTMLInputElement) {
            if (oldNode.value !== newNode.value) oldNode.value = newNode.value;
            if (oldNode.checked !== newNode.checked) oldNode.checked = newNode.checked;
        } else if (oldNode instanceof HTMLTextAreaElement && newNode instanceof HTMLTextAreaElement) {
            if (oldNode.value !== newNode.value) oldNode.value = newNode.value;
        } else if (oldNode instanceof HTMLSelectElement && newNode instanceof HTMLSelectElement) {
            if (oldNode.value !== newNode.value) oldNode.value = newNode.value;
        }
    }

    return {
        morph,
        defaults
    };
})();
