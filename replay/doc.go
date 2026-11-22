// Package replay est la couche de PUBLICATION du décodeur de film (ADR 0034 D-1 et D-7) : il
// assemble l'artefact de rejeu 2D (vue du dessus) d'un match à partir des faits décodés du
// film — trajectoires, géométrie de carte, tirs, équipement, véhicules, morts et effets de
// kill, objectifs, identités, couverture. Assemblage pur — aucun accès DB ni HTTP ; les
// lectures du film sont faites par les couches `grammar` et `facts`.
//
// Le document (ReplayDocument) est produit HORS LIGNE par `internal/replaybuild` (étape
// post-sync, `levelup backfill-replay`, `cmd/replay-build`) à partir des SEULS chunks du film
// (zéro capture Cheat Engine) et servi tel quel par l'API. C'est
// délibérément un DTO d'artefact bespoke (pas un type canonical) : c'est une charge utile
// de rendu, versionnée par SchemaVersion pour la compat client.
//
// Repère : les positions sont en MÈTRES MONDE. Ce paragraphe disait le contraire
// (« l'échelle/offset absolus ne sont PAS garantis », handoff ALL_PLAYERS_TRAJECTORIES) et
// c'était vrai AVANT profile/map_bounds.go : le film ne porte que des indices de quantum, et
// tant que les bornes du BSP manquaient, la déquantification employait celles de Cliffhanger
// pour toutes les cartes — d'où un facteur d'échelle arbitraire. Depuis, cmd/replay-build
// EXIGE la carte (`-map`) et refuse de produire un artefact sans ses bornes.
//
// CE QUE ÇA AUTORISE, et pourquoi la correction n'est pas cosmétique : le fond de carte
// figé (`MapBackground`) est calé dans ce même repère, donc il se superpose au rejeu.
// Contrôle sur 000d5950 : les bornes du rejeu tombent dans le cadre de `ridgeline.png`
// (cf. TestMapBackground_DonneesReelles). Le client garde son auto-cadrage via Bounds.
package replay
