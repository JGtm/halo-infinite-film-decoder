package grammar

// film_context_fils.go — LE NOMBRE DE FILS PERMIS A UN DECODAGE.
//
// Le contexte porte le nombre de fils que les etapes parallelisables ont le droit d'employer. Il
// vaut 1 a la construction : un contexte que l'hote ne regle pas decode comme avant, dans le fil
// de l'appelant. Le plafond (nombre de coeurs, borne de la brique) n'est pas decide ici : l'hote
// le normalise avant d'appeler [FilmContext.PoserFils].

// PoserFils regle le nombre de fils permis ; une valeur inferieure a 1 se range a 1. A poser avant
// le premier balayage du film, depuis le fil qui possede le contexte.
func (c *FilmContext) PoserFils(n int) {
	if n < 1 {
		n = 1
	}
	c.fils = n
}

// Fils rend le nombre de fils permis, au moins 1.
func (c *FilmContext) Fils() int {
	if c.fils < 1 {
		return 1
	}
	return c.fils
}

// SuivreLeCalcul signale au suivi du contexte, s'il y en a un, qu'un fil de calcul commence (delta = 1)
// ou finit (-1). Chaque point de lancement du decodeur (l'assistant [executerEnParallele], la tache de
// fond et les voies du balayage de `replay`) l'appelle : +1 quand un fil commence a calculer, -1 quand il
// s'arrete ; l'appelant d'une jointure se compte -1 pendant qu'il attend. Sans suivi (le cas de la
// production), le cout est un test de nullite ; le suivi appartient au contexte, donc a UNE cuisson, et
// aucun etat de paquet n'y concourt.
func (c *FilmContext) SuivreLeCalcul(delta int) {
	if c != nil && c.suivi != nil {
		c.suivi(delta)
	}
}

// PoserLeSuiviDuCalcul installe `f` comme suivi du contexte (nil le retire). Reserve aux tests, qui
// comptent les fils : a poser avant la cuisson qu'il observe, depuis le fil qui possede le contexte. Les
// voies derivees ([FilmContext.Deriver]) ne le recoivent pas : elles ne lancent aucun fil.
func (c *FilmContext) PoserLeSuiviDuCalcul(f func(delta int)) {
	c.suivi = f
}
