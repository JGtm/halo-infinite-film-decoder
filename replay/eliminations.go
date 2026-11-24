package replay

// eliminations.go — LA CONSTRUCTION DU CALQUE DES ÉLIMINATIONS, PURE : elle lit des calques déjà publiés (le tableau des
// scores, les actions de score, les médailles) et n'ouvre ni film ni base. Le contrat de forme, les états et leurs
// conditions sont dans document_eliminations.go.

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/objectives"
)

// Noms et fenêtres du calque. Les deux fenêtres sont des écarts d'horloge entre deux écritures du même fait, pas des
// seuils de décision : une action de score et un pas de compteur tombent à une image de la grille par défaut près l'un
// de l'autre, une médaille à deux images près d'un pas. Elles s'écrivent en millisecondes et se convertissent avec le
// pas du document ([windowFrames]) : la fenêtre en images est la durée arrondie vers le haut au pas, donc elle couvre au moins la durée déclarée, jamais zéro image, et dépasse d'au plus une image moins une milliseconde.
const (
	// killScoreActionName est le nom de la réponse de score que le jeu envoie au tueur ; sa cible est la victime.
	killScoreActionName = "killPersonalScore"
	// medalPerfectName est le nom de la médaille du frag parfait (la voisine `Perfection` n'en est pas un).
	medalPerfectName = "Perfect"
	// victimWindowMS est l'écart maximal, en millisecondes, entre un pas d'éliminations et l'action de score de sa victime :
	// une image de la grille par défaut.
	victimWindowMS = DefaultFrameIntervalMS
	// perfectWindowMS est l'écart maximal, en millisecondes, entre une médaille `Perfect` et son pas d'éliminations : deux
	// images de la grille par défaut.
	perfectWindowMS = 2 * DefaultFrameIntervalMS
)

// windowFrames convertit une fenêtre en millisecondes en nombre d'images du pas `stepMS` (0 : la grille par défaut),
// arrondi vers le haut : une fenêtre ne tombe jamais à zéro image.
func windowFrames(windowMS, stepMS int) int {
	if stepMS <= 0 {
		stepMS = DefaultFrameIntervalMS
	}
	return (windowMS + stepMS - 1) / stepMS
}

// eliminationInput est ce que le calque lit : les trois calques publiés dont il dérive et le témoin de la garde.
type eliminationInput struct {
	Players []PlayerScore
	Actions []ScoreAction
	Medals  []Medal
	// HeadshotsGuarded dit que la garde des compteurs a écarté des émissions du compteur de tirs à la tête
	// (`coverage.score.statsGuarded`).
	HeadshotsGuarded bool
	// StepMS est le pas de temps du document (`frameIntervalMs`) ; 0 : la grille par défaut.
	StepMS int
}

// elimStep est un pas d'une courbe cumulée : à l'image t, la valeur monte de n unités.
type elimStep struct{ t, n int }

// stepsOfTotal rend les pas d'une courbe cumulée : un par point dont la valeur dépasse toutes les précédentes.
func stepsOfTotal(ticks []ScoreTick) []elimStep {
	var out []elimStep
	prev := 0
	for _, k := range ticks {
		if k.V > prev {
			out = append(out, elimStep{t: k.T, n: k.V - prev})
			prev = k.V
		}
	}
	return out
}

// headVerdict est la conclusion sur la marque de tir à la tête d'un pas.
type headVerdict int

const (
	headVerdictNo headVerdict = iota
	headVerdictYes
	headVerdictNoCounter
	headVerdictAmbiguous
	headVerdictGuarded
)

// headVerdictsOf juge chaque pas d'éliminations contre les montées du compteur de tirs à la tête, et rend en second le
// nombre d'unités de montée qu'aucune élimination ne porte. `counter` est la série du compteur, nil quand le joueur n'en
// a pas.
func headVerdictsOf(steps []elimStep, counter *ScoreSeries, guarded bool) ([]headVerdict, int) {
	out := make([]headVerdict, len(steps))
	if counter == nil {
		for i := range out {
			out[i] = headVerdictNoCounter
		}
		return out, 0
	}
	rises := map[int]int{}
	for _, s := range stepsOfTotal(counter.Total) {
		rises[s.t] = s.n
	}
	orphans := 0
	for i, s := range steps {
		m := rises[s.t]
		delete(rises, s.t)
		switch {
		case m == 0 && guarded:
			out[i] = headVerdictGuarded
		case m == 0:
			out[i] = headVerdictNo
		case m == s.n:
			out[i] = headVerdictYes
		default:
			out[i] = headVerdictAmbiguous
			orphans += max(m-s.n, 0)
		}
	}
	for _, m := range rises {
		orphans += m
	}
	return out, orphans
}

// perfectStepsOf rattache chaque médaille `Perfect` (donnée par son image) au pas du joueur situé à deux images au
// plus (`window`), quand il est unique, et rend en premier les pas marqués. Rend ensuite le nombre de médailles ambiguës (plusieurs
// pas candidats, ou un pas qui porte un autre nombre de médailles que d'éliminations) et celui des médailles sans pas.
func perfectStepsOf(steps []elimStep, medalFrames []int, window int) (marked []bool, ambiguous, orphans int) {
	attached := make([]int, len(steps))
	for _, tm := range medalFrames {
		candidate, n := 0, 0
		for i, s := range steps {
			if abs(s.t-tm) <= window {
				candidate = i
				n++
			}
		}
		switch n {
		case 0:
			orphans++
		case 1:
			attached[candidate]++
		default:
			ambiguous++
		}
	}
	marked = make([]bool, len(steps))
	for i, k := range attached {
		switch k {
		case 0:
		case steps[i].n:
			marked[i] = true
		default:
			ambiguous += k
		}
	}
	return marked, ambiguous, orphans
}

// abs rend la valeur absolue d'un entier.
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// cleDeVictime est l'identité de la victime que nomme une action de score : le xuid, ou la place d'un bot (-1 sans
// place). Deux actions qui ont la même clé nomment la même victime.
type cleDeVictime struct {
	xuid  string
	place int
}

// cleDe rend la clé de victime d'une action.
func cleDe(a ScoreAction) cleDeVictime {
	k := cleDeVictime{xuid: a.Target, place: -1}
	if a.TargetPlace != nil {
		k.place = *a.TargetPlace
	}
	return k
}

// groupesEnAttente regroupe par image les éliminations pas encore réglées (frames croissantes) : chaque groupe est une
// image et les index de ses lignes, qui ne se distinguent que par leur victime.
func groupesEnAttente(frames []int, regle []bool) [][]int {
	var out [][]int
	for i, f := range frames {
		if regle[i] {
			continue
		}
		if n := len(out); n > 0 && frames[out[n-1][0]] == f {
			out[n-1] = append(out[n-1], i)
			continue
		}
		out = append(out, []int{i})
	}
	return out
}

// candidatesDansLaFenetre rend les index des actions pas encore utilisées à `window` images au plus de `frame`, dans
// l'ordre des actions.
func candidatesDansLaFenetre(actions []ScoreAction, used []bool, frame, window int) []int {
	var out []int
	for j, a := range actions {
		if !used[j] && abs(a.T-frame) <= window {
			out = append(out, j)
		}
	}
	return out
}

// appariementSur : l'appariement est sûr quand aucune candidate n'est réclamée par un autre groupe d'images, et que les
// candidates ne dépassent pas les lignes du groupe, sauf si elles nomment toutes la même victime. Sinon plusieurs
// victimes restent possibles pour ces lignes : le film ne dit pas laquelle.
func appariementSur(candidates []int, lignes int, reclame map[int]int, actions []ScoreAction) bool {
	for _, j := range candidates {
		if reclame[j] > 1 {
			return false
		}
	}
	if len(candidates) <= lignes {
		return true
	}
	for _, j := range candidates[1:] {
		if cleDe(actions[j]) != cleDe(actions[candidates[0]]) {
			return false
		}
	}
	return true
}

// victimsOf rend, pour chaque élimination (frames croissantes), l'index de l'action de score du joueur qui lui
// correspond, -1 sans action, et dit lesquelles ont plusieurs victimes possibles. Les actions sont triées par image.
// Fenêtres croissantes : d'abord la même image, puis l'image voisine pour ce qui reste ; une action ne sert qu'une fois.
//
// Une élimination n'est appariée que si son appariement est sûr ([appariementSur]). Quand plusieurs victimes restent
// possibles, la ligne n'est PAS appariée au plus proche : elle est rendue ambiguë (victime absente) et ses candidates
// sont écartées pour que nulle autre élimination ne les reçoive en silence. Des lignes de la même image sont
// interchangeables : elles reçoivent les candidates dans l'ordre.
func victimsOf(frames []int, actions []ScoreAction, maxWindow int) (victims []int, ambiguous []bool) {
	victims = make([]int, len(frames))
	for i := range victims {
		victims[i] = -1
	}
	ambiguous = make([]bool, len(frames))
	regle := make([]bool, len(frames))
	used := make([]bool, len(actions))
	for window := 0; window <= maxWindow; window++ {
		groupes := groupesEnAttente(frames, regle)
		candidates := make([][]int, len(groupes))
		reclame := map[int]int{}
		for g, lignes := range groupes {
			candidates[g] = candidatesDansLaFenetre(actions, used, frames[lignes[0]], window)
			for _, j := range candidates[g] {
				reclame[j]++
			}
		}
		for g, lignes := range groupes {
			c := candidates[g]
			switch {
			case len(c) == 0:
			case appariementSur(c, len(lignes), reclame, actions):
				for k := 0; k < len(lignes) && k < len(c); k++ {
					victims[lignes[k]], regle[lignes[k]], used[c[k]] = c[k], true, true
				}
			default:
				for _, i := range lignes {
					ambiguous[i], regle[i] = true, true
				}
				for _, j := range c {
					used[j] = true
				}
			}
		}
	}
	return victims, ambiguous
}

// boolPtr rend un pointeur vers une copie de v : chaque ligne porte le sien.
func boolPtr(v bool) *bool { return &v }

// markHead pose la marque de tir à la tête de l'élimination `e` selon le verdict de son pas, et la compte dans cov.
func markHead(e *Elimination, v headVerdict, cov *EliminationCoverage) {
	switch v {
	case headVerdictYes:
		e.Headshot = boolPtr(true)
		cov.HeadYes++
	case headVerdictNo:
		e.Headshot = boolPtr(false)
		cov.HeadNo++
	case headVerdictNoCounter:
		cov.HeadNoCounter++
	case headVerdictAmbiguous:
		cov.HeadAmbiguous++
	case headVerdictGuarded:
		cov.HeadGuarded++
	}
}

// poserLaVictime pose sur `e` la victime que l'action de score donne (un xuid, ou la place d'un bot) et la compte dans cov ;
// une action qui ne nomme personne ne pose rien.
func poserLaVictime(e *Elimination, a ScoreAction, cov *EliminationCoverage) {
	switch {
	case a.Target != "":
		e.Victim = a.Target
	case a.TargetPlace != nil:
		e.VictimPlace = a.TargetPlace
		cov.VictimPlaces++
	default:
		return
	}
	cov.WithVictim++
}

// eliminationsDeBots rend les éliminations des bots : une par action `killPersonalScore` posée à une place de bot. Le bot n'a
// ni courbe de tableau ni compteur de tirs à la tête ni médaille rattachée : `headshot` et `perfect` restent absents, et la
// ligne se compte parmi les inconnus faute de compteur.
func eliminationsDeBots(actions []ScoreAction, cov *EliminationCoverage) []Elimination {
	var out []Elimination
	for _, a := range actions {
		if a.Name != killScoreActionName || a.XUID != "" || a.Place == nil {
			continue
		}
		e := Elimination{T: a.T, Place: a.Place}
		poserLaVictime(&e, a, cov)
		cov.BotEliminations++
		cov.HeadNoCounter++
		out = append(out, e)
	}
	return out
}

// eliminationsOfPlayer rend les éliminations d'un joueur et compte dans cov ce qu'elles valent. Rend nil sans pas.
func eliminationsOfPlayer(p PlayerScore, actions []ScoreAction, medalFrames []int, in eliminationInput, cov *EliminationCoverage) []Elimination {
	steps := stepsOfTotal(p.Kills.Total)
	if len(steps) == 0 {
		cov.PerfectOrphans += len(medalFrames)
		return nil
	}
	var counter *ScoreSeries
	if s, ok := p.Stats[objectives.StatHeadshots]; ok {
		counter = &s
	}
	heads, orphanRises := headVerdictsOf(steps, counter, in.HeadshotsGuarded)
	cov.HeadOrphanRises += orphanRises
	perfect, ambiguous, orphans := perfectStepsOf(steps, medalFrames, windowFrames(perfectWindowMS, in.StepMS))
	cov.PerfectAmbiguous += ambiguous
	cov.PerfectOrphans += orphans

	var out []Elimination
	var frames []int
	for i, s := range steps {
		for range s.n {
			e := Elimination{T: s.t, XUID: p.XUID}
			markHead(&e, heads[i], cov)
			if perfect[i] {
				e.Perfect = boolPtr(true)
				cov.PerfectYes++
			}
			out = append(out, e)
			frames = append(frames, s.t)
		}
	}
	victimes, plusieurs := victimsOf(frames, actions, windowFrames(victimWindowMS, in.StepMS))
	for i, j := range victimes {
		switch {
		case plusieurs[i]:
			cov.VictimAmbiguous++
		case j >= 0:
			poserLaVictime(&out[i], actions[j], cov)
		}
	}
	return out
}

// buildEliminations assemble le calque des éliminations et sa couverture. Les lignes sont triées par image, tueur, victime.
func buildEliminations(in eliminationInput) ([]Elimination, EliminationCoverage) {
	var cov EliminationCoverage
	actionsBy := map[string][]ScoreAction{}
	for _, a := range in.Actions {
		if a.Name == killScoreActionName {
			actionsBy[a.XUID] = append(actionsBy[a.XUID], a)
		}
	}
	medalsBy := map[string][]int{}
	for _, m := range in.Medals {
		if m.Name == medalPerfectName {
			medalsBy[m.XUID] = append(medalsBy[m.XUID], m.T)
		}
	}
	var out []Elimination //nolint:prealloc // nil sans élimination : l'empreinte d'étape distingue nil d'une tranche vide, et la taille se connaît au pas près seulement
	for _, p := range in.Players {
		out = append(out, eliminationsOfPlayer(p, actionsBy[p.XUID], medalsBy[p.XUID], in, &cov)...)
		delete(medalsBy, p.XUID)
	}
	out = append(out, eliminationsDeBots(in.Actions, &cov)...)
	for _, frames := range medalsBy {
		cov.PerfectOrphans += len(frames)
	}
	cov.Eliminations = len(out)
	cov.HeadUnknown = cov.HeadNoCounter + cov.HeadAmbiguous + cov.HeadGuarded
	cov.PerfectUnknown = cov.Eliminations - cov.PerfectYes
	slices.SortStableFunc(out, func(a, b Elimination) int {
		return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.XUID, b.XUID), cmp.Compare(placeOuMoinsUn(a.Place), placeOuMoinsUn(b.Place)),
			cmp.Compare(a.Victim, b.Victim), cmp.Compare(placeOuMoinsUn(a.VictimPlace), placeOuMoinsUn(b.VictimPlace)))
	})
	return out, cov
}

// poserLesEliminations publie le calque : seulement quand le tableau des scores est publié, et APRÈS les médailles et les
// actions de score dont il dérive.
func (a *assemblage) poserLesEliminations() {
	tl := a.doc.ScoreTimeline
	if tl == nil {
		return
	}
	in := eliminationInput{Players: tl.Players, Actions: a.doc.ScoreActions, Medals: a.doc.Medals, StepMS: a.doc.FrameIntervalMS}
	if sc := a.doc.Coverage.Score; sc != nil {
		in.HeadshotsGuarded = sc.StatsGuarded[objectives.StatHeadshots] > 0
	}
	elims, cov := buildEliminations(in)
	a.doc.Eliminations = elims
	a.doc.Coverage.Eliminations = &cov
}
