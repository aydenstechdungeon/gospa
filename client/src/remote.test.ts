import { describe, it, expect, beforeEach, afterEach, mock } from 'bun:test';
import { remote, remoteAction, configureRemote, getRemotePrefix, type RemoteResult } from './remote';

// Mock global fetch
declare var global: {
	fetch: typeof fetch;
};

describe('remote', () => {
	let fetchMock: ReturnType<typeof mock>;

	beforeEach(() => {
		// Reset to default prefix
		configureRemote({ prefix: '/_gospa/remote' });

		// Mock fetch
		fetchMock = mock(() => Promise.resolve({
			ok: true,
			status: 200,
			headers: new Map([['content-type', 'application/json']]),
			json: () => Promise.resolve({ data: 'test result', code: 'SUCCESS' }),
		} as unknown as Response));
		global.fetch = fetchMock as unknown as typeof fetch;
	});

	afterEach(() => {
		fetchMock.mockClear();
	});

	it('should call remote action with correct URL', async () => {
		await remote('testAction', { key: 'value' });

		expect(fetchMock).toHaveBeenCalled();
		const call = fetchMock.mock.calls[0];
		expect(call[0]).toBe('/_gospa/remote/testAction');
	});

	it('should encode action name in URL', async () => {
		await remote('action with spaces', null);

		const call = fetchMock.mock.calls[0];
		expect(call[0]).toContain('action%20with%20spaces');
	});

	it('should send POST request with JSON body', async () => {
		const input = { name: 'test', value: 42 };
		await remote('testAction', input);

		const call = fetchMock.mock.calls[0];
		const options = call[1] as RequestInit;

		expect(options.method).toBe('POST');
		expect(options.body).toBe(JSON.stringify(input));
	});

	it('should include correct headers', async () => {
		await remote('testAction', {});

		const call = fetchMock.mock.calls[0];
		const options = call[1] as RequestInit;
		const headers = options.headers as Record<string, string>;

		expect(headers['Content-Type']).toBe('application/json');
		expect(headers['Accept']).toBe('application/json');
	});

	it('should handle successful response', async () => {
		const result = await remote('testAction', null);

		expect(result.ok).toBe(true);
		expect(result.status).toBe(200);
		expect(result.data).toBe('test result');
	});

	it('should handle error response', async () => {
		fetchMock.mockImplementation(() => Promise.resolve({
			ok: false,
			status: 400,
			headers: new Map([['content-type', 'application/json']]),
			json: () => Promise.resolve({ error: 'Bad Request', code: 'BAD_REQUEST' }),
		} as unknown as Response));

		const result = await remote('testAction', null);

		expect(result.ok).toBe(false);
		expect(result.status).toBe(400);
		expect(result.error).toBe('Bad Request');
		expect(result.code).toBe('BAD_REQUEST');
	});

	it('should handle network error', async () => {
		fetchMock.mockImplementation(() => Promise.reject(new Error('Network failed')));

		const result = await remote('testAction', null);

		expect(result.ok).toBe(false);
		expect(result.status).toBe(0);
		expect(result.code).toBe('NETWORK_ERROR');
		expect(result.error).toContain('Network failed');
	});

	it('should handle timeout', async () => {
		// Mock fetch that never resolves (simulating timeout)
		fetchMock.mockImplementation(() => new Promise(() => {}));

		const result = await remote('testAction', null, { timeout: 1 });

		expect(result.ok).toBe(false);
		expect(result.status).toBe(0);
		expect(result.code).toBe('TIMEOUT');
		expect(result.error).toBe('Request timeout');
	});

	it('should handle non-JSON response', async () => {
		fetchMock.mockImplementation(() => Promise.resolve({
			ok: false,
			status: 500,
			headers: new Map([['content-type', 'text/plain']]),
			statusText: 'Internal Server Error',
		} as unknown as Response));

		const result = await remote('testAction', null);

		expect(result.ok).toBe(false);
		expect(result.status).toBe(500);
		expect(result.error).toContain('HTTP 500');
	});

	it('should handle abort signal', async () => {
		const controller = new AbortController();
		controller.abort();

		const result = await remote('testAction', null, { signal: controller.signal });

		expect(result.ok).toBe(false);
		expect(result.status).toBe(0);
		expect(result.code).toBe('NETWORK_ERROR');
	});

	it('should configure custom prefix', async () => {
		configureRemote({ prefix: '/api/remote' });
		await remote('testAction', null);

		const call = fetchMock.mock.calls[0];
		expect(call[0]).toBe('/api/remote/testAction');
	});

	it('should get current prefix', () => {
		configureRemote({ prefix: '/custom/prefix' });
		expect(getRemotePrefix()).toBe('/custom/prefix');
	});

	it('should include custom headers', async () => {
		await remote('testAction', null, {
			headers: { 'X-Custom-Header': 'custom-value' }
		});

		const call = fetchMock.mock.calls[0];
		const options = call[1] as RequestInit;
		const headers = options.headers as Record<string, string>;

		expect(headers['X-Custom-Header']).toBe('custom-value');
	});

	it('should handle wrapped response with data field', async () => {
		fetchMock.mockImplementation(() => Promise.resolve({
			ok: true,
			status: 200,
			headers: new Map([['content-type', 'application/json']]),
			json: () => Promise.resolve({ data: { nested: 'value' }, code: 'SUCCESS' }),
		} as unknown as Response));

		const result = await remote('testAction', null);

		expect(result.ok).toBe(true);
		expect(result.data).toEqual({ nested: 'value' });
	});

	it('should handle plain JSON response without wrapper', async () => {
		fetchMock.mockImplementation(() => Promise.resolve({
			ok: true,
			status: 200,
			headers: new Map([['content-type', 'application/json']]),
			json: () => Promise.resolve({ direct: 'value' }),
		} as unknown as Response));

		const result = await remote('testAction', null);

		expect(result.ok).toBe(true);
		expect(result.data).toEqual({ direct: 'value' });
	});

	it('should handle empty response body', async () => {
		fetchMock.mockImplementation(() => Promise.resolve({
			ok: true,
			status: 204,
			headers: new Map(),
		} as unknown as Response));

		const result = await remote('testAction', null);

		expect(result.ok).toBe(true);
		expect(result.status).toBe(204);
	});
});

describe('remoteAction', () => {
	let fetchMock: ReturnType<typeof mock>;

	beforeEach(() => {
		configureRemote({ prefix: '/_gospa/remote' });

		fetchMock = mock(() => Promise.resolve({
			ok: true,
			status: 200,
			headers: new Map([['content-type', 'application/json']]),
			json: () => Promise.resolve({ data: { id: '123' }, code: 'SUCCESS' }),
		} as unknown as Response));
		global.fetch = fetchMock as unknown as typeof fetch;
	});

	afterEach(() => {
		fetchMock.mockClear();
	});

	it('should create typed remote action caller', async () => {
		interface CreateUserInput {
			username: string;
			email: string;
		}

		interface CreateUserOutput {
			id: string;
		}

		const createUser = remoteAction<CreateUserInput, CreateUserOutput>('createUser');

		const result = await createUser({ username: 'alice', email: 'alice@example.com' });

		expect(result.ok).toBe(true);
		expect(result.data?.id).toBe('123');
	});

	it('should pass options to remote call', async () => {
		const action = remoteAction<string, string>('test');

		await action('input', { timeout: 5000, headers: { 'X-Test': 'value' } });

		const call = fetchMock.mock.calls[0];
		const options = call[1] as RequestInit;
		const headers = options.headers as Record<string, string>;

		expect(headers['X-Test']).toBe('value');
	});
});
