// shopping-view.js — shopping-list view: event wiring (click/tap and
// Enter/Space on ItemRow via native <button>, add form, delete button,
// mobile tabs, toast "×"/Escape) and the syncFromServer() poll cycle.
//
// Split out of the old main.js (single-page-per-route era) so the SPA
// bootstrap in main.js can mount this view without also owning identity
// resolution — GET /api/v1/me now happens exactly once, in main.js, not
// once per view (that duplication across full-page navigations is the
// bug this SPA merge fixes).

import { initStore, addItemOptimistic, moveItemOptimistic, deleteItemOptimistic, syncFromServer, setCurrentUserId, prefetchItems, rerender } from './store.js';
import { dismissToast } from './render.js';
import { wireTabs, setActivePanel } from './tabs.js';
import { onLocaleChange } from './strings.js';
import { t } from './strings.js';

const POLL_INTERVAL_MS = 10000;
const MAX_NAME_LENGTH = 200;

function validateNameClientSide(raw) {
  const trimmed = raw.trim();
  if (trimmed.length === 0) {
    return { ok: false, message: t('errorEmptyName') };
  }
  if (raw.length > MAX_NAME_LENGTH) {
    return { ok: false, message: t('errorTooLong', raw.length) };
  }
  // eslint-disable-next-line no-control-regex
  if (/[\x00-\x08\x0B\x0C\x0E-\x1F]/.test(raw)) {
    return { ok: false, message: t('errorInvalidChars') };
  }
  return { ok: true, name: trimmed };
}

// initShoppingView() mounts the shopping-list view: wires its DOM
// controls, kicks off the immediate sync + 10s poll + focus refetch
// (AC-08), and re-renders on locale change. Called once at bootstrap,
// regardless of which view is initially visible (design.md §5 Flux 3 —
// polling keeps running even for the view you are not looking at, so a
// remote change is announced/reflected promptly whenever you do switch,
// AC-06/AC-07-style).
export function initShoppingView(me) {
  initStore({
    onMove: (id) => moveItemOptimistic(id),
    onDelete: (id) => deleteItemOptimistic(id),
  });

  wireTabs();
  setActivePanel('shopping');
  wireAddForm();
  wireToastDismiss();
  onLocaleChange(() => rerender());

  setCurrentUserId(me.id);

  syncFromServer();
  setInterval(syncFromServer, POLL_INTERVAL_MS);
  window.addEventListener('focus', syncFromServer);
}

// prefetchShoppingItems() lets main.js kick off GET /api/v1/items
// concurrently with GET /api/v1/me, before initShoppingView() runs.
export function prefetchShoppingItems() {
  prefetchItems();
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
      const message = err && err.message ? err.message : t('errorGeneric');
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
