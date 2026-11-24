export type ApiError = {
  code?: string;
  message?: string;
  params?: Record<string, unknown>;
};

export class ApiRequestError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

export async function api<T = unknown>(path: string, body?: unknown, method = "POST"): Promise<T> {
  const init: RequestInit = {
    method,
    credentials: "same-origin",
    headers: body !== undefined ? { "Content-Type": "application/json" } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  };
  const res = await fetch(path, init);
  const data = (await res.json().catch(() => ({}))) as { error?: ApiError } & T;
  if (!res.ok) {
    const err = (data as { error?: ApiError }).error;
    throw new ApiRequestError(err?.code || "ERR_INTERNAL", err?.message || res.statusText);
  }
  return data as T;
}

export async function apiGet<T = unknown>(path: string): Promise<T> {
  return api<T>(path, undefined, "GET");
}
