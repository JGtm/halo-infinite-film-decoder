package replay

// document_score_actions_tableau.go — LA SOMME DES POINTS PUBLIES CONTRE LE SCORE DU TABLEAU DU FILM.
//
// Le tableau des scores du film porte, pour chaque joueur qu'il nomme, son score personnel
// (`scoreTimeline.players[].score`). La somme des `scoreActions[].points` d'un joueur est ce que la table des
// réponses lui donne : quand les deux diffèrent, une action manque, est de trop ou vaut autre chose que dans le
// jeu (points de bombe hors de l'arbre). Un tableau AU-DESSUS des points positifs publiés (`scoreboardMissing`) accuse
// la lecture et rend `complete` faux ; un tableau en dessous (`scoreboardBehind`) n'accuse pas la lecture : le jeu
// retire bien une pénalité du score, mais la courbe publiée du tableau (la plus longue suite non décroissante des
// relevés) ne recule pas, et quand la pénalité est le dernier mouvement d'un joueur son total publié garde la
// valeur d'avant.

// scoreFinalDuTableau rend, par xuid, le dernier total du score personnel du tableau du film. Un joueur nommé par
// le tableau sans total y vaut zéro. Nil quand le film ne publie aucun joueur du tableau : rien ne se compare.
func scoreFinalDuTableau(tl *ScoreTimeline) map[string]int {
	if tl == nil || len(tl.Players) == 0 {
		return nil
	}
	out := make(map[string]int, len(tl.Players))
	for _, p := range tl.Players {
		total := 0
		if n := len(p.Score.Total); n > 0 {
			total = p.Score.Total[n-1].V
		}
		out[p.XUID] = total
	}
	return out
}

// comparerAuTableau pose `scoreboardCompared`, `scoreboardMissing` et `scoreboardBehind`. La courbe publiée du tableau ne
// recule jamais, donc ses relevés ne dépassent pas la somme cumulée des points positifs : la somme comparée est celle
// des actions à points POSITIFS. Parmi
// les joueurs qui ont une action publiée ou une ligne au tableau, `missing` est celui dont le score du tableau dépasse
// cette somme (des points que le calque ne publie pas), `behind` celui chez qui il est en dessous (le tableau est en
// retard sur les points publiés). Sans tableau, rien n'est comparé et les trois comptes restent à zéro.
func comparerAuTableau(actions []ScoreAction, tableau map[string]int, cov *ScoreActionCoverage) {
	if tableau == nil {
		return
	}
	positifs := map[string]int{}
	for _, a := range actions {
		if a.XUID == "" { // un bot : le tableau ne le nomme pas, rien à comparer
			continue
		}
		positifs[a.XUID] += max(a.Points, 0)
	}
	comparer := func(xuid string) {
		cov.ScoreboardCompared++
		switch score := tableau[xuid]; {
		case score > positifs[xuid]:
			cov.ScoreboardMissing++
		case score < positifs[xuid]:
			cov.ScoreboardBehind++
		}
	}
	for xuid := range positifs {
		comparer(xuid)
	}
	for xuid := range tableau {
		if _, ok := positifs[xuid]; !ok {
			comparer(xuid)
		}
	}
}
