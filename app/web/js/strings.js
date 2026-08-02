// strings.js — lightweight dictionary for text built dynamically in JS
// (toasts, aria-labels, validation errors). Static HTML markup is left in
// Catalan and delegated to the browser's native translation feature —
// this file only covers the DOM nodes that browsers cannot reliably
// re-translate after insertion (Chrome does it inconsistently, Safari
// mostly doesn't). No build step, no external i18n library, by design.

const DICT = {
  ca: {
    boxShopping: 'A comprar',
    boxPantry: 'Rebost',
    emptyShopping: 'Res per comprar ara mateix.',
    emptyPantry: 'El rebost encara està buit.',
    addedBy: (name) => `Afegit per ${name}`,
    movedBy: (name) => `Mogut per ${name}`,
    moveTo: (item, target) => `Moure ${item} a ${target}`,
    movedAddedBy: (added, moved) => `. Afegit per ${added}, mogut per ${moved}`,
    deleteItem: (item) => `Eliminar ${item}`,
    closeToast: 'Tancar avís',
    announceMoved: (item, target) => `${item} mogut a ${target}.`,
    announceMovedBy: (item, target, who) => `${item} mogut a ${target} per ${who}.`,
    errorEmptyName: "Escriu un nom abans d'afegir.",
    errorTooLong: (len) => `Massa llarg — màxim 200 caràcters (portes ${len}/200).`,
    errorInvalidChars: 'Aquest nom conté caràcters no vàlids.',
    errorGeneric: "S'ha produït un error inesperat.",
    errorNeedsLogin: 'Cal iniciar sessió.',
  },
  pt: {
    boxShopping: 'A comprar',
    boxPantry: 'Despensa',
    emptyShopping: 'Nada para comprar neste momento.',
    emptyPantry: 'A despensa ainda está vazia.',
    addedBy: (name) => `Adicionado por ${name}`,
    movedBy: (name) => `Movido por ${name}`,
    moveTo: (item, target) => `Mover ${item} para ${target}`,
    movedAddedBy: (added, moved) => `. Adicionado por ${added}, movido por ${moved}`,
    deleteItem: (item) => `Eliminar ${item}`,
    closeToast: 'Fechar aviso',
    announceMoved: (item, target) => `${item} movido para ${target}.`,
    announceMovedBy: (item, target, who) => `${item} movido para ${target} por ${who}.`,
    errorEmptyName: 'Escreve um nome antes de adicionar.',
    errorTooLong: (len) => `Demasiado longo — máximo 200 caracteres (tens ${len}/200).`,
    errorInvalidChars: 'Este nome contém caracteres inválidos.',
    errorGeneric: 'Ocorreu um erro inesperado.',
    errorNeedsLogin: 'É necessário iniciar sessão.',
  },
};

// Supported locales, most-specific-first is not needed here since we only
// key off the primary language subtag (navigator.language: "pt-BR" -> "pt").
const DEFAULT_LOCALE = 'ca';

// resolveLocale prefers <html lang="..."> over navigator.language because
// Chrome/Edge's native "Translate this page" rewrites documentElement.lang
// to the target language when the user picks one from the translate
// bubble — navigator.language never changes (it stays the OS/browser
// setting), so it alone can't detect that choice. Firefox/Safari's
// translation features do not rewrite lang, so navigator.language remains
// the fallback for those.
function resolveLocale() {
  const htmlLang = document.documentElement.lang;
  const navLang = navigator.language || DEFAULT_LOCALE;
  const lang = (htmlLang || navLang).slice(0, 2).toLowerCase();
  return DICT[lang] ? lang : DEFAULT_LOCALE;
}

export let locale = resolveLocale();

const localeChangeListeners = new Set();

export function onLocaleChange(fn) {
  localeChangeListeners.add(fn);
  return () => localeChangeListeners.delete(fn);
}

// Chrome Translate mutates documentElement.lang once the translation
// finishes applying (and again if the user reverts to "Show original") —
// observe that attribute so already-mounted dynamic text can be
// re-rendered to match without a page reload.
new MutationObserver(() => {
  const next = resolveLocale();
  if (next === locale) return;
  locale = next;
  localeChangeListeners.forEach((fn) => fn(locale));
}).observe(document.documentElement, { attributeFilter: ['lang'] });

export function t(key, ...args) {
  const entry = DICT[locale][key] ?? DICT[DEFAULT_LOCALE][key];
  return typeof entry === 'function' ? entry(...args) : entry;
}
