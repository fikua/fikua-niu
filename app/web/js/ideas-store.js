// ideas-store.js — in-memory `ideas` array, addIdeaOptimistic() (shows a
// temporary "Recuperant..." card immediately, §8.2 Estat D),
// deleteIdeaOptimistic(), and syncIdeasFromServer() following exactly the
// same poll + refetch-on-focus cycle already implemented for items/
// projects (AC-06) — no second sync mechanism invented. A pending idea
// discovered via polling (added by the OTHER user, or resolved by the
// background worker pool) is how AC-06 convergence naturally covers the
// "Recuperant..." -> resolved transition without any extra client logic.

import * as api from './ideas-api.js';
import { renderIdeas, showToast } from './ideas-render.js';
import { announceIdeaSaving, announceIdeaResolved } from './a11y.js';

let ideasList = [];
let currentUserId = null;
let handlers = null;

export function setCurrentUserId(id) {
  currentUserId = id;
}

export function initIdeasStore(h) {
  handlers = h;
}

function doRender() {
  renderIdeas(ideasList, handlers);
}

export function getIdeasList() {
  return ideasList;
}

// addIdeaOptimistic posts the new idea (server responds 201 immediately
// with preview_status='pending', ADR-03) and unshifts it to the top of
// the grid right away — the "Recuperant..." card (Estat D) IS the pending
// idea's own real state, not a separate client-side placeholder, so there
// is nothing to reconcile later: the next poll simply carries the same
// row forward with an updated preview_status.
export async function addIdeaOptimistic(rawUrl) {
  announceIdeaSaving();
  const created = await api.addIdea(rawUrl);
  ideasList.unshift(created);
  doRender();
  return created;
}

export async function deleteIdeaOptimistic(id) {
  const idea = ideasList.find((i) => i.id === id);
  ideasList = ideasList.filter((i) => i.id !== id);
  doRender();

  try {
    await api.deleteIdea(id);
  } catch (err) {
    if (idea) showToast(`No s'ha pogut eliminar aquesta idea. Torna-ho a provar.`);
    await syncIdeasFromServer();
  }
}

// prefetchIdeas lets main.js kick off GET /api/v1/ideas concurrently with
// GET /api/v1/me, same rationale as store.js's prefetchItems /
// projects-store.js's prefetchProjects (avoid an extra serialized round
// trip on a slow connection).
let prefetchedIdeasPromise = null;

export function prefetchIdeas() {
  prefetchedIdeasPromise = api.getIdeas();
  prefetchedIdeasPromise.catch(() => {});
}

// syncIdeasFromServer() polls GET /api/v1/ideas and diffs by id +
// preview_status against the previously known state, to announce a
// resolution (ready/partial/failed) via aria-live (AC-01/AC-02/AC-03/
// AC-11) once a "pending" idea settles — including ideas resolved or
// added by the OTHER user (AC-06 convergence), which is exactly how a
// second household member finds out an idea is done "Recuperant..."
// without any dedicated push mechanism (ADR-03).
export async function syncIdeasFromServer() {
  let serverIdeas;
  try {
    const pending = prefetchedIdeasPromise;
    prefetchedIdeasPromise = null;
    serverIdeas = await (pending || api.getIdeas());
  } catch {
    return; // transient network failure — try again on the next tick
  }

  const previousById = new Map(ideasList.map((i) => [i.id, i]));

  for (const idea of serverIdeas) {
    const prev = previousById.get(idea.id);
    if (!prev) continue; // brand new idea, no resolution announcement needed
    const statusChanged = prev.preview_status !== idea.preview_status;
    if (statusChanged && prev.preview_status === 'pending') {
      announceIdeaResolved(idea.preview_status);
    }
  }

  ideasList = serverIdeas;
  doRender();
}
