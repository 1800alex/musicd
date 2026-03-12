import { Capacitor } from "@capacitor/core";
import { isElectron } from "~/utils/platform";

const httpService = {
	get,
	post,
	delete: del
};

export interface IRequestOpts {
	disableRedirect?: boolean;
	headers?: HeadersInit;
	baseURL?: string;
	params?: Record<string, any>;
}

export interface IMessage {
	message?: string;
	code?: number;
}

export interface IMessagePayload<T> extends IMessage {
	data: T;
}

export interface IResponse<T> {
	data: T;
	status: number;
	headers: Headers;
	error?: Error;
}

/**
 * Resolve the base URL for API requests.
 * In Capacitor native builds and Electron builds the app is served from local files, so API
 * requests need an absolute URL pointing at the backend server.
 * On the web the relative "/" base works because the browser resolves it
 * against the current origin.
 */
export function resolveBaseURL(override?: string): string {
	if (override) return override;

	if (Capacitor.isNativePlatform() || isElectron()) {
		// In native/electron builds, read the backend URL from localStorage (set by
		// a settings page) or fall back to the compile-time env variable.
		const stored = localStorage.getItem("backendURL");
		if (stored) return stored;

		// Fall back to the Nuxt runtime config value injected at build time.
		// __NUXT_PUBLIC_BACKEND_URL__ is replaced by Vite's define plugin.
		try {
			const rc = useRuntimeConfig();
			if (rc?.public?.backendURL) return rc.public.backendURL as string;
		} catch {
			// useRuntimeConfig may not be available outside setup context
		}

		return "/";
	}

	return "/";
}

export default httpService;

async function get<T>(apiEndpoint: string, opts: IRequestOpts = {}) {
	return new Promise<IResponse<T>>((resolve, reject) => {
		let done = false;

		if (!opts.headers) {
			opts.headers = {};
		}

		const res: IResponse<T> = {} as IResponse<T>;

		$fetch<T>(apiEndpoint, {
			method: "GET",
			headers: opts.headers,
			baseURL: resolveBaseURL(opts.baseURL),
			params: opts.params,
			async onRequestError({ request, error }) {
				// Log error
				console.error("[fetch request error]", request, error);
			},
			async onResponse({ request, response }) {
				res.status = response.status;
				res.headers = response.headers;
			},
			async onResponseError({ request, response, error }) {
				if (done) {
					return;
				}
				done = true;

				// Log error
				res.status = response.status;

				res.error = error;
				// console.error("[fetch response error]", request, response.status, response.body, error);
				reject(res);
			}
		})
			.then((data) => {
				if (done) {
					return;
				}
				done = true;

				res.data = data;
				resolve(res);
			})
			.catch((error) => {
				if (done) {
					return;
				}
				done = true;

				res.error = error;
				reject(res);
			});
	});
}

async function post<T>(apiEndpoint: string, payload: any, opts: IRequestOpts = {}) {
	return new Promise<IResponse<T>>((resolve, reject) => {
		let done = false;

		if (!opts.headers) {
			opts.headers = {};
		}

		const res: IResponse<T> = {} as IResponse<T>;

		$fetch<T>(apiEndpoint, {
			method: "POST",
			headers: opts.headers,
			body: payload,
			baseURL: resolveBaseURL(opts.baseURL),
			params: opts.params,
			async onRequestError({ request, error }) {
				// Log error
				console.error("[post request error]", request, error);
			},
			async onResponse({ request, response }) {
				res.status = response.status;
				res.headers = response.headers;
			},
			async onResponseError({ request, response, error }) {
				if (done) {
					return;
				}
				done = true;

				// Log error
				res.data = response._data;
				res.status = response.status;

				// console.log("[post response error]", request, response.status, response.body, error);
				res.error = error;
				reject(res);
			}
		})
			.then((data) => {
				if (done) {
					return;
				}
				done = true;

				res.data = data;
				resolve(res);
			})
			.catch((error) => {
				if (done) {
					return;
				}
				done = true;

				res.error = error;
				reject(res);
			});
	});
}

async function del<T>(apiEndpoint: string, opts: IRequestOpts = {}) {
	return new Promise<IResponse<T>>((resolve, reject) => {
		let done = false;

		if (!opts.headers) {
			opts.headers = {};
		}

		const res: IResponse<T> = {} as IResponse<T>;

		$fetch<T>(apiEndpoint, {
			method: "DELETE",
			headers: opts.headers,
			baseURL: resolveBaseURL(opts.baseURL),
			params: opts.params,
			async onRequestError({ request, error }) {
				// Log error
				console.error("[delete request error]", request, error);
			},
			async onResponse({ request, response }) {
				res.status = response.status;
				res.headers = response.headers;
			},
			async onResponseError({ request, response, error }) {
				if (done) {
					return;
				}
				done = true;

				// Log error
				res.data = response._data;
				res.status = response.status;

				res.error = error;
				reject(res);
			}
		})
			.then((data) => {
				if (done) {
					return;
				}
				done = true;

				res.data = data;
				resolve(res);
			})
			.catch((error) => {
				if (done) {
					return;
				}
				done = true;

				res.error = error;
				reject(res);
			});
	});
}
