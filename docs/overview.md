# Niu — Visió de producte

> Document de producte. El *què* i el *per què*. El *com* viu a
> [architecture.md](architecture.md).

## El problema

Una parella comparteix una casa. Les coses de casa — què falta comprar,
què ja hi ha al rebost — viuen avui en caps, notes soltes i missatges de
WhatsApp que s'esborren. Ningú sap del cert si queda oli fins que obre
l'armari.

No és un problema greu. És un fregament petit i constant.

## Per a qui

Exactament dues persones. Una parella, una llar.

Això no és una limitació temporal a superar: és una **decisió de
disseny**. Assumir dos usuaris coneguts elimina el registre, els rols, les
invitacions, la recuperació de contrasenya i la gestió d'usuaris. Tot això
és complexitat que no aporta res a una casa.

## Què fa la v1

Una llista de la compra compartida amb dues caixes:

- **A comprar** — el que falta.
- **Rebost** — el que ja tenim a casa.

Seleccionar un ítem el mou d'una caixa a l'altra. Al súper, marques el que
compres i passa al rebost. A casa, quan s'acaba una cosa, la tornes a la
llista.

Els dos veuen la mateixa llista. Cada ítem mostra qui l'ha tocat.

## Què no fa la v1

Deliberadament fora d'abast:

- Registre, recuperació de contrasenya, gestió d'usuaris.
- Múltiples llars, rols, permisos.
- Notificacions push.
- Apps mòbils natives.
- Tasques, menús, despeses compartides.

## Futur possible

Sense compromís ni data:

- Tasques de casa i qui les fa.
- Planificació de menús.
- Despeses compartides.
- Capes de gamificació (ratxes, punts).

La taula `events` recull dades des del primer dia perquè aquestes coses
siguin possibles després sense una migració dolorosa. Però **no es
construeix res d'això fins que hi hagi ús real**. Gamificar abans que
existeixi l'hàbit és disseny especulatiu.

## Com sabrem que funciona

No hi ha mètriques de producte, ni analítica, ni objectius de conversió.
És una app per a dues persones.

L'única prova que serveix: **que l'utilitzeu al súper sense pensar-hi**.
Si acabeu tornant al WhatsApp, no ha funcionat.

## Principis

1. **Càlida, no infantil.** Ha de fer gust d'obrir. Res de blau
   corporatiu.
2. **Ràpida per damunt de tot.** Afegir un ítem ha de costar menys que
   escriure'l a una nota.
3. **Segura tot i ser simple.** Login senzill no vol dir fluix.
4. **Sense fricció d'entrada.** Res de configuració, res d'onboarding.
