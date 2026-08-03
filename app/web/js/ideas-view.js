// ideas-view.js — ideas view: event wiring for the add-link form and
// delete buttons, and the syncIdeasFromServer() poll cycle (same 10s
// cycle already used by shopping-view.js/projects-view.js — no second
// sync mechanism, AC-06).
//
// Keyboard (AC-09): the add form is a plain text <input> + a "Desar"
// <button>, both natively focusable/activatable via Tab/Enter/Space —
// no custom keyboard handling is needed beyond submitting on Enter from
// the input (same pattern as projects-view.js's wireAddForm). Each
// IdeaCard's delete button is a native <button>, always in the tab
// order (no visibility:hidden/tabindex gymnastics, same fix already
// applied to render.js's ItemRow).

import {
  initIdeasStore,
  addIdeaOptimistic,
  deleteIdeaOptimistic,
  syncIdeasFromServer,
  setCurrentUserId,
  prefetchIdeas,
} from './ideas-store.js';
import { dismissToast } from './ideas-render.js';

const POLL_INTERVAL_MS = 10000;

// initIdeasView() mounts the ideas view: wires its DOM controls, kicks
// off the immediate sync + 10s poll + focus refetch. Called once at
// bootstrap, regardless of which view is initially visible — polling
// keeps running for the view you are not looking at (design.md §5, same
// rationale as shopping-view.js/projects-view.js).
export function initIdeasView(me) {
  initIdeasStore({
    onDelete: (id) => deleteIdeaOptimistic(id),
  });

  wireAddForm();
  wireToastDismiss();

  setCurrentUserId(me.id);

  syncIdeasFromServer();
  setInterval(syncIdeasFromServer, POLL_INTERVAL_MS);
  window.addEventListener('focus', syncIdeasFromServer);
}

// prefetchIdeasList() lets main.js kick off GET /api/v1/ideas
// concurrently with GET /api/v1/me, before initIdeasView() runs.
export function prefetchIdeasList() {
  prefetchIdeas();
}

function wireAddForm() {
  const urlInput = document.getElementById('add-idea-url');
  const addBtn = document.getElementById('add-idea-btn');
  const addGroup = document.getElementById('add-idea-group');
  const addError = document.getElementById('add-idea-error');

  function showError(message) {
    addError.textContent = message;
    addError.hidden = false;
    addGroup.classList.add('has-error');
  }

  function clearError() {
    addError.hidden = true;
    addGroup.classList.remove('has-error');
  }

  async function submitNewIdea() {
    const raw = urlInput.value;
    if (raw.trim().length === 0) {
      showError('Enganxa un enllaç abans de desar.');
      urlInput.focus();
      return;
    }

    urlInput.disabled = true;
    addBtn.disabled = true;
    try {
      await addIdeaOptimistic(raw);
      clearError();
      urlInput.value = '';
    } catch (err) {
      const message = err && err.message ? err.message : 'S’ha produït un error inesperat.';
      showError(message);
    } finally {
      urlInput.disabled = false;
      addBtn.disabled = false;
      // §8.3: the input empties immediately and regains focus so the
      // user can keep pasting more links without waiting for the scrape
      // (ADR-03) — this happens on both success and failure, matching
      // AddItemInput's existing "success" pattern (proposal.md §8.3).
      urlInput.focus();
    }
  }

  addBtn.addEventListener('click', submitNewIdea);
  urlInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      submitNewIdea();
    }
  });
}

function wireToastDismiss() {
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      dismissToast();
    }
  });
}
