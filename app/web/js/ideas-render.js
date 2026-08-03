// ideas-render.js — renderIdeas(), renderIdeaCard(), renderEmptyIdeasState()
// implementing the four IdeaCard states (proposal.md §8.2): A (ready), B
// (failed/fallback), C (partial), D (pending, "Recuperant..."), mapped
// exactly to the four preview_status values. Same discipline as
// render.js/projects-render.js: ZERO innerHTML with external or
// user-controlled data anywhere in this file — title, description, and
// url are always built with document.createElement + .textContent
// (EC-11/NFR-01). List container is cleared with replaceChildren(),
// never innerHTML = ''.
//
// proposal.md §8.2 Estat A/C displays an <img> with the recovered
// og:image, fetched directly by the browser from the external host
// (design.md §6.1: image_url is the full external URL). CSP's
// `img-src` was relaxed to `'self' https:` to allow this (see
// internal/httpapi/middleware.go) — fetchsafe already guarantees
// image_url only ever holds an http(s) URL that passed SSRF validation.

import { t } from './strings.js';

export { showToast, dismissToast } from './toast.js';

const grid = () => document.getElementById('ideas-grid');
const count = () => document.getElementById('ideas-count');

export function renderEmptyIdeasState(target) {
  const wrap = document.createElement('li');
  wrap.style.listStyle = 'none';
  const el = document.createElement('div');
  el.className = 'empty-state';
  el.id = 'empty-ideas';
  const icon = document.createElement('span');
  icon.className = 'icon';
  icon.setAttribute('aria-hidden', 'true');
  icon.textContent = '💡';
  const msg = document.createElement('p');
  msg.className = 'msg';
  msg.textContent = 'Cap idea desada encara — enganxa un enllaç per començar.';
  el.appendChild(icon);
  el.appendChild(msg);
  wrap.appendChild(el);
  target.appendChild(wrap);
}

// domainOf extracts just the hostname from a URL, for the discreet
// "🔗 domini" treatment in Estat A/C (proposal.md §8.2) — falls back to
// the full string if URL parsing fails for any reason (should not happen
// for a URL that was already accepted server-side, but this file must
// never throw on untrusted data).
function domainOf(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return url;
  }
}

function renderLinkRow(idea, { fullURL }) {
  const link = document.createElement('a');
  link.className = 'idea-link';
  link.href = idea.url;
  link.target = '_blank';
  link.rel = 'noopener noreferrer';

  const icon = document.createElement('span');
  icon.setAttribute('aria-hidden', 'true');
  icon.textContent = '🔗 ';
  link.appendChild(icon);

  const text = document.createElement('span');
  text.textContent = fullURL ? idea.url : domainOf(idea.url); // textContent only, never innerHTML (EC-11/NFR-01)
  link.appendChild(text);

  return link;
}

function renderAvatar(idea) {
  const wrap = document.createElement('span');
  wrap.className = 'avatars';
  wrap.setAttribute('aria-hidden', 'true');

  const avatar = document.createElement('span');
  avatar.className = 'avatar';
  if (idea.added_by) {
    avatar.title = `Afegit per ${idea.added_by.display_name}`;
    avatar.textContent = idea.added_by.avatar_emoji;
  }
  wrap.appendChild(avatar);
  return wrap;
}

function renderDeleteButton(idea, handlers) {
  const del = document.createElement('button');
  del.type = 'button';
  del.className = 'delete-btn';
  del.setAttribute('aria-label', `Eliminar la idea «${idea.title || domainOf(idea.url)}»`);
  del.textContent = '🗑';
  del.addEventListener('click', (e) => {
    e.stopPropagation();
    handlers.onDelete(idea.id);
  });
  return del;
}

// renderIdeaCardBody builds the shared skeleton (container, avatar,
// delete button) — callers fill in the state-specific middle (image,
// title, description) before this returns.
function newCardContainer(idea) {
  const li = document.createElement('li');
  li.style.listStyle = 'none';

  const card = document.createElement('div');
  card.className = 'idea-card';
  card.dataset.id = String(idea.id);
  card.setAttribute('role', 'group');
  card.setAttribute('aria-label', idea.title || domainOf(idea.url));

  li.appendChild(card);
  return { li, card };
}

// renderIdeaCardReady is Estat A (proposal.md §8.2): title, image,
// description, link, avatar, delete — everything the scrape recovered.
function renderIdeaCardReady(idea, handlers) {
  const { li, card } = newCardContainer(idea);
  card.classList.add('idea-card-ready');

  if (idea.image_url) {
    const img = document.createElement('img');
    img.className = 'idea-image';
    img.src = idea.image_url; // external URL by design (design.md §6.1)
    img.alt = ''; // decorative (confirmed at Stage 1.5 human gate, proposal.md §8.6)
    img.loading = 'lazy';
    card.appendChild(img);
  }

  if (idea.title) {
    const title = document.createElement('p');
    title.className = 'idea-title';
    title.textContent = idea.title; // textContent only, never innerHTML (EC-11/NFR-01)
    card.appendChild(title);
  }

  if (idea.description) {
    const desc = document.createElement('p');
    desc.className = 'idea-description';
    desc.textContent = idea.description; // textContent only, never innerHTML (EC-11/NFR-01)
    card.appendChild(desc);
  }

  card.appendChild(renderLinkRow(idea, { fullURL: false }));
  card.appendChild(renderAvatar(idea));
  card.appendChild(renderDeleteButton(idea, handlers));

  return li;
}

// renderIdeaCardFailed is Estat B (proposal.md §8.2): no error styling —
// a discreet substitute icon, the domain as the title (never an empty
// string or "Sense títol"), a non-accusatory message, and the FULL url
// (the only way to identify the content, AC-02).
function renderIdeaCardFailed(idea, handlers) {
  const { li, card } = newCardContainer(idea);
  card.classList.add('idea-card-failed');

  const icon = document.createElement('span');
  icon.className = 'idea-fallback-icon';
  icon.setAttribute('aria-hidden', 'true');
  icon.textContent = '🔗';
  card.appendChild(icon);

  const title = document.createElement('p');
  title.className = 'idea-title';
  title.textContent = idea.title || domainOf(idea.url);
  card.appendChild(title);

  const message = document.createElement('p');
  message.className = 'idea-fallback-message';
  message.textContent = 'No s’ha pogut generar una previsualització d’aquest enllaç.';
  card.appendChild(message);

  card.appendChild(renderLinkRow(idea, { fullURL: true }));
  card.appendChild(renderAvatar(idea));
  card.appendChild(renderDeleteButton(idea, handlers));

  return li;
}

// renderIdeaCardPartial is Estat C (proposal.md §8.2): same anatomy as
// Estat A, each absent zone (image, title, or description) is simply
// omitted — never a placeholder/broken-looking gap.
function renderIdeaCardPartial(idea, handlers) {
  const { li, card } = newCardContainer(idea);
  card.classList.add('idea-card-partial');

  if (idea.image_url) {
    const img = document.createElement('img');
    img.className = 'idea-image';
    img.src = idea.image_url; // external URL by design (design.md §6.1)
    img.alt = '';
    img.loading = 'lazy';
    card.appendChild(img);
  }

  if (idea.title) {
    const title = document.createElement('p');
    title.className = 'idea-title';
    title.textContent = idea.title;
    card.appendChild(title);
  }

  if (idea.description) {
    const desc = document.createElement('p');
    desc.className = 'idea-description';
    desc.textContent = idea.description;
    card.appendChild(desc);
  }

  card.appendChild(renderLinkRow(idea, { fullURL: false }));
  card.appendChild(renderAvatar(idea));
  card.appendChild(renderDeleteButton(idea, handlers));

  return li;
}

// renderIdeaCardPending is Estat D (proposal.md §8.2): a shimmering
// placeholder + "Recuperant la previsualització..." + the domain, so the
// user recognizes which idea it is even before it resolves.
function renderIdeaCardPending(idea, handlers) {
  const { li, card } = newCardContainer(idea);
  card.classList.add('idea-card-pending');

  const shimmer = document.createElement('div');
  shimmer.className = 'idea-shimmer';
  shimmer.setAttribute('aria-hidden', 'true');
  card.appendChild(shimmer);

  const message = document.createElement('p');
  message.className = 'idea-pending-message';
  message.textContent = 'Recuperant la previsualització…';
  card.appendChild(message);

  card.appendChild(renderLinkRow(idea, { fullURL: false }));
  card.appendChild(renderAvatar(idea));
  card.appendChild(renderDeleteButton(idea, handlers));

  return li;
}

export function renderIdeaCard(idea, handlers) {
  switch (idea.preview_status) {
    case 'ready':
      return renderIdeaCardReady(idea, handlers);
    case 'partial':
      return renderIdeaCardPartial(idea, handlers);
    case 'pending':
      return renderIdeaCardPending(idea, handlers);
    case 'failed':
    default:
      return renderIdeaCardFailed(idea, handlers);
  }
}

export function renderIdeas(ideasList, handlers) {
  const target = grid();
  if (!target) return;

  if (count()) count().textContent = `(${ideasList.length})`;

  target.replaceChildren();

  if (ideasList.length === 0) {
    renderEmptyIdeasState(target);
  } else {
    ideasList.forEach((idea) => target.appendChild(renderIdeaCard(idea, handlers)));
  }
}
