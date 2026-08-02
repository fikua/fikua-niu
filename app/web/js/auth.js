// auth.js — login/logout for the standalone login screen (design.md §7).
// No innerHTML with user data anywhere here — only textContent.

import { ApiError } from './api.js';

function getCsrfToken() {
  const match = document.cookie.match(/(?:^|; )niu_csrf=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

async function login(username, password) {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password }),
  });

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

export async function logout() {
  await fetch('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'X-CSRF-Token': getCsrfToken() },
  });
  window.location.href = '/login.html';
}

function nextURL() {
  const params = new URLSearchParams(window.location.search);
  return params.get('next') || '/';
}

function wireLoginForm() {
  const form = document.getElementById('login-form');
  if (!form) return; // Not on the login screen.

  const usernameInput = document.getElementById('username-input');
  const passwordInput = document.getElementById('password-input');
  const usernameGroup = document.getElementById('username-group');
  const passwordGroup = document.getElementById('password-group');
  const errorEl = document.getElementById('login-error');
  const submitBtn = document.getElementById('login-submit');

  function showError(message) {
    errorEl.textContent = message;
    errorEl.hidden = false;
    usernameGroup.classList.add('has-error');
    passwordGroup.classList.add('has-error');
  }

  function clearError() {
    errorEl.hidden = true;
    usernameGroup.classList.remove('has-error');
    passwordGroup.classList.remove('has-error');
  }

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    clearError();

    const username = usernameInput.value;
    const password = passwordInput.value;

    usernameInput.disabled = true;
    passwordInput.disabled = true;
    submitBtn.disabled = true;

    try {
      await login(username, password);
      window.location.href = nextURL();
    } catch (err) {
      const message = err instanceof ApiError
        ? err.message
        : "S'ha produït un error inesperat.";
      showError(message);
    } finally {
      usernameInput.disabled = false;
      passwordInput.disabled = false;
      submitBtn.disabled = false;
    }
  });
}

wireLoginForm();
