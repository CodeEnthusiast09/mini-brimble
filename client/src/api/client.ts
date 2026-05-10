export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1";

export async function apiFetch<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const res = await fetch(`${API_BASE_URL}${path}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });

  const body = await res
    .json()
    .catch(() => ({ message: `HTTP ${res.status}` }));

  if (!res.ok) {
    throw new Error(
      (body as { message?: string }).message ?? `HTTP ${res.status}`,
    );
  }

  return body as T;
}
