package replay

// tracks_visees_seules.go — LES VISEES SANS POSITION ENTRENT DANS LE PLEIN DEBIT DE LEUR VIE.
//
// Le film replique la visee `i21` d un biped aussi dans des records qui ne portent aucune position
// (`grammar.ScanBipedAimOnly`) : le nuage des positions ne les voit pas, et les colonnes de visee de
// `tracks[].fullRate` (aimT, h, p) ne portaient que les visees des records de position. Chaque visee sans
// position se pose sur la vie de son slot dont les bornes (celles du registre, `bornesDesVies`) couvrent son
// instant, et les deux suites se fusionnent dans l ordre du temps : les colonnes restent UNE suite de visees,
// triee, au debit du film. A instant egal (a la milliseconde), la visee du record de position passe la
// premiere. Les positions, les points, les lacunes et leurs colonnes ne bougent pas.
//
// CE QUI NE SE POSE PAS SE COMPTE, ET PAR CAUSE ([ViseesSeulesCoverage]). Les visees lues sont les
// `coverage.vehicles.aimReads` ; chacune est posee, ou ecartee pour l une de deux raisons :
//
//   - HORS DE L AXE : son instant precede l origine de la grille du document, il n a pas d echantillon
//     a lui ;
//   - SANS VIE : aucune vie PUBLIEE de son slot ne couvre son instant (avant la premiere position du slot,
//     entre deux vies, apres la derniere, sur une vie que le seuil de publication a refusee).
//
// Le document interne du sync ne retient pas la pleine cadence : rien ne s y pose ni ne s y compte.

import (
	"cmp"
	"context"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// ViseesSeulesCoverage est le sort des visees sans position dans les colonnes de visee de la pleine cadence.
// Posee a l identique sur tout document publie : posees + HORS DE L AXE + SANS VIE = les lectures du film.
type ViseesSeulesCoverage struct {
	// FullRateAimOnlySamples : les visees posees dans `tracks[].fullRate`, comprises dans `fullRateAimSamples`.
	FullRateAimOnlySamples int `json:"fullRateAimOnlySamples"`
	// AimOnlyBeforeOrigin : les visees ecartees parce que leur instant precede l origine de la grille.
	AimOnlyBeforeOrigin int `json:"aimOnlyBeforeOrigin"`
	// AimOnlyNoLife : les visees ecartees faute de vie publiee qui couvre leur instant.
	AimOnlyNoLife int `json:"aimOnlyNoLife"`
}

// vieIndexee designe une vie : son slot et son indice dans `bornes[slot]`.
type vieIndexee struct {
	slot uint32
	vie  int
}

// repartirLesViseesSeules pose chaque visee sans position sur la vie de son slot dont les bornes couvrent son
// instant (`from` <= instant <= `to`, la premiere trouvee : les vies d un slot sont disjointes). Elle rend les
// visees par vie, dans l ordre ou elles sont entrees, et compte celles que leur instant place avant l origine
// de la grille. Une visee sans aucune vie qui la couvre n est pas rendue : l appelant la compte par difference.
func repartirLesViseesSeules(visees []grammar.BipedAim, bornes map[uint32][]lifeSpan, origineUS uint64,
	cov *ViseesSeulesCoverage) map[vieIndexee][]grammar.BipedAim {
	out := map[vieIndexee][]grammar.BipedAim{}
	for _, v := range visees {
		if v.TimestampUS < origineUS {
			cov.AimOnlyBeforeOrigin++
			continue
		}
		t := int64(v.TimestampUS) //nolint:gosec // horloge du film en microsecondes
		for i, s := range bornes[v.Slot] {
			if t >= s.from && t <= s.to {
				k := vieIndexee{v.Slot, i}
				out[k] = append(out[k], v)
				break
			}
		}
	}
	return out
}

// viseeAbsolue est une visee du plein debit en absolu : instant en ms depuis l origine de la grille, quanta
// bruts d `i21`.
type viseeAbsolue struct {
	t    int64
	h, p int32
}

// fusionnerLesViseesSeules ajoute les visees sans position aux colonnes de visee de la vie, dans l ordre du
// temps. Un recepteur nil (document interne) et une liste vide ne changent rien.
func (d *pleinDebit) fusionnerLesViseesSeules(seules []grammar.BipedAim, origineUS uint64) {
	if d == nil || len(seules) == 0 {
		return
	}
	lues := make([]viseeAbsolue, 0, len(d.piste.AimT))
	var t int64
	var h, p int32
	for i := range d.piste.AimT {
		t, h, p = t+d.piste.AimT[i], h+d.piste.H[i], p+d.piste.P[i]
		lues = append(lues, viseeAbsolue{t: t, h: h, p: p})
	}
	ajoutees := make([]viseeAbsolue, 0, len(seules))
	for _, v := range seules {
		ajoutees = append(ajoutees, viseeAbsolue{
			t: int64((v.TimestampUS - origineUS) / fullRateTimeUnitUS),
			h: int32(v.YawRaw), p: int32(v.PitchRaw), //nolint:gosec // quanta de 12 et 11 bits
		})
	}
	slices.SortStableFunc(ajoutees, func(a, b viseeAbsolue) int { return cmp.Compare(a.t, b.t) })
	d.poserLesVisees(fusionTriee(lues, ajoutees))
}

// fusionTriee fusionne deux suites triees par instant ; a instant egal, `a` passe la premiere.
func fusionTriee(a, b []viseeAbsolue) []viseeAbsolue {
	out := make([]viseeAbsolue, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if b[j].t < a[i].t {
			out = append(out, b[j])
			j++
			continue
		}
		out = append(out, a[i])
		i++
	}
	out = append(out, a[i:]...)
	return append(out, b[j:]...)
}

// poserLesVisees recode une suite absolue en colonnes d ecarts (cf. tracks_plein_debit.go) et la pose sur la
// piste, avec les derniers absolus que `ajouter` reprendrait.
func (d *pleinDebit) poserLesVisees(vs []viseeAbsolue) {
	n := len(vs)
	d.piste.AimT, d.piste.H, d.piste.P = make([]int64, n), make([]int32, n), make([]int32, n)
	var prev viseeAbsolue
	for i, v := range vs {
		d.piste.AimT[i], d.piste.H[i], d.piste.P[i] = v.t-prev.t, v.h-prev.h, v.p-prev.p
		prev = v
	}
	d.aimT, d.h, d.p = prev.t, prev.h, prev.p
}

// journaliser dit ce que la pose des visees sans position a ecarte : un ecart a zero est un silence, un ecart
// non nul est ce que le film porte sans que les colonnes de visee puissent le dire (cf. l en-tete).
func (c ViseesSeulesCoverage) journaliser(ctx context.Context, matchID string) {
	if c.AimOnlyBeforeOrigin == 0 && c.AimOnlyNoLife == 0 {
		return
	}
	slog.InfoContext(ctx, "rejeu : visees sans position ecartees de la pleine cadence",
		"match_id", matchID, "posees", c.FullRateAimOnlySamples,
		"avantLOrigine", c.AimOnlyBeforeOrigin, "sansVie", c.AimOnlyNoLife)
}
