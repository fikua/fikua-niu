# Fonts — Nunito autoallotjada

`Nunito-Variable-Latin.woff2` — Nunito variable, subconjunt llatí, 39 KB.
Llicència SIL Open Font License 1.1 (permet allotjament propi i
redistribució).

## Per què un fitxer variable

`proposal.md` §8.2 demanava dos pesos: Regular (400) i Bold (700). Un
fitxer variable cobreix tot l'eix 400–700, així que és **una sola petició
en lloc de dues** — millor per a NFR-06 (<1s en 3G) que el pla original
de dos fitxers estàtics.

La regla a `app.css` ho declara amb `font-weight: 400 700` i
`format("woff2-variations")`.

## Per què autoallotjada i no un enllaç a Google Fonts

Dos motius, tots dos vinculants:

1. La CSP de l'aplicació no permet `font-src` cap a hosts externs
   (`PLAN.md` §3, S3). Un enllaç a un CDN seria bloquejat pel navegador.
2. Cada càrrega de pàgina filtraria l'adreça IP dels usuaris a un tercer.
   Per a una app domèstica de dues persones això no té cap justificació.

El binari Go serveix aquest fitxer des de `embed.FS`, com la resta
d'estàtics.

## Actualitzar-la

Descarregar el subconjunt llatí des de Google Fonts (la font és OFL, no
cal cap acord) i substituir el fitxer mantenint el nom. Si es canvia el
nom, cal actualitzar la regla `@font-face` a `app.css`.
