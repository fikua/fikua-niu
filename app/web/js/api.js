// api.js — fetch wrappers against /api/v1/*. No innerHTML, no DOM here.
// Each function returns the parsed JSON body on success, or throws an
// ApiError with { status, code, message } on failure so callers can
// branch on the uniform error envelope (design.md §6.1).

export class ApiError extends Error {
  constructor(status, code, message) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

async function request(path, options = {}) {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });

  if (res.status === 204) {
    return null;
  }

  let body = null;
  try {
    body = await res.json();
  } catch {
    body = null;
  }

  if (!res.ok) {
    const code = body?.error?.code || 'internal_error';
    const message = body?.error?.message || "S'ha produït un error inesperat.";
    throw new ApiError(res.status, code, message);
  }

  return body;
}

export function getItems() {
  return request('/api/v1/items').then((body) => body.items);
}

export function addItem(name) {
  return request('/api/v1/items', {
    method: 'POST',
    body: JSON.stringify({ name }),
  });
}

export function moveItem(id, location) {
  return request(`/api/v1/items/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ location }),
  });
}

export function deleteItem(id) {
  return request(`/api/v1/items/${id}`, { method: 'DELETE' });
}

export function getMe() {
  return request('/api/v1/me');
}
