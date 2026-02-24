/**
 * Client-side error overlay for development.
 * Displays errors in a user-friendly overlay with stack traces and source code.
 */

// Error information received from server or captured locally
interface ErrorData {
	message: string;
	type: string;
	stack?: StackFrame[];
	file?: string;
	line?: number;
	column?: number;
	codeSnippet?: string;
	timestamp: number;
	request?: {
		method: string;
		url: string;
		headers?: Record<string, string>;
		query?: Record<string, string>;
	};
	cause?: ErrorData;
}

interface StackFrame {
	file: string;
	line: number;
	function: string;
	source?: string;
}

interface ErrorOverlayOptions {
	theme?: 'dark' | 'light';
	editor?: 'code' | 'idea' | 'sublime';
	showStack?: boolean;
	showRequest?: boolean;
	showCode?: boolean;
}

const defaultOptions: ErrorOverlayOptions = {
	theme: 'dark',
	editor: 'code',
	showStack: true,
	showRequest: true,
	showCode: true,
};

/**
 * ErrorOverlay manages the display of errors in development mode.
 */
export class ErrorOverlay {
	private container: HTMLElement | null = null;
	private options: ErrorOverlayOptions;
	private currentError: ErrorData | null = null;

	constructor(options: ErrorOverlayOptions = {}) {
		this.options = { ...defaultOptions, ...options };
		this.setupGlobalHandlers();
	}

	/**
	 * Set up global error handlers to catch unhandled errors.
	 */
	private setupGlobalHandlers(): void {
		if (typeof window === 'undefined') return;

		// Handle uncaught errors
		window.addEventListener('error', (event) => {
			this.showError({
				message: event.message,
				type: 'Error',
				file: event.filename,
				line: event.lineno,
				column: event.colno,
				timestamp: Date.now(),
			});
			event.preventDefault();
		});

		// Handle unhandled promise rejections
		window.addEventListener('unhandledrejection', (event) => {
			const error = event.reason;
			this.showError({
				message: error?.message || String(error),
				type: error?.constructor?.name || 'UnhandledRejection',
				stack: this.parseStack(error?.stack),
				timestamp: Date.now(),
			});
			event.preventDefault();
		});
	}

	/**
	 * Display an error in the overlay.
	 */
	showError(error: ErrorData): void {
		this.currentError = error;
		
		if (!this.container) {
			this.createContainer();
		}

		this.render();
		this.show();
	}

	/**
	 * Hide and clear the error overlay.
	 */
	hide(): void {
		if (this.container) {
			this.container.remove();
			this.container = null;
		}
		this.currentError = null;
	}

	/**
	 * Create the overlay container element.
	 */
	private createContainer(): void {
		this.container = document.createElement('div');
		this.container.id = 'gospa-error-overlay';
		this.container.setAttribute('role', 'dialog');
		this.container.setAttribute('aria-modal', 'true');
		this.container.setAttribute('aria-labelledby', 'gospa-error-title');
		document.body.appendChild(this.container);
	}

	/**
	 * Show the overlay.
	 */
	private show(): void {
		if (this.container) {
			this.container.style.display = 'block';
		}
	}

	/**
	 * Parse a stack trace string into structured frames.
	 */
	private parseStack(stack?: string): StackFrame[] {
		if (!stack) return [];

		const frames: StackFrame[] = [];
		const lines = stack.split('\n');

		for (const line of lines) {
			// Match patterns like "    at functionName (file:line:col)" or "    at file:line:col"
			const match = line.match(/^\s*at\s+(?:(.+?)\s+\()?(.+):(\d+):(\d+)\)?$/);
			if (match) {
				frames.push({
					function: match[1] || '<anonymous>',
					file: match[2],
					line: parseInt(match[3], 10),
				});
			}
		}

		return frames;
	}

	/**
	 * Render the error overlay content.
	 */
	private render(): void {
		if (!this.container || !this.currentError) return;

		const error = this.currentError;
		const theme = this.options.theme || 'dark';

		this.container.innerHTML = `
			<style>${this.getStyles(theme)}</style>
			<div class="gospa-overlay-backdrop" onclick="this.parentElement.remove()"></div>
			<div class="gospa-overlay-container">
				<div class="gospa-overlay-header">
					<div class="gospa-error-type">${this.escapeHtml(error.type)}</div>
					<h1 id="gospa-error-title" class="gospa-error-message">${this.escapeHtml(error.message)}</h1>
					${error.file ? `
						<div class="gospa-error-location">
							<span>📍</span>
							<a href="${this.buildEditorURL(error.file, error.line || 1)}" title="Open in editor">
								${this.escapeHtml(error.file)}${error.line ? `:${error.line}` : ''}
							</a>
						</div>
					` : ''}
					<div class="gospa-overlay-actions">
						<button class="gospa-btn gospa-btn-primary" onclick="navigator.clipboard.writeText(document.querySelector('.gospa-error-message').textContent)">
							📋 Copy Error
						</button>
						<button class="gospa-btn gospa-btn-secondary" onclick="location.reload()">
							🔄 Reload
						</button>
						<button class="gospa-btn gospa-btn-secondary" onclick="this.closest('#gospa-error-overlay').remove()">
							✕ Close
						</button>
					</div>
				</div>

				${error.request && this.options.showRequest ? this.renderRequest(error.request) : ''}

				${error.stack && this.options.showStack ? this.renderStack(error.stack) : ''}

				${error.cause ? this.renderCause(error.cause) : ''}
			</div>
		`;
	}

	/**
	 * Render request information section.
	 */
	private renderRequest(request: NonNullable<ErrorData['request']>): string {
		return `
			<div class="gospa-section">
				<div class="gospa-section-header">
					<span>🌐</span> Request
				</div>
				<div class="gospa-section-content">
					<div class="gospa-request-info">
						<div class="gospa-request-row">
							<div class="gospa-request-key">Method</div>
							<div class="gospa-request-value">${this.escapeHtml(request.method)}</div>
						</div>
						<div class="gospa-request-row">
							<div class="gospa-request-key">URL</div>
							<div class="gospa-request-value">${this.escapeHtml(request.url)}</div>
						</div>
						${request.query ? Object.entries(request.query).map(([key, value]) => `
							<div class="gospa-request-row">
								<div class="gospa-request-key">Query[${this.escapeHtml(key)}]</div>
								<div class="gospa-request-value">${this.escapeHtml(value)}</div>
							</div>
						`).join('') : ''}
					</div>
				</div>
			</div>
		`;
	}

	/**
	 * Render stack trace section.
	 */
	private renderStack(stack: StackFrame[]): string {
		if (!stack.length) {
			return `
				<div class="gospa-section">
					<div class="gospa-section-header">
						<span>📚</span> Stack Trace
					</div>
					<div class="gospa-section-content">
						<div class="gospa-empty">No stack trace available</div>
					</div>
				</div>
			`;
		}

		const frames = stack.slice(0, 15).map((frame, index) => `
			<div class="gospa-stack-frame" data-index="${index}">
				<div class="gospa-stack-frame-header">
					<div class="gospa-stack-function">${this.escapeHtml(frame.function)}</div>
				</div>
				<div class="gospa-stack-file">
					<a href="${this.buildEditorURL(frame.file, frame.line)}" title="Open in editor">
						${this.escapeHtml(frame.file)}:${frame.line}
					</a>
				</div>
			</div>
		`).join('');

		return `
			<div class="gospa-section">
				<div class="gospa-section-header">
					<span>📚</span> Stack Trace
				</div>
				<div class="gospa-section-content">
					${frames}
				</div>
			</div>
		`;
	}

	/**
	 * Render error cause chain.
	 */
	private renderCause(cause: ErrorData): string {
		const causes: string[] = [];
		let current: ErrorData | undefined = cause;

		while (current) {
			causes.push(`
				<div class="gospa-cause-item">
					<div class="gospa-cause-type">${this.escapeHtml(current.type)}</div>
					<div class="gospa-cause-message">${this.escapeHtml(current.message)}</div>
				</div>
			`);
			current = current.cause;
		}

		return `
			<div class="gospa-section">
				<div class="gospa-section-header">
					<span>🔗</span> Caused By
				</div>
				<div class="gospa-section-content">
					<div class="gospa-cause-chain">
						${causes.join('')}
					</div>
				</div>
			</div>
		`;
	}

	/**
	 * Build a URL to open a file in an editor.
	 */
	private buildEditorURL(file: string, line: number): string {
		const editor = this.options.editor || 'code';

		switch (editor) {
			case 'code':
				return `vscode://file/${file}:${line}`;
			case 'idea':
				return `idea://open?file=${file}&line=${line}`;
			case 'sublime':
				return `subl://open?url=file://${file}&line=${line}`;
			default:
				return `vscode://file/${file}:${line}`;
		}
	}

	/**
	 * Escape HTML special characters.
	 */
	private escapeHtml(str: string): string {
		return str
			.replace(/&/g, '\x26amp;')
			.replace(/</g, '\x26lt;')
			.replace(/>/g, '\x26gt;')
			.replace(/"/g, '\x26quot;')
			.replace(/'/g, '\x26#39;');
	}

	/**
	 * Get CSS styles for the overlay.
	 */
	private getStyles(theme: string): string {
		const colors = theme === 'light' ? {
			bgPrimary: '#ffffff',
			bgSecondary: '#f5f5f5',
			bgTertiary: '#ebebeb',
			textPrimary: '#1a1a1a',
			textSecondary: '#666666',
			textMuted: '#999999',
			border: '#e0e0e0',
			codeBg: '#f8f8f8',
			accent: '#dc3545',
			accentHover: '#c82333',
		} : {
			bgPrimary: '#1a1a1a',
			bgSecondary: '#242424',
			bgTertiary: '#2a2a2a',
			textPrimary: '#ffffff',
			textSecondary: '#a0a0a0',
			textMuted: '#666666',
			border: '#333333',
			codeBg: '#1e1e1e',
			accent: '#ff4444',
			accentHover: '#ff6666',
		};

		return `
			#gospa-error-overlay {
				position: fixed;
				top: 0;
				left: 0;
				right: 0;
				bottom: 0;
				z-index: 99999;
				font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
				line-height: 1.6;
			}
			.gospa-overlay-backdrop {
				position: absolute;
				top: 0;
				left: 0;
				right: 0;
				bottom: 0;
				background: rgba(0, 0, 0, 0.7);
				backdrop-filter: blur(4px);
			}
			.gospa-overlay-container {
				position: relative;
				max-width: 1200px;
				margin: 20px auto;
				padding: 0 20px;
				max-height: calc(100vh - 40px);
				overflow-y: auto;
			}
			.gospa-overlay-header {
				background: ${colors.bgSecondary};
				border: 1px solid ${colors.border};
				border-radius: 8px;
				padding: 24px;
				margin-bottom: 20px;
			}
			.gospa-error-type {
				font-size: 12px;
				color: ${colors.accent};
				text-transform: uppercase;
				letter-spacing: 1px;
				margin-bottom: 8px;
			}
			.gospa-error-message {
				font-size: 24px;
				font-weight: 600;
				margin: 0 0 16px 0;
				color: ${colors.textPrimary};
				word-break: break-word;
			}
			.gospa-error-location {
				font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
				font-size: 14px;
				color: ${colors.textSecondary};
				background: ${colors.codeBg};
				padding: 8px 12px;
				border-radius: 4px;
				display: inline-flex;
				align-items: center;
				gap: 8px;
			}
			.gospa-error-location a {
				color: ${colors.textSecondary};
				text-decoration: none;
			}
			.gospa-error-location a:hover {
				color: ${colors.accent};
			}
			.gospa-overlay-actions {
				display: flex;
				gap: 12px;
				margin-top: 16px;
			}
			.gospa-btn {
				padding: 8px 16px;
				border-radius: 4px;
				font-size: 14px;
				cursor: pointer;
				border: none;
				transition: all 0.15s;
			}
			.gospa-btn-primary {
				background: ${colors.accent};
				color: white;
			}
			.gospa-btn-primary:hover {
				background: ${colors.accentHover};
			}
			.gospa-btn-secondary {
				background: ${colors.bgTertiary};
				color: ${colors.textPrimary};
				border: 1px solid ${colors.border};
			}
			.gospa-btn-secondary:hover {
				background: ${colors.bgSecondary};
			}
			.gospa-section {
				background: ${colors.bgSecondary};
				border: 1px solid ${colors.border};
				border-radius: 8px;
				margin-bottom: 20px;
				overflow: hidden;
			}
			.gospa-section-header {
				background: ${colors.bgTertiary};
				padding: 12px 16px;
				font-weight: 600;
				font-size: 14px;
				border-bottom: 1px solid ${colors.border};
				display: flex;
				align-items: center;
				gap: 8px;
				color: ${colors.textPrimary};
			}
			.gospa-section-content {
				padding: 16px;
			}
			.gospa-stack-frame {
				padding: 12px;
				border-bottom: 1px solid ${colors.border};
				cursor: pointer;
				transition: background 0.15s;
			}
			.gospa-stack-frame:hover {
				background: ${colors.bgTertiary};
			}
			.gospa-stack-frame:last-child {
				border-bottom: none;
			}
			.gospa-stack-function {
				font-family: 'SF Mono', Monaco, monospace;
				font-size: 13px;
				color: ${colors.textPrimary};
			}
			.gospa-stack-file {
				font-family: 'SF Mono', Monaco, monospace;
				font-size: 12px;
				color: ${colors.textSecondary};
				margin-top: 4px;
			}
			.gospa-stack-file a {
				color: ${colors.textSecondary};
				text-decoration: none;
			}
			.gospa-stack-file a:hover {
				color: ${colors.accent};
			}
			.gospa-request-info {
				font-family: 'SF Mono', Monaco, monospace;
				font-size: 13px;
			}
			.gospa-request-row {
				display: flex;
				padding: 8px 0;
				border-bottom: 1px solid ${colors.border};
			}
			.gospa-request-row:last-child {
				border-bottom: none;
			}
			.gospa-request-key {
				min-width: 120px;
				color: ${colors.textSecondary};
			}
			.gospa-request-value {
				color: ${colors.textPrimary};
				word-break: break-all;
			}
			.gospa-cause-chain {
				padding-left: 20px;
				border-left: 2px solid ${colors.border};
			}
			.gospa-cause-item {
				padding: 12px;
				background: ${colors.bgTertiary};
				border-radius: 4px;
				margin-bottom: 8px;
			}
			.gospa-cause-type {
				font-size: 11px;
				color: ${colors.accent};
				text-transform: uppercase;
				letter-spacing: 0.5px;
			}
			.gospa-cause-message {
				font-size: 14px;
				margin-top: 4px;
				color: ${colors.textPrimary};
			}
			.gospa-empty {
				color: ${colors.textMuted};
				font-style: italic;
			}
		`;
	}
}

// Singleton instance
let overlayInstance: ErrorOverlay | null = null;

/**
 * Initialize the error overlay.
 */
export function initErrorOverlay(options?: ErrorOverlayOptions): ErrorOverlay {
	if (!overlayInstance) {
		overlayInstance = new ErrorOverlay(options);
	}
	return overlayInstance;
}

/**
 * Show an error in the overlay.
 */
export function showError(error: ErrorData): void {
	if (!overlayInstance) {
		initErrorOverlay();
	}
	overlayInstance?.showError(error);
}

/**
 * Hide the error overlay.
 */
export function hideErrorOverlay(): void {
	overlayInstance?.hide();
}

// Auto-initialize in development
if (typeof window !== 'undefined' && import.meta.env?.DEV) {
	initErrorOverlay();
}