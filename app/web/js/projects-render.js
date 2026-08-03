// projects-render.js — renderProjects(), renderProjectRow(),
// renderEmptyProjectsState() and a local Toast, mirroring the discipline
// already proved in render.js: zero innerHTML with user data anywhere —
// every node carrying user-controlled text (name, budget, target_date,
// display names) is built with document.createElement + .textContent
// only (EC-08/NFR-02). List container is cleared with replaceChildren(),
// never innerHTML = ''.
//
// design.md §7/ADR-03: state change is a text/badge update, no FLIP, no
// movement animation — prefers-reduced-motion needs no special handling
// here because there is no motion to reduce (NFR-08 not applicable).

export { showToast, dismissToast } from './toast.js';

const STATE_LABELS = {
  idea: 'Idea',
  decidit: 'Decidit',
  fet: 'Fet',
};

const STATE_ORDER = ['idea', 'decidit', 'fet'];

export function stateLabel(state) {
  return STATE_LABELS[state] || state;
}

const list = () => document.getElementById('projects-list');
const count = () => document.getElementById('projects-count');

export function renderEmptyProjectsState(target) {
  const wrap = document.createElement('li');
  wrap.style.listStyle = 'none';
  const el = document.createElement('div');
  el.className = 'empty-state';
  el.id = 'empty-projects';
  const icon = document.createElement('span');
  icon.className = 'icon';
  icon.setAttribute('aria-hidden', 'true');
  icon.textContent = '🏠';
  const msg = document.createElement('p');
  msg.className = 'msg';
  msg.textContent = 'Cap projecte encara — afegeix una idea de compra gran o un projecte de casa.';
  el.appendChild(icon);
  el.appendChild(msg);
  wrap.appendChild(el);
  target.appendChild(wrap);
}

function renderAvatars(project) {
  const wrap = document.createElement('span');
  wrap.className = 'avatars';
  wrap.setAttribute('aria-hidden', 'true');

  const addedAvatar = document.createElement('span');
  addedAvatar.className = 'avatar';
  if (project.added_by) {
    addedAvatar.title = `Afegit per ${project.added_by.display_name}`;
    addedAvatar.textContent = project.added_by.avatar_emoji;
  }
  wrap.appendChild(addedAvatar);

  const updatedByDifferent = project.last_updated_by
    && (!project.added_by || project.last_updated_by.id !== project.added_by.id);
  if (updatedByDifferent) {
    const link = document.createElement('span');
    link.className = 'avatar-link';
    link.textContent = '↩';
    wrap.appendChild(link);
    const updatedAvatar = document.createElement('span');
    updatedAvatar.className = 'avatar';
    updatedAvatar.title = `Actualitzat per ${project.last_updated_by.display_name}`;
    updatedAvatar.textContent = project.last_updated_by.avatar_emoji;
    wrap.appendChild(updatedAvatar);
  }
  return wrap;
}

function formatDate(isoDate) {
  // Same "AAAA-MM-DD" → "DD/MM/AAAA" format already used elsewhere in
  // Niu's Catalan locale (design.md §7: "format de data ja establert").
  const [y, m, d] = isoDate.split('-');
  return `${d}/${m}/${y}`;
}

// renderStateSelector builds a three-option control (never a "next
// state" toggle, AC-09) so the user can move a project to ANY of the
// three states directly, in any direction.
function renderStateSelector(project, handlers) {
  const wrap = document.createElement('div');
  wrap.className = 'project-state-selector';
  wrap.setAttribute('role', 'group');
  wrap.setAttribute('aria-label', `Estat de ${project.name}`);

  STATE_ORDER.forEach((state) => {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = `state-badge state-${state}`;
    if (state === project.state) {
      btn.classList.add('is-current');
      btn.setAttribute('aria-pressed', 'true');
    } else {
      btn.setAttribute('aria-pressed', 'false');
    }
    btn.textContent = stateLabel(state);
    btn.setAttribute('aria-label', `Marcar ${project.name} com a ${stateLabel(state)}`);
    btn.addEventListener('click', () => handlers.onChangeState(project.id, state));
    wrap.appendChild(btn);
  });

  return wrap;
}

export function renderProjectRow(project, handlers) {
  const li = document.createElement('li');
  li.style.listStyle = 'none';

  const row = document.createElement('div');
  row.className = 'project-row';
  row.dataset.id = String(project.id);

  const name = document.createElement('span');
  name.className = 'project-name';
  name.textContent = project.name; // EC-08/NFR-02: textContent only, never innerHTML
  row.appendChild(name);

  row.appendChild(renderStateSelector(project, handlers));

  if (project.budget) {
    const budget = document.createElement('span');
    budget.className = 'project-budget';
    budget.textContent = project.budget;
    row.appendChild(budget);
  }

  if (project.target_date) {
    const targetDate = document.createElement('span');
    targetDate.className = 'project-target-date';
    targetDate.textContent = formatDate(project.target_date);
    row.appendChild(targetDate);
  }

  row.appendChild(renderAvatars(project));

  const del = document.createElement('button');
  del.type = 'button';
  del.className = 'delete-btn';
  del.setAttribute('aria-label', `Eliminar ${project.name}`);
  del.textContent = '🗑';
  del.addEventListener('click', () => handlers.onDelete(project.id));
  row.appendChild(del);

  li.appendChild(row);
  return li;
}

export function renderProjects(projectsList, handlers) {
  const target = list();
  if (!target) return;

  if (count()) count().textContent = `(${projectsList.length})`;

  target.replaceChildren();

  if (projectsList.length === 0) {
    renderEmptyProjectsState(target);
  } else {
    projectsList.forEach((p) => target.appendChild(renderProjectRow(p, handlers)));
  }
}

// ================= Toast =================
//
// Implementation moved to toast.js (shared with render.js — both views'
// toasts now live on the same #toast-wrap node in the merged SPA shell,
// so a single shared timer replaces what used to be two independent
// module-level timers, one per page). Re-exported above for callers that
// already `import { showToast } from './projects-render.js'`.
