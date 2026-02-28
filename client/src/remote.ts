// GoSPA Remote Actions Client
// Type-safe HTTP client for calling server-side remote actions

/**
 * Configuration options for remote action calls
 */
export interface RemoteOptions {
	/** Custom headers to include in the request */
	headers?: Record<string, string>;
	/** Request timeout in milliseconds (default: 30000) */
	timeout?: number;
	/** Signal for request cancellation */
	signal?: AbortSignal;
}

/**
 * Result from a remote action call
 */
export interface RemoteResult<T = unknown> {
	/** The response data on success */
	data?: T;
	/** Error message if the call failed */
	error?: string;
	/** Error code for programmatic handling */
	code?: string;
	/** HTTP status code */
	status: number;
	/** Whether the request was successful */
	ok: boolean;
}

// Default remote prefix - matches server default
let remotePrefix = '/_gospa/remote';

/**
 * Configure the remote action client
 */
export function configureRemote(options: { prefix?: string }): void {
	if (options.prefix) {
		remotePrefix = options.prefix;
	}
}

/**
 * Get the current remote prefix
 */
export function getRemotePrefix(): string {
	return remotePrefix;
}

/**
 * Call a remote action on the server via HTTP POST
 *
 * @param name - The name of the registered remote action
 * @param input - The input data to pass to the action (will be JSON serialized)
 * @param options - Optional configuration for the request
 * @returns Promise resolving to the action result
 *
 * @example
 * ```typescript
 * const result = await remote('createUser', { username: 'alice' });
 * if (result.ok) {
 *   console.log('Success:', result.data);
 * } else {
 *   console.error('Error:', result.error);
 * }
 * ```
 */
export async function remote<T = unknown>(
	name: string,
	input?: unknown,
	options: RemoteOptions = {}
): Promise<RemoteResult<T>> {
	const url = `${remotePrefix}/${encodeURIComponent(name)}`;
	const timeout = options.timeout ?? 30000;

	// Only create AbortController if we need timeout or to combine with external signal
	const controller = (timeout !== undefined || options.signal)
		? new AbortController()
		: null;
	const timeoutId = controller ? setTimeout(() => controller.abort(), timeout) : null;

	// Track external signal listener for cleanup
	let abortListener: (() => void) | undefined;

	// Use provided signal or create one from our controller
	if (options.signal && controller) {
		abortListener = () => controller.abort();
		options.signal.addEventListener('abort', abortListener);
	}

	// Determine the signal to use for fetch
	const fetchSignal = controller?.signal ?? options.signal;

	try {
		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'Accept': 'application/json',
				...options.headers,
			},
			body: input !== undefined ? JSON.stringify(input) : undefined,
			signal: fetchSignal,
			credentials: 'same-origin',
		});

		if (timeoutId !== null) clearTimeout(timeoutId);

		// Parse response body
		let data: T | undefined;
		let error: string | undefined;
		let code: string | undefined;

		const contentType = response.headers.get('content-type');
		if (contentType?.includes('application/json')) {
			try {
				const json = await response.json();
				code = json.code;
				if (!response.ok) {
					error = json.error || `HTTP ${response.status}`;
				} else {
					// Handle wrapped response format: { data: ..., code: "SUCCESS" }
					data = json.data !== undefined ? json.data : json as T;
				}
			} catch (parseErr) {
				error = parseErr instanceof Error
					? `Invalid JSON: ${parseErr.message}`
					: 'Invalid JSON response';
				code = 'PARSE_ERROR';
			}
		} else if (!response.ok) {
			error = `HTTP ${response.status}: ${response.statusText}`;
			code = 'HTTP_ERROR';
		}

		return {
			data,
			error,
			code,
			status: response.status,
			ok: response.ok,
		};
	} catch (err) {
		if (timeoutId !== null) clearTimeout(timeoutId);

		if (err instanceof Error) {
			if (err.name === 'AbortError') {
				return {
					error: 'Request timeout',
					code: 'TIMEOUT',
					status: 0,
					ok: false,
				};
			}
			return {
				error: err.message,
				code: 'NETWORK_ERROR',
				status: 0,
				ok: false,
			};
		}

		return {
			error: 'Unknown error',
			code: 'UNKNOWN_ERROR',
			status: 0,
			ok: false,
		};
	} finally {
		// Clean up external signal listener to prevent memory leaks
		if (abortListener && options.signal) {
			options.signal.removeEventListener('abort', abortListener);
		}
	}
}

/**
 * Create a typed remote action caller for a specific action
 *
 * @param name - The name of the remote action
 * @returns A function that calls the remote action with the given input type
 *
 * @example
 * ```typescript
 * const createUser = remoteAction<{ username: string }, { id: string }>('createUser');
 * const result = await createUser({ username: 'alice' });
 * ```
 */
export function remoteAction<TInput = unknown, TOutput = unknown>(name: string) {
	return (input: TInput, options?: RemoteOptions): Promise<RemoteResult<TOutput>> => {
		return remote<TOutput>(name, input, options);
	};
}

// Expose to window for debugging
if (typeof window !== 'undefined') {
	(window as any).__GOSPA_REMOTE__ = {
		remote,
		remoteAction,
		configureRemote,
		getRemotePrefix,
	};
}
