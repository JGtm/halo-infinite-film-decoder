package replay

// projectiles_tirs.go — LE LIEN TIR -> PROJECTILE (`Projectile.Shot`).
//
// La grammaire du film ne porte aucune référence du projectile dans l'évènement de tir, ni de
// l'arme dans le message de détonation : le lien exact n'est pas décodable. Il se POSE donc par
// coïncidence, la même famille de règle que l'appariement lancer -> naissance des grenades
// ([birthForThrow]) : un projectile né sur la frame d'un tir, à sa bouche.
//
// La règle est conservatrice : en cas de doute, aucun lien. Un lien faux ferait exploser le
// projectile d'une autre arme ; l'absence de lien ne coûte qu'une explosion non dessinée, et
// elle est comptée ([ProjectileCoverage.ShotAmbiguous]).

const (
	// projShotWindowMS est l'écart maximal, en millisecondes, entre la naissance du projectile et
	// l'instant du tir, jugé sur les instants EXACTS des deux enregistrements : la décision ne
	// dépend pas du pas de la grille.
	projShotWindowMS = 100
	// projShotMaxDistM est la distance maximale, en mètres, entre le point du tir (la bouche, ou
	// le véhicule pour un tir embarqué) et le premier point publié du projectile.
	projShotMaxDistM = 3.0
)

// lierLesProjectilesAuxTirs pose les liens sur le document en construction et en publie le
// compte dans la couverture des projectiles (absente, donc rien à compter, sans piste).
func (a *assemblage) lierLesProjectilesAuxTirs() {
	linked, ambiguous := linkShotsToProjectiles(a.doc.Projectiles, a.doc.Shots, a.doc.Grenades, a.interval)
	if a.projCov != nil {
		a.projCov.ShotLinked, a.projCov.ShotAmbiguous = linked, ambiguous
	}
}

// tirCandidat résume un tir candidat : son index publié et sa distance au départ du projectile.
type tirCandidat struct {
	index int
	dist  float64
}

// linkShotsToProjectiles pose `Projectile.Shot` sur les projectiles NON-GRENADE dont un tir
// publié rend la naissance. Il rend le nombre de liens posés et le nombre de projectiles
// laissés sans lien parce qu'AMBIGUS.
//
// Les candidats d'un projectile sont les tirs dont l'instant est à `projShotWindowMS` de sa
// naissance — comparés en MICROSECONDES : la fenêtre valait 99 ms ou 200 ms de portée réelle selon que les deux
// instants tombaient dans la même frame ou non, donc selon le pas de la grille ; `intervalMS` ne sert plus qu'à
// ranger les tirs par frame pour ne parcourir que ceux qui peuvent tomber dans la fenêtre — et dont le point est à
// `projShotMaxDistM` de son premier point. Le lien est publié quand tous les candidats sont de la
// MÊME arme et du MÊME tireur (un seul tir, ou une salve) : le plus proche l'emporte, à égalité le plus petit index — l'ordre de `shots` est
// déterministe, donc le lien aussi. Deux armes ou deux tireurs candidats : ambigu, aucun lien.
// Un projectile sans candidat n'est ni lié ni ambigu (balle, trace, tir non publié).
//
// Les projectiles que référence un lancer de grenade ([Grenade.Proj]) sont exclus : leur lien
// est celui du lancer.
func linkShotsToProjectiles(projs []Projectile, shots []Shot, grenades []Grenade,
	intervalMS int) (linked, ambiguous int) {
	if len(projs) == 0 || len(shots) == 0 {
		return 0, 0
	}
	// `window` ne sert qu'a ranger les tirs par frame : un tir et une naissance a au plus `projShotWindowMS` l'un de
	// l'autre sont separes d'au plus cette duree arrondie au-dessus, plus une frame (l'arrondi des deux instants a
	// la grille). Un pas absent est le pas historique, comme pour `framesOfMS`.
	if intervalMS <= 0 {
		intervalMS = HistoricalFrameIntervalMS
	}
	window := (projShotWindowMS+intervalMS-1)/intervalMS + 1
	byFrame := make(map[int][]int, len(shots))
	for i, s := range shots {
		byFrame[s.T] = append(byFrame[s.T], i)
	}
	fromGrenade := make(map[int]bool, len(grenades))
	for _, g := range grenades {
		if g.Proj != nil {
			fromGrenade[*g.Proj] = true
		}
	}
	for i := range projs {
		if fromGrenade[i] || len(projs[i].P) == 0 {
			continue
		}
		cands := candidateShots(projs[i], shots, byFrame, window)
		if len(cands) == 0 {
			continue
		}
		best, ok := pickShot(cands, shots)
		if !ok {
			ambiguous++
			continue
		}
		projs[i].Shot = &best
		linked++
	}
	return linked, ambiguous
}

// candidateShots rend les tirs candidats de `p`, par frame croissante puis index croissant.
func candidateShots(p Projectile, shots []Shot, byFrame map[int][]int, window int) []tirCandidat {
	var out []tirCandidat
	for f := p.T0 - window; f <= p.T0+window; f++ {
		for _, idx := range byFrame[f] {
			if !dansLaFenetreDuTir(shots[idx].tUS, p.t0US) {
				continue
			}
			d := planDist(shots[idx].X, shots[idx].Y, p.P[0][1], p.P[0][2])
			if d <= projShotMaxDistM {
				out = append(out, tirCandidat{index: idx, dist: d})
			}
		}
	}
	return out
}

// dansLaFenetreDuTir dit que le tir et la naissance du projectile (microsecondes du film) sont a au plus
// [projShotWindowMS] l'un de l'autre.
func dansLaFenetreDuTir(tirUS, naissanceUS uint64) bool {
	ecart := tirUS - naissanceUS
	if naissanceUS > tirUS {
		ecart = naissanceUS - tirUS
	}
	return ecart <= projShotWindowMS*1000
}

// pickShot choisit le tir le plus proche parmi des candidats de même arme et même tireur ;
// faux dès que deux armes ou deux tireurs se disputent le projectile.
func pickShot(cands []tirCandidat, shots []Shot) (int, bool) {
	first := shots[cands[0].index]
	best := cands[0]
	for _, c := range cands[1:] {
		s := shots[c.index]
		if s.Weapon != first.Weapon || s.Slot != first.Slot {
			return 0, false
		}
		if c.dist < best.dist || (c.dist == best.dist && c.index < best.index) {
			best = c
		}
	}
	return best.index, true
}
