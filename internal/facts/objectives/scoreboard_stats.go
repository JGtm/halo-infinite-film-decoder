package objectives

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// scoreboard_stats.go — LES COMPTEURS DU TABLEAU DES SCORES QUE LE FILM REMPLIT ET QUI ONT UN NOM.
//
// Le statborg replique, pour chaque joueur, une ligne de composants a deux valeurs (A et B). Les
// quatre premiers compteurs publies (score personnel, frags, morts, assistances) sont nommes dans
// score.go ; cette table nomme les autres emplacements que le film remplit. Chaque nom vient de la
// confrontation du compteur a une lecture independante du meme film (la bibliotheque libfilmshell :
// son emplacement n est 2 x composant + cote) et, quand le film le permet, a ce que le fil des
// temps forts ou les vies donnent du meme joueur.

// Les noms publies de [ScoreboardStats], ceux des cles de `scoreTimeline.players[].stats`.
const (
	StatSuicides         = "suicides"
	StatBetrayals        = "betrayals"
	StatMaxSpree         = "maxSpree"
	StatHeadshots        = "headshots"
	StatMeleeKills       = "meleeKills"
	StatGrenadeKills     = "grenadeKills"
	StatPowerWeaponKills = "powerWeaponKills"
	StatSpawns           = "spawns"
)

// Les noms publies des compteurs du tableau propres au VIP ([ScoreboardStatsOf]). Les durees sont des DIXIEMES de seconde,
// comme le film les porte.
const (
	StatTimeAsVip        = "timeAsVip"
	StatLongestTimeAsVip = "longestTimeAsVip"
	StatMaxSpreeAsVip    = "maxSpreeAsVip"
)

// ScoreboardStat est un compteur nomme du tableau des scores.
type ScoreboardStat struct {
	// Name est le nom publie, en anglais simple.
	Name string
	// Component est son emplacement : composant du statborg et cote de la paire de valeurs.
	Component StatComponent
	// Peak dit que la valeur est un maximum (la plus longue serie), pas un compte : le cumul des
	// manches n'a pas de sens, et seule sa courbe par manche se publie.
	Peak bool
}

// ScoreboardStats rend les compteurs nommes publies avec le score, dans l'ordre de leur composant.
func ScoreboardStats() []ScoreboardStat {
	// Chaque unite d'un compteur nomme est un evenement du joueur (une mort, une reapparition...) : la
	// borne par pas des compteurs unitaires ecarte les emissions mal alignees qui portent un saut.
	held := func(comp int, side bool) StatComponent {
		return StatComponent{Comp: comp, SideB: side, Unitary: true, HoldOnClear: true, Joint: true}
	}
	// Un compteur d'eliminations nomme est en plus borne par les eliminations de la manche ([withinKillBound]).
	heldKills := func(comp int, side bool) StatComponent {
		c := held(comp, side)
		c.KillBounded = true
		return c
	}
	// La valeur A du composant 5 n'est pas un compteur (elle recule d'une vie a l'autre) : seule
	// sa valeur B, la serie maximale, est lue.
	spree := StatComponent{Comp: 5, SideB: true, HoldOnClear: true, KillBounded: true}
	return []ScoreboardStat{
		{Name: StatSuicides, Component: held(4, false)},
		{Name: StatBetrayals, Component: held(4, true)},
		{Name: StatMaxSpree, Component: spree, Peak: true},
		{Name: StatHeadshots, Component: heldKills(6, false)},
		{Name: StatMeleeKills, Component: heldKills(8, false)},
		{Name: StatGrenadeKills, Component: heldKills(8, true)},
		{Name: StatPowerWeaponKills, Component: heldKills(9, false)},
		{Name: StatSpawns, Component: held(10, true)},
	}
}

// ScoreboardStatsOf rend les compteurs nommes du tableau d'un match de la famille `objectiveType` ([ScoreboardStats] et ceux
// que le mode ajoute). Le sens d'un emplacement dependant du mode, les compteurs de mode ne se lisent que quand l'appelant
// reconnait la famille ; une famille sans compteur de mode (ou vide) rend exactement [ScoreboardStats].
//
// VIP : `comp 23 B` est le temps passe en VIP (`TimeAsVip`), `comp 23 A` le plus long temps d'une traite
// (`LongestTimeAsVip`, un maximum) et `comp 22 B` la meilleure serie d'eliminations faite en VIP (`BestKillingSpreeAsVip`, un
// maximum, borne par les eliminations de la manche). Les durees sont des dixiemes de seconde ; contre l'API, sur les trois
// films VIP du parc (24 joueurs), `timeAsVip` et la serie sont exacts pour 24 joueurs, `longestTimeAsVip` pour 22 et a un
// dixieme de seconde pour les deux autres.
func ScoreboardStatsOf(objectiveType string) []ScoreboardStat {
	base := ScoreboardStats()
	mode := modeScoreboardStats[objectiveType]
	if len(mode) == 0 {
		return base
	}
	return append(base, mode...)
}

// modeScoreboardStats : par famille, les compteurs du tableau que le mode ajoute. Les emplacements sont ceux de
// [namedStatSlots] : aucun ne coincide avec un compteur de [ScoreboardStats] (composants 4 a 10).
var modeScoreboardStats = map[string][]ScoreboardStat{
	ObjectiveTypeVip: {
		{Name: StatMaxSpreeAsVip, Component: StatComponent{Comp: 22, SideB: true, HoldOnClear: true, KillBounded: true}, Peak: true},
		{Name: StatLongestTimeAsVip, Component: StatComponent{Comp: 23, HoldOnClear: true, Joint: true}, Peak: true},
		{Name: StatTimeAsVip, Component: StatComponent{Comp: 23, SideB: true, HoldOnClear: true, Joint: true}},
	},
}

// tousLesCompteursDuTableau rend les compteurs de base et ceux de chaque mode : la table dont se servent les lectures qui
// ne savent pas de quelle famille est le film (garde par les eliminations, comptes des emissions ecartees).
func tousLesCompteursDuTableau() []ScoreboardStat {
	out := ScoreboardStats()
	for _, mode := range modeScoreboardStats {
		out = append(out, mode...)
	}
	return out
}

// rawSeries rend les emissions de l'emplacement `c` par slot puis par manche, avec les regles de
// [StatComponent.HoldOnClear] et [StatComponent.Joint] quand il les demande.
func rawSeries(recs []types.StatRecord, c StatComponent, teams bool,
	cons *ReplisALaConsultation) map[int]map[int][]types.ScorePoint {
	if !c.HoldOnClear {
		if c.DropClears && hasCounterChannels(c.Comp) {
			recs = withoutNegativeChannels(recs, c, cons)
		}
		raw := rawSeriesByRound(recs, c.key(), teams, cons)
		if c.DropClears {
			for slot, byRound := range raw {
				for round, pts := range byRound {
					byRound[round] = dropClearPoints(pts, slot, round, c, cons)
				}
			}
		}
		return raw
	}
	pairs := withinKillBound(recs, c, valuePairsByRound(recs, c, teams, cons), cons)
	raw := make(map[int]map[int][]types.ScorePoint, len(pairs))
	for slot, byRound := range pairs {
		raw[slot] = map[int][]types.ScorePoint{}
		for round, pts := range byRound {
			raw[slot][round] = c.series(pts, slot, round, cons)
		}
	}
	return raw
}

// valuePairsByRound range par slot puis par manche les couples (A, B) de l'emplacement `c`, hors manches exclues ; une emission qui
// porte un canal negatif est ecartee et notee dans `cons`.
func valuePairsByRound(recs []types.StatRecord, c StatComponent, teams bool,
	cons *ReplisALaConsultation) map[int]map[int][]valuePair {
	bornes := ResolveRoundBounds(recs)
	pairs := map[int]map[int][]valuePair{}
	for _, r := range recs {
		v, ok := r.Comps[c.Comp]
		if !ok || IsTeamSlot(r.Slot) != teams || bornes.Excludes(r) {
			continue
		}
		if v.A < 0 || v.B < 0 || v.C < 0 || v.D < 0 {
			cons.noterEmissionJetee(c.key(), r)
			continue
		}
		if pairs[r.Slot] == nil {
			pairs[r.Slot] = map[int][]valuePair{}
		}
		pairs[r.Slot][r.Round] = append(pairs[r.Slot][r.Round], valuePair{r.TimeMS, v.A, v.B})
	}
	return pairs
}

// withoutNegativeChannels ecarte les enregistrements dont l'un des canaux compteurs du composant de `c` est negatif ([negativeCounter]), et
// note chacun dans `cons`. Les valeurs d'un composant sortent de la MEME emission : un canal aberrant prouve l'emission mal alignee, et
// la valeur de l'autre canal ne vaut rien non plus. Mesure : sur b1b62f38, des enregistrements fortuits (frags A = -31744, morts B = 16)
// apres la derniere vraie mort (7) la prolongeaient par [longestRun] et publiaient 16 morts pour 7 a l'API.
// LES CANAUX B ET D DES ASSISTANCES SONT DES POINTS, PAS DES COMPTES : leur signe est legitime (-100, -66, -533 avec chaque assistance d'un
// joueur de 00ba2e1c), les ecarter perdait les assistances de 700 joueurs. Elle ne vaut que pour les compteurs du tableau des
// scores : les series qui nomment les evenements (pont d'identite, drapeau, crane) gardent leur domaine.
func withoutNegativeChannels(recs []types.StatRecord, c StatComponent, cons *ReplisALaConsultation) []types.StatRecord {
	out := make([]types.StatRecord, 0, len(recs))
	for _, r := range recs {
		if v, ok := r.Comps[c.Comp]; ok && negativeCounter(c.Comp, v) {
			cons.noterEmissionJetee(c.key(), r)
			continue
		}
		out = append(out, r)
	}
	return out
}

// hasCounterChannels dit que le composant `comp` est un compteur de base dont [negativeCounter] juge les canaux.
func hasCounterChannels(comp int) bool { return comp == coreKillsComp || comp == coreAssistsComp }

// negativeCounter dit qu'une emission du composant `comp` porte un canal compteur negatif. Les frags et les morts (composant 2) sont
// les canaux A et B ; les assistances (composant 3) sont le canal A, et son canal C n'est jamais un score signe.
func negativeCounter(comp int, v types.StatValue) bool {
	switch comp {
	case coreKillsComp:
		return v.A < 0 || v.B < 0
	case coreAssistsComp:
		return v.A < 0 || v.C < 0
	}
	return false
}

// ClearsDropped rend le meme emplacement lu avec la regle du vidage d'une ligne ([StatComponent.DropClears]).
func (c StatComponent) ClearsDropped() StatComponent {
	c.DropClears = true
	return c
}

// dropClearPoints ecarte d'une suite d'emissions d'un slot et d'une manche celles qui retombent a zero apres une valeur
// positive, et note chacune dans `cons`. La suite est triee par instant.
func dropClearPoints(pts []types.ScorePoint, slot, round int, c StatComponent, cons *ReplisALaConsultation) []types.ScorePoint {
	slices.SortStableFunc(pts, func(x, y types.ScorePoint) int { return cmp.Compare(x.TimeMS, y.TimeMS) })
	out := make([]types.ScorePoint, 0, len(pts))
	positive := false
	for _, p := range pts {
		if p.Value == 0 && positive {
			cons.noterEmissionJetee(c.key(), types.StatRecord{TimeMS: p.TimeMS, Slot: slot, Round: round})
			continue
		}
		positive = positive || p.Value > 0
		out = append(out, p)
	}
	return out
}

// series rend les points de la valeur lue de `c` parmi des emissions d'un slot et d'une manche. Chaque
// emission ecartee (retour a zero, suite jointe) est notee dans `cons` comme le filtre de domaine.
func (c StatComponent) series(pts []valuePair, slot, round int, cons *ReplisALaConsultation) []types.ScorePoint {
	slices.SortStableFunc(pts, func(x, y valuePair) int { return cmp.Compare(x.t, y.t) })
	kept := withoutClears(pts, c.SideB)
	if c.Joint {
		kept = jointRun(kept)
	}
	if cons != nil && len(kept) < len(pts) {
		for _, p := range ecartees(pts, kept) {
			cons.noterEmissionJetee(c.key(), types.StatRecord{TimeMS: p.t, Slot: slot, Round: round})
		}
	}
	out := make([]types.ScorePoint, 0, len(kept))
	for _, p := range kept {
		out = append(out, types.ScorePoint{TimeMS: p.t, Slot: slot, Value: p.read(c.SideB)})
	}
	return out
}

// ecartees rend les emissions de `all` qui ne sont pas dans `kept`, sous-suite de `all` dans le meme ordre.
func ecartees(all, kept []valuePair) []valuePair {
	var out []valuePair
	k := 0
	for _, p := range all {
		if k < len(kept) && kept[k] == p {
			k++
			continue
		}
		out = append(out, p)
	}
	return out
}

// withoutClears ecarte, parmi des emissions triees par instant, celles dont la valeur lue retombe a
// zero apres une valeur positive : le vidage de la ligne d'un joueur qui part, pas une valeur.
func withoutClears(pts []valuePair, sideB bool) []valuePair {
	out := make([]valuePair, 0, len(pts))
	positive := false
	for _, p := range pts {
		v := p.read(sideB)
		if v == 0 && positive {
			continue
		}
		positive = positive || v > 0
		out = append(out, p)
	}
	return out
}

// valuePair est une emission d'un composant : son instant et ses deux valeurs.
type valuePair struct {
	t    int
	a, b int64
}

// read rend la valeur A ou B de l'emission.
func (p valuePair) read(sideB bool) int64 {
	if sideB {
		return p.b
	}
	return p.a
}

// jointRun rend, dans l'ordre du temps, la plus longue suite d'emissions dont les DEUX valeurs ne
// decroissent pas ; a longueur egale, celle qui finit le plus haut l'emporte.
//
// Les deux valeurs d'un composant sortent de la meme emission : une emission dont l'une recule est
// mal alignee, et l'autre valeur ne vaut rien non plus.
func jointRun(kept []valuePair) []valuePair {
	n := len(kept)
	if n == 0 {
		return nil
	}
	length, prev := make([]int, n), make([]int, n)
	best := 0
	for i := range kept {
		length[i], prev[i] = 1, -1
		for j := range i {
			if kept[j].a <= kept[i].a && kept[j].b <= kept[i].b && length[j]+1 > length[i] {
				length[i], prev[i] = length[j]+1, j
			}
		}
		if length[i] > length[best] || (length[i] == length[best] && kept[i].a+kept[i].b >= kept[best].a+kept[best].b) {
			best = i
		}
	}
	out := make([]valuePair, length[best])
	for i, k := length[best]-1, best; i >= 0; i, k = i-1, prev[k] {
		out[i] = kept[k]
	}
	return out
}
