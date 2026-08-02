// api.js — fetch wrappers against /api/v1/*. No innerHTML, no DOM here.
// Each function returns the parsed JSON body on success, or throws an
// ApiError with { status, code, message } on failure so callers can
// branch on the uniform error envelope (design.md §6.1).

import { t } from './strings.js';

export class ApiError extends Error {
  constructor(status, code, message) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

// getCsrfToken reads the non-HttpOnly niu_csrf cookie (ADR-05, design.md
// §7) — no external library, plain document.cookie parsing.
export function getCsrfToken() {
  const match = document.cookie.match(/(?:^|; )niu_csrf=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

// handleUnauthenticated centralizes the 401 -> redirect-to-login behaviour
// so every wrapper (mutation or read) gets it without duplicating the
// logic at each call site (design.md §7). Idempotent: main.js's
// getMe()/prefetched getItems() calls can both resolve to a 401 for the
// same page load — without this guard, two concurrent
// window.location.href assignments can abort each other's navigation.
let redirectingToLogin = false;
function handleUnauthenticated() {
  if (redirectingToLogin) return;
  redirectingToLogin = true;
  const next = encodeURIComponent(window.location.pathname + window.location.search);
  window.location.href = `/login.html?next=${next}`;
}

async function request(path, options = {}) {
  const res = await fetch(path, {
    ...options,
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });

  if (res.status === 401) {
    handleUnauthenticated();
    // The redirect above is asynchronous; throw so callers' .then chains
    // do not proceed against an unauthenticated response in the meantime.
    throw new ApiError(401, 'unauthenticated', t('errorNeedsLogin'));
  }

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
    const message = body?.error?.message || t('errorGeneric');
    throw new ApiError(res.status, code, message);
  }

  return body;
}

function mutationHeaders() {
  return { 'X-CSRF-Token': getCsrfToken() };
}

export function getItems() {
  return request('/api/v1/items').then((body) => body.items);
}

export function addItem(name) {
  return request('/api/v1/items', {
    method: 'POST',
    headers: mutationHeaders(),
    body: JSON.stringify({ name }),
  });
}

export function moveItem(id, location) {
  return request(`/api/v1/items/${id}`, {
    method: 'PATCH',
    headers: mutationHeaders(),
    body: JSON.stringify({ location }),
  });
}

export function deleteItem(id) {
  return request(`/api/v1/items/${id}`, {
    method: 'DELETE',
    headers: mutationHeaders(),
  });
}

export function getMe() {
  return request('/api/v1/me');
}
