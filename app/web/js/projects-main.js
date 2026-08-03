// projects-main.js — entry point for web/projects.html: event wiring for
// the add form and state buttons, immediate syncProjectsFromServer() +
// setInterval(10000) + window focus listener (same cycle already wired
// in main.js for items — no second sync mechanism, AC-06), and
// GET /api/v1/me on load.

import * as api from './projects-api.js';
import { initAnnounce } from './a11y.js';
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
import { logout } from './auth.js';
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

async function main() {
  // Same rationale as main.js: resolve identity before mounting any UI;
  // kick off GET /api/v1/projects in parallel with GET /api/v1/me so the
  // two share one round trip's worth of latency (NFR-06 budget).
  prefetchProjects();

  let me;
  try {
    me = await api.getMe();
  } catch {
    return;
  }

  const liveRegion = document.getElementById('live-region');
  initAnnounce(liveRegion);

  initProjectsStore({
    onChangeState: (id, state) => changeProjectState(id, state),
    onDelete: (id) => deleteProjectOptimistic(id),
  });

  wireAddForm();
  wireToastDismiss();
  wireLogoutButton();

  setCurrentUserId(me.id);
  const nameEl = document.getElementById('user-name');
  const avatarEl = document.getElementById('user-avatar');
  if (nameEl) nameEl.textContent = me.display_name;
  if (avatarEl) avatarEl.textContent = me.avatar_emoji;

  syncProjectsFromServer();
  setInterval(syncProjectsFromServer, POLL_INTERVAL_MS);
  window.addEventListener('focus', syncProjectsFromServer);
}

function wireAddForm() {
  const nameInput = document.getElementById('add-project-name');
  const budgetInput = document.getElementById('add-project-budget');
  const targetDateInput = document.getElementById('add-project-target-date');
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
      await addProjectOptimistic(raw, budgetInput.value, targetDateInput.value);
      clearError();
      nameInput.value = '';
      budgetInput.value = '';
      targetDateInput.value = '';
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

function wireLogoutButton() {
  const btn = document.getElementById('logout-btn');
  if (!btn) return;
  btn.addEventListener('click', () => {
    logout();
  });
}

main();
