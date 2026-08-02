// main.js — entry point: event wiring (click/tap and Enter/Space on
// ItemRow via native <button>, add form, delete button, mobile tabs,
// toast "×"/Escape), immediate syncFromServer() + setInterval(10000) +
// window focus listener, and GET /api/v1/me on load.

import * as api from './api.js';
import { initAnnounce } from './a11y.js';
import { initStore, addItemOptimistic, moveItemOptimistic, deleteItemOptimistic, syncFromServer, setCurrentUserId, prefetchItems } from './store.js';
import { dismissToast } from './render.js';
import { wireTabs, setActivePanel } from './tabs.js';
import { logout } from './auth.js';

const POLL_INTERVAL_MS = 10000;

const MAX_NAME_LENGTH = 200;

function boxLabelFor(location) {
  return location === 'shopping' ? 'A comprar' : 'Rebost';
}

function validateNameClientSide(raw) {
  const trimmed = raw.trim();
  if (trimmed.length === 0) {
    return { ok: false, message: "Escriu un nom abans d'afegir." };
  }
  if (raw.length > MAX_NAME_LENGTH) {
    return { ok: false, message: `Massa llarg — màxim 200 caràcters (portes ${raw.length}/200).` };
  }
  // eslint-disable-next-line no-control-regex
  if (/[\x00-\x08\x0B\x0C\x0E-\x1F]/.test(raw)) {
    return { ok: false, message: 'Aquest nom conté caràcters no vàlids.' };
  }
  return { ok: true, name: trimmed };
}

async function main() {
  // AC-05/design.md §7: resolve identity BEFORE mounting any UI. On 401,
  // api.getMe() already triggers the redirect to /login.html?next=... (via
  // api.js's centralized handleUnauthenticated) — returning here avoids
  // rendering so much as an empty list first (no flicker).
  //
  // GET /api/v1/items is kicked off in parallel (not awaited here) so the
  // two requests share one round trip's worth of latency instead of being
  // serialized — NFR-06 budgets initial load at <1s even on a throttled
  // connection, and getMe()+getItems() run one-after-another would burn a
  // second RTT for no reason (both succeed or fail together: no session
  // means neither endpoint returns data).
  prefetchItems();

  let me;
  try {
    me = await api.getMe();
  } catch {
    return;
  }

  const liveRegion = document.getElementById('live-region');
  initAnnounce(liveRegion);

  initStore({
    onMove: (id) => moveItemOptimistic(id),
    onDelete: (id) => deleteItemOptimistic(id),
  });

  wireTabs();
  setActivePanel('shopping');
  wireAddForm();
  wireToastDismiss();
  wireLogoutButton();

  setCurrentUserId(me.id);
  const nameEl = document.getElementById('user-name');
  const avatarEl = document.getElementById('user-avatar');
  if (nameEl) nameEl.textContent = me.display_name;
  if (avatarEl) avatarEl.textContent = me.avatar_emoji;

  // Flux 3 (design.md §5): immediate sync, then poll every ~10s, plus a
  // refetch on window focus (AC-08).
  syncFromServer();
  setInterval(syncFromServer, POLL_INTERVAL_MS);
  window.addEventListener('focus', syncFromServer);
}

function wireAddForm() {
  const addInput = document.getElementById('add-input');
  const addBtn = document.getElementById('add-btn');
  const addGroup = document.getElementById('add-group');
  const addError = document.getElementById('add-error');
  const addCounter = document.getElementById('add-counter');

  function showError(message) {
    addError.textContent = message;
    addError.hidden = false;
    addGroup.classList.add('has-error');
  }

  function clearError() {
    addError.hidden = true;
    addGroup.classList.remove('has-error');
  }

  function updateCounter() {
    const len = addInput.value.length;
    addCounter.textContent = `${len}/200`;
    // §8.7: counter is aria-live only when <=20 chars remain, to avoid
    // noise while typing a short name.
    if (200 - len <= 20) {
      addCounter.setAttribute('aria-live', 'polite');
    } else {
      addCounter.removeAttribute('aria-live');
    }
  }

  async function submitNewItem() {
    const raw = addInput.value;
    const clientCheck = validateNameClientSide(raw);
    if (!clientCheck.ok) {
      showError(clientCheck.message);
      addInput.focus(); // focus stays on input, text is NOT cleared (§8.4.3)
      return;
    }

    addInput.disabled = true;
    addBtn.disabled = true;
    try {
      await addItemOptimistic(raw);
      clearError();
      addInput.value = '';
      updateCounter();
    } catch (err) {
      // Server-side validation/duplicate errors (EC-06, EC-01..EC-05).
      const message = err && err.message ? err.message : "S'ha produït un error inesperat.";
      showError(message);
    } finally {
      addInput.disabled = false;
      addBtn.disabled = false;
      addInput.focus();
    }
  }

  addBtn.addEventListener('click', submitNewItem);
  addInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      submitNewItem();
    }
  });
  addInput.addEventListener('input', updateCounter);

  updateCounter();
}

function wireToastDismiss() {
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      dismissToast();
    }
  });
}

function wireLogoutButton() {
  const btn = document.getElementById('logout-btn');
  if (!btn) return;
  btn.addEventListener('click', () => {
    logout();
  });
}

main();
