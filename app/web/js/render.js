// render.js — render(), renderRow(), renderEmptyState(), renderAvatars(),
// and the Toast component (§8.4.4). Direct port of the logic already
// proved in design-system/screen-desktop.html / screen-mobile.html.
//
// S3/NFR-01: zero innerHTML with user data anywhere in this file — every
// node carrying user-controlled text (item.name, display names) is built
// with document.createElement + .textContent only. List containers are
// cleared with replaceChildren() (no HTML parsing involved), never
// innerHTML = ''.

import { playFlip } from './flip.js';
import { boxLabel, wireRowFocusTabindex } from './a11y.js';
import { t } from './strings.js';

const listShopping = () => document.getElementById('list-shopping');
const listPantry = () => document.getElementById('list-pantry');
const countShopping = () => document.getElementById('count-shopping');
const countPantry = () => document.getElementById('count-pantry');
const tabCountShopping = () => document.getElementById('tab-count-shopping');
const tabCountPantry = () => document.getElementById('tab-count-pantry');

export function renderEmptyState(list, location) {
  const wrap = document.createElement('li');
  wrap.style.listStyle = 'none';
  const el = document.createElement('div');
  el.className = 'empty-state';
  el.id = location === 'shopping' ? 'empty-shopping' : 'empty-pantry';
  const icon = document.createElement('span');
  icon.className = 'icon';
  icon.setAttribute('aria-hidden', 'true');
  icon.textContent = '🌿';
  const msg = document.createElement('p');
  msg.className = 'msg';
  msg.textContent = location === 'shopping'
    ? t('emptyShopping')
    : t('emptyPantry');
  el.appendChild(icon);
  el.appendChild(msg);
  wrap.appendChild(el);
  list.appendChild(wrap);
}

export function renderAvatars(item) {
  const wrap = document.createElement('span');
  wrap.className = 'avatars';
  wrap.setAttribute('aria-hidden', 'true');

  const addedAvatar = document.createElement('span');
  addedAvatar.className = 'avatar';
  if (item.added_by) {
    addedAvatar.title = t('addedBy', item.added_by.display_name);
    addedAvatar.textContent = item.added_by.avatar_emoji;
  }
  wrap.appendChild(addedAvatar);

  const movedByDifferent = item.moved_by && (!item.added_by || item.moved_by.id !== item.added_by.id);
  if (movedByDifferent) {
    const link = document.createElement('span');
    link.className = 'avatar-link';
    link.textContent = '↩';
    wrap.appendChild(link);
    const movedAvatar = document.createElement('span');
    movedAvatar.className = 'avatar';
    movedAvatar.title = t('movedBy', item.moved_by.display_name);
    movedAvatar.textContent = item.moved_by.avatar_emoji;
    wrap.appendChild(movedAvatar);
  }
  return wrap;
}

function ariaLabelFor(item) {
  const target = item.location === 'shopping' ? t('boxPantry') : t('boxShopping');
  let label = t('moveTo', item.name, target);
  const movedByDifferent = item.moved_by && (!item.added_by || item.moved_by.id !== item.added_by.id);
  if (movedByDifferent) {
    label += t('movedAddedBy', item.added_by ? item.added_by.display_name : '', item.moved_by.display_name);
  }
  return label;
}

export function renderRow(item, handlers) {
  const li = document.createElement('li');
  li.style.listStyle = 'none';

  // NOTE (deviation from design-system/screen-*.html, flagged for
  // /audit): the reference preview builds the row as a native <button>
  // containing a second nested <button class="delete-btn">. Two
  // independent problems were found empirically against the literal
  // port (T-29/T-33 Playwright findings) and both trace back to the same
  // root cause — nested interactive controls:
  //   1. Keyboard: Tab from the row never reaches the inner button,
  //      because the CSS that reveals it (`visibility: hidden` by
  //      default, shown via `:focus-visible`) excludes it from the
  //      accessibility tree's tab order at the exact moment Tab needs it
  //      (a real Chromium behaviour, not a bug in this port).
  //   2. axe-core (WCAG 4.1.2 "Interactive controls must not be nested"):
  //      a <button>-in-a-<button> (or div[role=button] containing a
  //      <button>) is invalid regardless of tag choice — assistive tech
  //      cannot reliably disambiguate which control is "the" interactive
  //      one.
  // Fix: `row` (the visual, CSS-styled container — .item-row) is now a
  // plain, NON-interactive <div>. The "move" action is a separate
  // full-cover <button> absolutely positioned to fill the row (so the
  // whole row is still clickable/tappable, matching the approved visual
  // spec pixel-for-pixel), and the delete button is a sibling, not a
  // child of it. No class name, id, or visual CSS rule changes — only
  // which element receives `role`/`tabindex`/the click handler.
  const row = document.createElement('div');
  row.className = 'item-row';
  row.dataset.id = String(item.id);

  const moveBtn = document.createElement('button');
  moveBtn.type = 'button';
  moveBtn.className = 'item-row-move-target';
  moveBtn.setAttribute('aria-label', ariaLabelFor(item));
  moveBtn.addEventListener('click', () => handlers.onMove(item.id));
  row.appendChild(moveBtn);

  const indicator = document.createElement('span');
  indicator.className = `indicator ${item.location === 'shopping' ? 'shopping' : 'pantry'}`;
  indicator.setAttribute('aria-hidden', 'true');
  row.appendChild(indicator);

  const name = document.createElement('span');
  name.className = 'item-name';
  name.textContent = item.name; // S3/NFR-01: textContent only, never innerHTML
  row.appendChild(name);

  row.appendChild(renderAvatars(item));

  const del = document.createElement('button');
  del.type = 'button';
  del.className = 'delete-btn';
  del.setAttribute('aria-label', t('deleteItem', item.name));
  del.textContent = '🗑';
  del.addEventListener('click', (e) => {
    e.stopPropagation();
    handlers.onDelete(item.id);
  });
  row.appendChild(del);

  wireRowFocusTabindex(row, moveBtn, del);

  row.addEventListener('mousedown', () => row.classList.add('is-pressing'));
  row.addEventListener('mouseup', () => row.classList.remove('is-pressing'));
  row.addEventListener('mouseleave', () => row.classList.remove('is-pressing'));

  if (item.pending) {
    row.classList.add('is-pending');
  }

  li.appendChild(row);
  return li;
}

// render() is pure with respect to the `items` array passed in: it
// rebuilds both lists from scratch every call. flipFromRects, if given,
// triggers a FLIP/cross-fade for rows whose position changed.
//
// Focus preservation: render() is called twice per optimistic move (once
// immediately, once again when the server confirms) — a full rebuild
// would otherwise silently drop keyboard focus from whichever row/button
// the user was on, right in the middle of a Tab sequence (found via the
// T-29 Playwright keyboard-navigation spec). We snapshot which item id
// currently holds focus (row itself, or its delete button) before
// clearing the lists, and restore focus to the equivalent element in the
// freshly-built row after rebuilding.
export function render(items, handlers, { flipFromRects } = {}) {
  const shoppingItems = items.filter((i) => i.location === 'shopping');
  const pantryItems = items.filter((i) => i.location === 'pantry');

  countShopping().textContent = `(${shoppingItems.length})`;
  countPantry().textContent = `(${pantryItems.length})`;
  if (tabCountShopping()) tabCountShopping().textContent = `(${shoppingItems.length})`;
  if (tabCountPantry()) tabCountPantry().textContent = `(${pantryItems.length})`;

  const focused = captureFocusedItem();

  const shoppingList = listShopping();
  const pantryList = listPantry();
  shoppingList.replaceChildren();
  pantryList.replaceChildren();

  if (shoppingItems.length === 0) {
    renderEmptyState(shoppingList, 'shopping');
  } else {
    shoppingItems.forEach((i) => shoppingList.appendChild(renderRow(i, handlers)));
  }

  if (pantryItems.length === 0) {
    renderEmptyState(pantryList, 'pantry');
  } else {
    pantryItems.forEach((i) => pantryList.appendChild(renderRow(i, handlers)));
  }

  if (flipFromRects) {
    playFlip(flipFromRects);
  }

  restoreFocusedItem(focused);
}

export { boxLabel };

// captureFocusedItem records which item (by id) currently holds focus,
// and which of the row's two independent controls (the full-cover
// "move" button, or the "delete" button — see render.js's NOTE on
// non-nested interactive controls) so render() can restore an
// equivalent focus target after rebuilding the DOM.
function captureFocusedItem() {
  const active = document.activeElement;
  if (!active) return null;

  const row = active.closest('.item-row');
  if (!row) return null;

  return {
    id: row.dataset.id,
    target: active.classList.contains('delete-btn') ? 'delete' : 'move',
  };
}

function restoreFocusedItem(focused) {
  if (!focused) return;
  const row = document.querySelector(`.item-row[data-id="${focused.id}"]`);
  if (!row) return;
  const selector = focused.target === 'delete' ? '.delete-btn' : '.item-row-move-target';
  const el = row.querySelector(selector);
  if (el) el.focus();
}

// ================= Toast (§8.4.4) =================
//
// Non-blocking, auto-dismissed after 5s, also dismissable with the "×"
// button or the Escape key (wired in main.js). role="status" (not
// "alert") — it accompanies a reversible visual change, it does not
// interrupt.

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
