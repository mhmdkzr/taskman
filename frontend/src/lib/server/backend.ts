import { env } from '$env/dynamic/private';

function backendBaseURL(): string {
	let url = env.BACKEND_URL || 'http://127.0.0.1:8080';
	if (!url.startsWith('http://') && !url.startsWith('https://')) {
		url = 'http://' + url;
	}
	return url.replace(/\/+$/, '');
}

export async function api(
	request: Request,
	path: string,
	init?: RequestInit,
): Promise<Response> {
	const base = backendBaseURL();
	const headers = new Headers(init?.headers);
	const userId = request.headers.get('X-User-ID');
	if (userId) {
		headers.set('X-User-ID', userId);
	}
	const url = `${base}${path.startsWith('/') ? '' : '/'}${path}`;
	return fetch(url, { ...init, headers });
}
