// ideas-api.js — fetch wrappers against /api/v1/ideas/*. Same pattern as
// api.js's items wrappers and projects-api.js: no innerHTML, no DOM here,
// returns parsed JSON on success or throws ApiError on failure
// (design.md §6.1).

import { ApiError } from './api.js';
import { t } from './strings.js';

function getCsrfToken() {
  const match = document.cookie.match(/(?:^|; )niu_csrf=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

// handleUnauthenticated centralizes the 401 -> redirect-to-login
// behaviour, same guard as api.js/projects-api.js's private copies
// (idempotent: two concurrent 401s on the same page load must not race
// two window.location.href assignments against each other).
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

export function getIdeas() {
  return request('/api/v1/ideas').then((body) => body.ideas);
}

export function addIdea(url) {
  return request('/api/v1/ideas', {
    method: 'POST',
    headers: mutationHeaders(),
    body: JSON.stringify({ url }),
  });
}

export function deleteIdea(id) {
  return request(`/api/v1/ideas/${id}`, {
    method: 'DELETE',
    headers: mutationHeaders(),
  });
}

export function getMe() {
  return request('/api/v1/me');
}
