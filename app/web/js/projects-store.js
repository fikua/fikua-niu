// projects-store.js — in-memory `projects` array, changeProjectState()
// (absolute PATCH, any direction — AC-09), addProjectOptimistic(),
// deleteProjectOptimistic(), and syncProjectsFromServer() following
// exactly the same poll + refetch-on-focus cycle already implemented for
// items in store.js (AC-06) — no second sync mechanism invented.

import * as api from './projects-api.js';
import { renderProjects, showToast } from './projects-render.js';
import { announceProjectStateChange } from './a11y.js';

let projectsList = [];
let currentUserId = null;
let handlers = null;

export function setCurrentUserId(id) {
  currentUserId = id;
}

export function initProjectsStore(h) {
  handlers = h;
}

function doRender() {
  renderProjects(projectsList, handlers);
}

export function getProjectsList() {
  return projectsList;
}

export async function addProjectOptimistic(rawName, rawBudget, rawTargetDate, rawURL) {
  const created = await api.addProject(rawName, rawBudget, rawTargetDate, rawURL);
  projectsList.unshift(created);
  doRender();
  return created;
}

// changeProjectState issues an absolute PATCH (never a "next state"
// toggle, AC-09) and updates local state with the server's response —
// the server is the source of truth for last_updated_by/updated_at, so
// there is no optimistic local mutation to roll back here.
export async function changeProjectState(id, newState) {
  const project = projectsList.find((p) => p.id === id);
  if (!project) return;

  try {
    const updated = await api.patchProjectState(id, newState);
    applyServerProject(updated);
    doRender();
    announceProjectStateChange(updated.name, updated.state);
  } catch (err) {
    showToast(`No s'ha pogut canviar l'estat de «${project.name}». Torna-ho a provar.`);
  }
}

export async function deleteProjectOptimistic(id) {
  const project = projectsList.find((p) => p.id === id);
  projectsList = projectsList.filter((p) => p.id !== id);
  doRender();

  try {
    await api.deleteProject(id);
  } catch (err) {
    if (project) showToast(`No s'ha pogut eliminar «${project.name}». Torna-ho a provar.`);
    await syncProjectsFromServer();
  }
}

function applyServerProject(serverProject) {
  const idx = projectsList.findIndex((p) => p.id === serverProject.id);
  if (idx === -1) {
    projectsList.push(serverProject);
  } else {
    projectsList[idx] = serverProject;
  }
}

// prefetchProjects lets main entry points kick off GET /api/v1/projects
// concurrently with GET /api/v1/me, same rationale as store.js's
// prefetchItems (avoid an extra serialized round trip on a slow
// connection).
let prefetchedProjectsPromise = null;

export function prefetchProjects() {
  prefetchedProjectsPromise = api.getProjects();
  prefetchedProjectsPromise.catch(() => {});
}

// syncProjectsFromServer() polls GET /api/v1/projects and diffs by id +
// state + updated_at against the previously known state, to detect
// remote changes and announce them via aria-live (AC-12) with the exact
// "{nom} ara està {estat}" wording, for changes not made by the current
// user (avoids re-announcing our own change once confirmed).
export async function syncProjectsFromServer() {
  let serverProjects;
  try {
    const pending = prefetchedProjectsPromise;
    prefetchedProjectsPromise = null;
    serverProjects = await (pending || api.getProjects());
  } catch {
    return; // transient network failure — try again on the next tick
  }

  const previousById = new Map(projectsList.map((p) => [p.id, p]));

  for (const p of serverProjects) {
    const prev = previousById.get(p.id);
    if (!prev) continue; // brand new project, no remote-change announcement needed
    const stateChanged = prev.state !== p.state;
    const updatedAtChanged = prev.updated_at !== p.updated_at;
    if ((stateChanged || updatedAtChanged) && p.last_updated_by && p.last_updated_by.id !== currentUserId) {
      announceProjectStateChange(p.name, p.state);
    }
  }

  projectsList = serverProjects;
  doRender();
}
