// projects-view.js — projects view: event wiring for the add form and
// state buttons, and the syncProjectsFromServer() poll cycle (same 10s
// cycle already used by shopping-view.js for items — no second sync
// mechanism, AC-06).
//
// Split out of the old projects-main.js (single-page-per-route era) so
// the SPA bootstrap in main.js can mount this view without also owning
// identity resolution — GET /api/v1/me now happens exactly once, in
// main.js, not once per view.

import {
  initProjectsStore,
  addProjectOptimistic,
  changeProjectState,
  deleteProjectOptimistic,
  syncProjectsFromServer,
  setCurrentUserId,
  prefetchProjects,
} from './projects-store.js';
import { dismissToast } from './projects-render.js';
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
  return { ok: true, name: trimmed };
}

// initProjectsView() mounts the projects view: wires its DOM controls,
// kicks off the immediate sync + 10s poll + focus refetch. Called once at
// bootstrap, regardless of which view is initially visible — polling
// keeps running for the view you are not looking at (design.md §5, same
// rationale as shopping-view.js).
export function initProjectsView(me) {
  initProjectsStore({
    onChangeState: (id, state) => changeProjectState(id, state),
    onDelete: (id) => deleteProjectOptimistic(id),
  });

  wireAddForm();
  wireToastDismiss();

  setCurrentUserId(me.id);

  syncProjectsFromServer();
  setInterval(syncProjectsFromServer, POLL_INTERVAL_MS);
  window.addEventListener('focus', syncProjectsFromServer);
}

// prefetchProjectsList() lets main.js kick off GET /api/v1/projects
// concurrently with GET /api/v1/me, before initProjectsView() runs.
export function prefetchProjectsList() {
  prefetchProjects();
}

function wireAddForm() {
  const nameInput = document.getElementById('add-project-name');
  const budgetInput = document.getElementById('add-project-budget');
  const targetDateInput = document.getElementById('add-project-target-date');
  const urlInput = document.getElementById('add-project-url');
  const addBtn = document.getElementById('add-project-btn');
  const addGroup = document.getElementById('add-project-group');
  const addError = document.getElementById('add-project-error');

  function showError(message) {
    addError.textContent = message;
    addError.hidden = false;
    addGroup.classList.add('has-error');
  }

  function clearError() {
    addError.hidden = true;
    addGroup.classList.remove('has-error');
  }

  async function submitNewProject() {
    const raw = nameInput.value;
    const clientCheck = validateNameClientSide(raw);
    if (!clientCheck.ok) {
      showError(clientCheck.message);
      nameInput.focus();
      return;
    }

    nameInput.disabled = true;
    addBtn.disabled = true;
    try {
      await addProjectOptimistic(raw, budgetInput.value, targetDateInput.value, urlInput.value);
      clearError();
      nameInput.value = '';
      budgetInput.value = '';
      targetDateInput.value = '';
      urlInput.value = '';
    } catch (err) {
      const message = err && err.message ? err.message : t('errorGeneric');
      showError(message);
    } finally {
      nameInput.disabled = false;
      addBtn.disabled = false;
      nameInput.focus();
    }
  }

  addBtn.addEventListener('click', submitNewProject);
  nameInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      submitNewProject();
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
