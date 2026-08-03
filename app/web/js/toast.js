// toast.js — single shared toast implementation for the SPA shell.
//
// Before the SPA merge, render.js and projects-render.js each carried
// their own independent copy of this logic (their own module-level
// toastTimer), which was harmless when the shopping list and projects
// views lived on separate pages. Now that both views coexist in one page
// and share the same #toast-wrap DOM node, two independent timers racing
// against the same node let one view's toast get silently wiped by the
// other's dismiss timer. One shared timer/state fixes this at the root.

import { t } from './strings.js';

const TOAST_AUTO_DISMISS_MS = 5000;
let toastTimer = null;

export function showToast(message) {
  const wrap = document.getElementById('toast-wrap');
  if (!wrap) return;
  wrap.replaceChildren();

  const toast = document.createElement('div');
  toast.className = 'toast';
  toast.setAttribute('role', 'status');

  const icon = document.createElement('span');
  icon.className = 'icon';
  icon.setAttribute('aria-hidden', 'true');
  icon.textContent = '⚠';
  toast.appendChild(icon);

  const text = document.createElement('span');
  text.textContent = message; // user-supplied item name is interpolated by the caller
  toast.appendChild(text);

  const closeBtn = document.createElement('button');
  closeBtn.type = 'button';
  closeBtn.className = 'close-btn';
  closeBtn.setAttribute('aria-label', t('closeToast'));
  closeBtn.textContent = '×';
  closeBtn.addEventListener('click', dismissToast);
  toast.appendChild(closeBtn);

  wrap.appendChild(toast);

  if (toastTimer) clearTimeout(toastTimer);
  toastTimer = setTimeout(dismissToast, TOAST_AUTO_DISMISS_MS);
}

export function dismissToast() {
  const wrap = document.getElementById('toast-wrap');
  if (wrap) wrap.replaceChildren();
  if (toastTimer) {
    clearTimeout(toastTimer);
    toastTimer = null;
  }
}
