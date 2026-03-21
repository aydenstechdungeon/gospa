// GoSPA Accessibility Enhancements
// Provides screen reader announcements and ARIA utilities

/**
 * Accessibility configuration
 */
export interface A11yConfig {
  /** Enable screen reader announcements (default: true) */
  announceNavigation?: boolean;
  /** Announce state changes (default: false) */
  announceStateChanges?: boolean;
  /** Politeness level for announcements (default: 'polite') */
  politeness?: "polite" | "assertive";
}

/**
 * Screen reader announcer
 * Creates a live region for announcing dynamic content changes
 */
export class ScreenReaderAnnouncer {
  private container: HTMLElement | null = null;
  private config: Required<A11yConfig>;
  private announceTimer: ReturnType<typeof setTimeout> | null = null;
  private pendingAnnouncements: string[] = [];

  constructor(config: A11yConfig = {}) {
    this.config = {
      announceNavigation: config.announceNavigation ?? true,
      announceStateChanges: config.announceStateChanges ?? false,
      politeness: config.politeness ?? "polite",
    };

    if (typeof document !== "undefined") {
      this.init();
    }
  }

  /**
   * Initialize the announcer container
   */
  private init(): void {
    // Create container if it doesn't exist
    this.container = document.getElementById("gospa-announcer");
    if (!this.container) {
      this.container = document.createElement("div");
      this.container.id = "gospa-announcer";
      this.container.setAttribute("aria-live", this.config.politeness);
      this.container.setAttribute("aria-atomic", "true");
      this.container.setAttribute("role", "status");
      this.container.style.cssText = `
				position: absolute;
				width: 1px;
				height: 1px;
				padding: 0;
				margin: -1px;
				overflow: hidden;
				clip: rect(0, 0, 0, 0);
				white-space: nowrap;
				border: 0;
			`;
      document.body.appendChild(this.container);
    }
  }

  /**
   * Announce a message to screen readers
   */
  announce(message: string, priority?: "polite" | "assertive"): void {
    if (!this.container) {
      this.init();
    }

    // Update politeness if needed
    if (priority && priority !== this.config.politeness) {
      this.container?.setAttribute("aria-live", priority);
    }

    // Clear any pending announcements
    if (this.announceTimer) {
      clearTimeout(this.announceTimer);
    }

    // Queue announcement
    this.pendingAnnouncements.push(message);

    // Debounce announcements to avoid overwhelming screen readers
    this.announceTimer = setTimeout(() => {
      const announcement = this.pendingAnnouncements.join(". ");
      this.pendingAnnouncements = [];

      if (this.container) {
        // Clear previous content
        this.container.textContent = "";

        // Set new content after a brief delay to ensure screen readers pick it up
        requestAnimationFrame(() => {
          if (this.container) {
            this.container.textContent = announcement;
          }
        });
      }

      // Reset politeness
      if (priority && priority !== this.config.politeness) {
        this.container?.setAttribute("aria-live", this.config.politeness);
      }
    }, 100);
  }

  /**
   * Announce navigation change
   */
  announceNavigation(path: string, title?: string): void {
    if (!this.config.announceNavigation) return;

    const message = title ? `Navigated to ${title}` : `Navigated to ${path}`;

    this.announce(message);
  }

  /**
   * Announce state change
   */
  announceStateChange(key: string, value: unknown): void {
    if (!this.config.announceStateChanges) return;

    const valueStr =
      typeof value === "object" ? JSON.stringify(value) : String(value);

    this.announce(`${key} changed to ${valueStr}`);
  }

  /**
   * Announce loading state
   */
  announceLoading(message: string = "Loading"): void {
    this.announce(message, "assertive");
  }

  /**
   * Announce error
   */
  announceError(message: string): void {
    this.announce(`Error: ${message}`, "assertive");
  }

  /**
   * Announce success
   */
  announceSuccess(message: string): void {
    this.announce(message);
  }

  /**
   * Destroy the announcer
   */
  destroy(): void {
    if (this.announceTimer) {
      clearTimeout(this.announceTimer);
    }

    if (this.container) {
      this.container.remove();
      this.container = null;
    }

    this.pendingAnnouncements = [];
  }
}

/**
 * ARIA utilities
 */
export const aria = {
  /**
   * Set ARIA attributes on an element
   */
  setAttributes(
    element: Element,
    attributes: Record<string, string | boolean | null>,
  ): void {
    for (const [key, value] of Object.entries(attributes)) {
      if (value === null || value === false) {
        element.removeAttribute(key);
      } else if (value === true) {
        element.setAttribute(key, "");
      } else {
        element.setAttribute(key, String(value));
      }
    }
  },

  /**
   * Make an element focusable
   */
  makeFocusable(element: Element, tabIndex: number = 0): void {
    element.setAttribute("tabindex", String(tabIndex));
  },

  /**
   * Set ARIA label
   */
  label(element: Element, label: string): void {
    element.setAttribute("aria-label", label);
  },

  /**
   * Set ARIA describedby
   */
  describe(element: Element, descriptionId: string): void {
    element.setAttribute("aria-describedby", descriptionId);
  },

  /**
   * Set ARIA expanded state
   */
  expanded(element: Element, expanded: boolean): void {
    element.setAttribute("aria-expanded", String(expanded));
  },

  /**
   * Set ARIA hidden state
   */
  hidden(element: Element, hidden: boolean): void {
    if (hidden) {
      element.setAttribute("aria-hidden", "true");
    } else {
      element.removeAttribute("aria-hidden");
    }
  },

  /**
   * Set ARIA selected state
   */
  selected(element: Element, selected: boolean): void {
    element.setAttribute("aria-selected", String(selected));
  },

  /**
   * Set ARIA checked state
   */
  checked(element: Element, checked: boolean | "mixed"): void {
    element.setAttribute("aria-checked", String(checked));
  },

  /**
   * Set ARIA disabled state
   */
  disabled(element: Element, disabled: boolean): void {
    element.setAttribute("aria-disabled", String(disabled));
  },

  /**
   * Set ARIA busy state
   */
  busy(element: Element, busy: boolean): void {
    element.setAttribute("aria-busy", String(busy));
  },

  /**
   * Set ARIA live region
   */
  live(element: Element, politeness: "polite" | "assertive" | "off"): void {
    element.setAttribute("aria-live", politeness);
  },

  /**
   * Create a description element
   */
  createDescription(id: string, text: string): HTMLElement {
    const el = document.createElement("div");
    el.id = id;
    el.className = "gospa-sr-only";
    el.textContent = text;
    el.style.cssText = `
			position: absolute;
			width: 1px;
			height: 1px;
			padding: 0;
			margin: -1px;
			overflow: hidden;
			clip: rect(0, 0, 0, 0);
			white-space: nowrap;
			border: 0;
		`;
    return el;
  },
};

/**
 * Focus management utilities
 */
export const focus = {
  /**
   * Trap focus within an element
   */
  trap(element: Element): () => void {
    const focusableSelectors = [
      "a[href]",
      "button:not([disabled])",
      "input:not([disabled])",
      "textarea:not([disabled])",
      "select:not([disabled])",
      '[tabindex]:not([tabindex="-1"])',
    ].join(", ");

    const focusableElements = Array.from(
      element.querySelectorAll(focusableSelectors),
    ) as HTMLElement[];

    if (focusableElements.length === 0) return () => {};

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    const handleKeyDown = (event: Event) => {
      const keyEvent = event as KeyboardEvent;
      if (keyEvent.key !== "Tab") return;

      if (keyEvent.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstElement) {
          keyEvent.preventDefault();
          lastElement.focus();
        }
      } else {
        // Tab
        if (document.activeElement === lastElement) {
          keyEvent.preventDefault();
          firstElement.focus();
        }
      }
    };

    element.addEventListener("keydown", handleKeyDown);

    // Focus first element
    firstElement.focus();

    // Return cleanup function
    return () => {
      element.removeEventListener("keydown", handleKeyDown);
    };
  },

  /**
   * Restore focus to an element
   */
  restore(element: Element | null): void {
    if (element && element instanceof HTMLElement) {
      element.focus();
    }
  },

  /**
   * Save current focus and return restore function
   */
  save(): () => void {
    const activeElement = document.activeElement;
    return () => this.restore(activeElement);
  },

  /**
   * Move focus to an element
   */
  moveTo(element: Element): void {
    if (element instanceof HTMLElement) {
      element.focus();
    }
  },
};

/**
 * Create a screen reader announcer
 */
export function createAnnouncer(config?: A11yConfig): ScreenReaderAnnouncer {
  return new ScreenReaderAnnouncer(config);
}

/**
 * Global announcer instance
 */
let globalAnnouncer: ScreenReaderAnnouncer | null = null;

/**
 * Get or create the global announcer
 */
export function getAnnouncer(config?: A11yConfig): ScreenReaderAnnouncer {
  if (!globalAnnouncer) {
    globalAnnouncer = new ScreenReaderAnnouncer(config);
  }
  return globalAnnouncer;
}

/**
 * Destroy the global announcer
 */
export function destroyAnnouncer(): void {
  if (globalAnnouncer) {
    globalAnnouncer.destroy();
    globalAnnouncer = null;
  }
}

/**
 * Quick announce helper
 */
export function announce(
  message: string,
  priority?: "polite" | "assertive",
): void {
  getAnnouncer().announce(message, priority);
}
