const BASE = import.meta.env.VITE_API_BASE_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = init?.body instanceof FormData;
  const res = await fetch(`${BASE}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Accept": "application/json",
      ...(init?.body && !isFormData ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    let message = text;
    try {
      const json = JSON.parse(text);
      message = json.detail ?? json.title ?? json.error ?? json.message ?? text;
    } catch {}
    const err = new Error(message) as Error & { status: number };
    err.status = res.status;
    throw err;
  }

  if (res.status === 204 || res.headers.get("content-length") === "0") {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

let csrfToken: string | null = null;

export async function fetchCsrfToken(): Promise<string> {
  const data = await request<{ csrfToken: string }>("/api/v1/admin/auth/csrf");
  csrfToken = data.csrfToken;
  return csrfToken;
}

export function getCsrfToken(): string | null {
  return csrfToken;
}

export async function adminFetch<T>(path: string, init?: RequestInit): Promise<T> {
  if (init?.method && init.method !== "GET" && init.method !== "HEAD") {
    if (!csrfToken) {
      await fetchCsrfToken();
    }
    init = {
      ...init,
      headers: {
        ...init.headers,
        "X-CSRF-Token": csrfToken ?? "",
      },
    };
  }
  return request<T>(path, init);
}

export function selfServiceFetch<T>(path: string, init?: RequestInit): Promise<T> {
  return request<T>(path, init);
}

export function publicFetch<T>(path: string, init?: RequestInit): Promise<T> {
  return request<T>(path, init);
}
