// tabs.js — setActivePanel() for the mobile tab bar (§8.3.2, breakpoint
// 768px). Both panels remain in the DOM at all times (needed for FLIP
// continuity, design.md §8); only .panel.is-visible/display decides what
// is shown.

let activePanel = 'shopping';

export function getActivePanel() {
  return activePanel;
}

export function setActivePanel(panel) {
  activePanel = panel;

  document.querySelectorAll('.tab').forEach((tab) => {
    const isActive = tab.dataset.panel === panel;
    tab.classList.toggle('is-active', isActive);
    tab.setAttribute('aria-selected', String(isActive));
  });

  const boxShopping = document.getElementById('box-shopping');
  const boxPantry = document.getElementById('box-pantry');
  boxShopping.classList.toggle('is-visible', panel === 'shopping');
  boxPantry.classList.toggle('is-visible', panel === 'pantry');

  updateTabindexForInactivePanel();
}

// §8.7 mobile tab order: rows (and their delete buttons) in the inactive
// box are removed from the tab order entirely while hidden, so keyboard
// navigation never jumps to visually hidden content.
export function updateTabindexForInactivePanel() {
  document.querySelectorAll('#box-shopping .item-row').forEach((el) => {
    el.tabIndex = activePanel === 'shopping' ? 0 : -1;
  });
  document.querySelectorAll('#box-pantry .item-row').forEach((el) => {
    el.tabIndex = activePanel === 'pantry' ? 0 : -1;
  });
}

export function wireTabs() {
  document.querySelectorAll('.tab').forEach((tab) => {
    tab.addEventListener('click', () => setActivePanel(tab.dataset.panel));
  });
}
