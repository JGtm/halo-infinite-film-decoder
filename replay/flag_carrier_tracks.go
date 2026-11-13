package replay

import (
	"context"
	"log/slog"
	"slices"
	"strconv"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
)

// flag_carrier_tracks.go — LA POSITION DU PORTEUR SUR LES PISTES PUBLIEES.
//
// Deux helpers, et une seule question : ou dessiner l objet porte a un instant donne ? Ils sont
// PARTAGES par les deux consommateurs du calque drapeau — `attachFlagCarryPositions`
// (flag_carries.go, la position de prise et de lacher) et `closeByFreeLives`/`flagFreeDropInside`
// (flag_objects.go, le lacher volontaire date par la vie libre de l objet). Sortis de
// flag_carries.go le 2026-09-06, deplacement PUR (le fichier franchissait les 500 lignes).

// tracksByXUID range les pistes publiees par joueur. Rend aussi les SLOTS AMBIGUS : ceux dont les
// vies sans nom ont ete REFUSEES au repli ci-dessous (cf. les deux gardes).
//
// UNE VIE SANS NOM N'EST PAS UNE ABSENCE — MEME PRINCIPE QUE LE GATE DES PORTAGES DE CRANE
// (schema 43). Depuis le schema 36 (« une track = une vie ») un slot recycle publie PLUSIEURS
// pistes, et le fil des morts n'en nomme pas toujours toutes : une vie sans nom est une
// PRESENCE SANS IDENTITE PUBLIEE, pas une absence du porteur. N'indexer que les pistes NOMMEES
// faisait alors echouer `pointOfXUIDAt` sur des prises pourtant couvertes par une piste, et
// `attachFlagCarryPositions` les comptait `NoTrack` — mesure du corpus temoin : `bcb6d393`
// perd 9 prises sur 16 (toutes celles du slot 536 apres 2736, dont la vie est publiee sans nom)
// alors que l'artefact du parc, ne les voyant qu'a travers une piste unique par slot, les
// portait toutes.
//
// L'IDENTITE VIENT DU PONT CANONIQUE, PAS D'UNE DEDUCTION LOCALE, et la resolution elle-meme vit
// dans `published_tracks.go` (`xuidOfPublishedTrack`), partagee avec les trois autres lecteurs :
// trois copies de la meme regle, c'est trois occasions de la faire diverger — et c'est deja
// arrive une fois (regle n°6 du depot, garde-rail `published_tracks_guard_test.go`).
//
// # DEUX GARDES, DEUX POPULATIONS DISTINCTES — ET AUCUNE NE REND L'AUTRE INUTILE
//
// Elles viennent de deux revues et ne jugent pas sur la meme matiere. Les confondre serait la
// meilleure facon d'en perdre une :
//
//	le PONT EPURE   `OwnerReport.NamingBridge()` (revue VIES-R1, constat C2) retire du pont les
//	(en amont)      slots que `ownersFromLives` a vus revendiques par DEUX VIES NOMMEES
//	                differentes. Sa matiere est `own.lives`, les vies DECOUPEES — y compris
//	                celles que `minPoints` n'a pas publiees. Le lecteur ne peut plus servir un
//	                nom arbitraire : il n'a plus de quoi l'enfreindre.
//	`slotAmbigu`    (revue DUREES-R1, constat C1) juge sur les vies PUBLIEES, et attrape ce que
//	(ici)           le pont ne peut pas voir : le cas ou le pont nomme un joueur que les vies
//	                nommees du slot NE PORTENT PAS. C'est une contradiction entre le pont et le
//	                document, pas une collision entre deux vies nommees — elle n'entre donc
//	                jamais dans `SlotAmbiguous`.
//
// LES DEUX SE COMPTENT SOUS LE MEME COMPTEUR, et c'est delibere : du point de vue du calque le
// fait publie est le meme — « une prise est tombee sur un slot dont je refuse de deviner
// l'occupant ». Le refus se voit dans `coverage.flagCarries.ambiguousSlot` (servi jusqu'au
// contrat) et dans un `slog.Warn` ; sans eux, de la matiere que le calque renonce a exploiter
// disparaitrait en silence.
//
// Declenchement mesure : `084a804d` slot 734 — `[5872..6981]` nommee A, `[7123..7158]` SANS NOM,
// `[7457..7591]` nommee B ; 9 artefacts du parc sur 106 portent au moins un slot en collision.
//
// CE QU'AUCUNE DES DEUX NE PEUT ATTRAPER, et c'est ecrit pour que personne ne le croie : un slot
// occupe par A (vie nommee) puis par B dont AUCUNE vie n'est nommee sort avec un seul xuid nomme,
// et la vie de B est rangee sous A. Le document publie ne porte rien qui le distingue d'une vie
// de A coupee par un trou de replication ; le trancher demanderait de dater le pont, ce que
// `OwnerReport` ne fait pas.
func tracksByXUID(tracks []Track, slotXUID map[uint32]uint64,
	slotAmbiguous map[uint32]bool, fb *fallback.Compteur) (map[string][]Track, []uint32) {
	nommes := namedXUIDsBySlot(tracks)
	out := map[string][]Track{}
	var ambigus []uint32
	vus := map[uint32]bool{}
	for _, t := range tracks {
		if t.XUID == "" && replierRefuse(t.Slot, slotXUID, slotAmbiguous, nommes) {
			if !vus[t.Slot] {
				vus[t.Slot], ambigus = true, append(ambigus, t.Slot)
			}
			continue
		}
		xuid := xuidOfPublishedTrack(t, slotXUID)
		if xuid == "" {
			// REPLI NOMME ET COMPTE (D14) : distinct du refus NOMME voisin (`ambigus`), qui dit
			// « deux vies se partagent le slot ». Ici le pont ne nomme RIEN, et la piste sort du
			// calque sans laisser de trace : c'est ce silence que le compte rompt.
			fb.Declenche(fallback.NomPisteDrapeauSansPontEcartee)
			continue // le pont ne nomme pas ce slot : aucun porteur a inventer
		}
		out[xuid] = append(out[xuid], t)
	}
	slices.Sort(ambigus)
	return out, ambigus
}

// replierRefuse dit si le repli d'une vie SANS NOM sur le joueur du pont est refuse pour ce slot,
// par l'une ou l'autre des deux gardes (cf. l'en-tete de `tracksByXUID`).
//
// LE REFUS N'EST PAS L'ABSENCE. Un slot que le pont ne nomme pas du tout n'est pas « refuse » :
// il n'y avait rien a preter, et le compter gonflerait `ambiguousSlot` de tous les slots muets du
// film. On ne compte que ce que le calque RENONCE a exploiter.
func replierRefuse(slot uint32, slotXUID map[uint32]uint64, slotAmbiguous map[uint32]bool,
	nommes map[uint32]map[string]struct{}) bool {
	if slotAmbiguous[slot] {
		// Le pont EPURE ne porte plus ce slot : sans la garde amont, le repli aurait eu lieu.
		return true
	}
	x, ok := slotXUID[slot]
	if !ok || x == 0 {
		return false // rien a preter : ce n'est pas un refus
	}
	return slotAmbigu(nommes[slot], x)
}

// namedXUIDsBySlot rend, par slot, l'ensemble des identites que ses vies PUBLIEES portent.
func namedXUIDsBySlot(tracks []Track) map[uint32]map[string]struct{} {
	out := map[uint32]map[string]struct{}{}
	for _, t := range tracks {
		if t.XUID == "" {
			continue
		}
		if out[t.Slot] == nil {
			out[t.Slot] = map[string]struct{}{}
		}
		out[t.Slot][t.XUID] = struct{}{}
	}
	return out
}

// slotAmbigu dit si les vies NOMMEES d'un slot interdisent de preter ses vies sans nom au joueur
// que le pont designe : plusieurs occupants nommes, ou un occupant nomme qui n'est pas celui-la.
func slotAmbigu(noms map[string]struct{}, pont uint64) bool {
	if len(noms) == 0 {
		return false // rien ne contredit le pont
	}
	if len(noms) > 1 {
		return true // deux joueurs se partagent le slot : la vie sans nom n'appartient a personne
	}
	_, accord := noms[strconv.FormatUint(pont, 10)]
	return !accord
}

// logFlagAmbiguousSlots journalise les slots dont les vies sans nom ont ete refusees au repli.
// Un AVERTISSEMENT, pas une information : c'est de la matiere que le calque renonce a exploiter,
// et le seul endroit ou la limite ci-dessus se voit en production.
func logFlagAmbiguousSlots(ctx context.Context, slots []uint32) {
	if len(slots) == 0 {
		return
	}
	slog.WarnContext(ctx, "rejeu : vies sans nom refusees au calque drapeau — slot partage par plusieurs joueurs",
		"slots", slots, "nombre", len(slots))
}

// flagCarrierPointToleranceMS est l'ecart maximal, en millisecondes, entre l'instant d'une prise ou
// d'un lacher et le point publie qui le localise.
//
// C'EST UNE DUREE, JUGEE SUR LES INSTANTS EXACTS DES POINTS : le film ne replique que ce qui change,
// un corps immobile n'a donc aucun point sur plusieurs dizaines a centaines de millisecondes. Une
// tolerance d'une image de la grille valait 100 ms ou 33 ms selon le pas, et une tolerance convertie
// en images (arrondie) acceptait de 230 ms a 299 ms selon le pas. 200 ms laisse a un corps immobile le
// temps de ne rien repliquer sans aller chercher la position d'un autre instant du match.
const flagCarrierPointToleranceMS = 200

// pointOfXUIDAt rend le point PUBLIE qui localise un joueur a l'instant `atUS` (microsecondes du
// film) : le point le plus proche en temps EXACT, a au plus [flagCarrierPointToleranceMS], et a
// egalite celui d'avant — un corps immobile reste ou son dernier point le met. Il vient de la MEME
// VIE que l'instant sans qu'il faille le tester : une piste qui NAIT entre un point et l'instant a son
// premier point plus pres de l'instant que lui, donc c'est elle qui l'emporte (mort puis reapparition dans
// la fenetre). Rend (_, false) quand aucun point ne convient : le drapeau n'aurait pas de position a
// dessiner, et on prefere ne rien poser.
func pointOfXUIDAt(tracks []Track, atUS uint64) (Point, bool) {
	const tolUS = uint64(flagCarrierPointToleranceMS) * 1000
	var best Point
	var bestD uint64
	var bestBefore, found bool
	for _, tr := range tracks {
		for _, p := range tr.Points {
			d, before := p.us-atUS, false
			if p.us <= atUS {
				d, before = atUS-p.us, true
			}
			if d > tolUS || (found && (d > bestD || (d == bestD && (!before || bestBefore)))) {
				continue
			}
			best, bestD, bestBefore, found = p, d, before, true
		}
	}
	return best, found
}

// finDeLAxeUS est l'instant (microsecondes du film) du dernier point publie de toutes les pistes : la fin de
// l'axe du rejeu, la ou les instants demandes au dela s'arretent.
func finDeLAxeUS(tracks []Track) uint64 {
	var fin uint64
	for _, tr := range tracks {
		if n := len(tr.Points); n > 0 {
			fin = max(fin, tr.Points[n-1].us)
		}
	}
	return fin
}
