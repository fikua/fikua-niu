// store.js — in-memory `items` array, optimistic move with rollback
// (AC-12/AC-13), syncFromServer() diffing for remote-change detection
// (AC-08, AC-16 "per {usuari}" wording), and the one-shot confetti guard
// (AC-14/EC-13).

import * as api from './api.js';
import { render, boxLabel, showToast } from './render.js';
import { captureRects } from './flip.js';
import { announce, announceMove } from './a11y.js';
import { confetti } from './confetti.js';

let items = [];
let currentUserId = null;

// §8.6.3 one-shot confetti guard: fires only on the client-detected
// non-empty → empty transition caused by an action, never on initial
// load (EC-13), and never repeats on subsequent renders while it stays
// empty. Re-armed as soon as "A comprar" has at least one item again.
let shoppingWasEmptyCelebrated = false;

let handlers = null;

export function setCurrentUserId(id) {
  currentUserId = id;
}

function shoppingCount() {
  return items.filter((i) => i.location === 'shopping').length;
}

function doRender(opts) {
  render(items, handlers, opts);
}

function maybeCelebrateEmptying(wasNonEmptyBefore) {
  if (wasNonEmptyBefore && shoppingCount() === 0) {
    if (!shoppingWasEmptyCelebrated) {
      shoppingWasEmptyCelebrated = true;
      confetti();
    }
  } else if (shoppingCount() > 0) {
    // Shopping has items again — re-arm the trigger for the next emptying.
    shoppingWasEmptyCelebrated = false;
  }
}

// moveItemOptimistic updates the local array immediately, re-renders with
// a FLIP from the previous rects, then awaits the server call. On
// success, nothing further happens (no flicker, AC-12). On failure, it
// rolls back to the previous state, re-renders with an inverse FLIP, and
// shows a toast (AC-13).
export async function moveItemOptimistic(id) {
  const item = items.find((i) => i.id === id);
  if (!item) return;

  const previousLocation = item.location;
  const newLocation = previousLocation === 'shopping' ? 'pantry' : 'shopping';
  const wasNonEmptyBefore = shoppingCount() > 0;

  const firstRects = captureRects();
  item.location = newLocation;
  item.pending = true;
  doRender({ flipFromRects: firstRects });
  announce(`${item.name} mogut a ${boxLabel(newLocation)}.`);
  maybeCelebrateEmptying(wasNonEmptyBefore);

  try {
    const updated = await api.moveItem(id, newLocation);
    applyServerItem(updated);
    doRender({});
  } catch (err) {
    // Rollback: restore previous state, invert the FLIP, show toast.
    const rollbackRects = captureRects();
    item.location = previousLocation;
    item.pending = false;
    doRender({ flipFromRects: rollbackRects });
    showToast(`No s'ha pogut moure «${item.name}». Torna-ho a provar.`);
  }
}

export async function addItemOptimistic(rawName) {
  return api.addItem(rawName).then((created) => {
    items.unshift({ ...created, pending: false });
    doRender({});
    const newRow = document.querySelector(`#list-shopping .item-row[data-id="${created.id}"]`);
    if (newRow) newRow.classList.add('just-added');
    shoppingWasEmptyCelebrated = false; // shopping has items again
    return created;
  });
}

export async function deleteItemOptimistic(id) {
  const item = items.find((i) => i.id === id);
  const wasNonEmptyBefore = shoppingCount() > 0;
  const firstRects = captureRects();

  items = items.filter((i) => i.id !== id);
  doRender({ flipFromRects: firstRects });
  if (item) maybeCelebrateEmptying(wasNonEmptyBefore);

  try {
    await api.deleteItem(id);
  } catch (err) {
    // Deletion failures are rare (idempotent endpoint) — surface a toast
    // and resync from server to recover a consistent view.
    if (item) showToast(`No s'ha pogut eliminar «${item.name}». Torna-ho a provar.`);
    await syncFromServer();
  }
}

function applyServerItem(serverItem) {
  const idx = items.findIndex((i) => i.id === serverItem.id);
  if (idx === -1) {
    items.push({ ...serverItem, pending: false });
  } else {
    items[idx] = { ...serverItem, pending: false };
  }
}

// prefetchItems() lets main.js kick off GET /api/v1/items concurrently
// with GET /api/v1/me (NIU-4/AC-05 gates rendering on getMe(), but there
// is no reason to also serialize the items fetch behind it — on a slow
// connection that costs a full extra round trip, see perf-3g.spec.js/
// NFR-06). The first syncFromServer() call consumes this in-flight
// promise instead of issuing a second, redundant request.
let prefetchedItemsPromise = null;

export function prefetchItems() {
  prefetchedItemsPromise = api.getItems();
  // Swallow rejections here so an unused/failed prefetch never surfaces as
  // an unhandled promise rejection — syncFromServer() below still awaits
  // (and separately catches) the same promise.
  prefetchedItemsPromise.catch(() => {});
}

// syncFromServer() polls GET /api/v1/items and diffs by id + location +
// moved_at against the previously known state, to detect remote changes
// (AC-08) and announce them with the "per {usuari}" wording (AC-16,
// design.md §5 Flux 3).
export async function syncFromServer() {
  let serverItems;
  try {
    const pending = prefetchedItemsPromise;
    prefetchedItemsPromise = null;
    serverItems = await (pending || api.getItems());
  } catch {
    return; // transient network failure — try again on the next tick
  }

  const previousById = new Map(items.map((i) => [i.id, i]));
  const firstRects = captureRects();

  const nextItems = serverItems.map((s) => ({ ...s, pending: false }));

  // Detect remote moves: items known before, whose location or moved_at
  // changed, and whose mover is not the current user (avoids
  // re-announcing our own optimistic move once confirmed).
  for (const s of nextItems) {
    const prev = previousById.get(s.id);
    if (!prev) continue; // brand new item, handled by fade-in below
    const locationChanged = prev.location !== s.location;
    const movedAtChanged = (prev.moved_at || null) !== (s.moved_at || null);
    if ((locationChanged || movedAtChanged) && s.moved_by && s.moved_by.id !== currentUserId) {
      announceMove(s.name, s.location, s.moved_by.display_name);
    }
  }

  const wasNonEmptyBefore = items.length > 0 && shoppingCount() > 0;
  items = nextItems;
  doRender({ flipFromRects: firstRects });
  maybeCelebrateEmptying(wasNonEmptyBefore);
}

export function initStore(h) {
  handlers = h;
}

export function getItems() {
  return items;
}
