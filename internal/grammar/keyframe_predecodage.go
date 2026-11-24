package grammar

// keyframe_predecodage.go — LE PRE-PASSAGE DES IMAGES-CLES : les resultats purs d'un payload, calcules
// sur plusieurs fils avant que le premier balayage ne les demande.
//
// QUAND IL PART : une seule fois par contexte, a la creation du memo des marches
// ([FilmContext.MarcheDImageCle]), et seulement quand [FilmContext.Fils] vaut plus de 1. A un fil, rien
// de ce fichier ne tourne : les resultats se calculent a la demande, dans le fil de l'appelant, comme
// avant. Le premier acces aux images-cles est le meme instant dans les deux cas : la preuve
// ([PreuveDImageCle]) est figee avant le pre-passage, et ses entrees (registre, grammaire du film,
// largeurs MPP du format) ne bougent plus ensuite.
//
// POURQUOI C'EST SUR : une tache est une fonction PURE d'un payload (et de la preuve, en lecture
// seule) ; elle ne touche ni le contexte, ni un memo, ni une variable de paquet. Elle ne RENVOIE que
// la fermeture qui range son resultat. Les fermetures s'executent apres la jointure, depuis le fil
// appelant, dans l'ordre des (payload, tache) : le contexte n'est ecrit que par le fil qui le
// possede, et aucun verrou n'est ajoute a un chemin de lecture. Le resultat ne depend ni du nombre
// de fils ni de l'ordonnancement.
//
// COMMENT AJOUTER UNE TACHE : ecrire une fonction de signature [tacheDePredecodage] (calcul pur du
// payload, rangement rendu en fermeture) et l'ajouter, UNE ligne, a [tachesDePredecodage].

// entreesDuPredecodage est tout ce qu'une tache a le droit de lire en dehors du payload : des valeurs
// figees avant le depart, jamais le contexte.
type entreesDuPredecodage struct {
	// preuve : la preuve des ancres de ce film (nil : marche sans preuve), lue sans etre ecrite.
	preuve *PreuveDImageCle
}

// tacheDePredecodage calcule, pour UN payload non vide, un resultat pur ; elle rend la fermeture qui
// le range dans le contexte (jamais nil). Elle s'execute sur un fil quelconque et ne touche rien d'autre
// que ses arguments ; la fermeture rendue, elle, ne s'execute que sur le fil appelant, apres la jointure.
type tacheDePredecodage func(e entreesDuPredecodage, pay []byte) (ranger func(c *FilmContext))

// tachesDePredecodage liste les taches executees pour chaque payload. Ajouter un resultat pur d'une
// image-cle = une ligne ici.
func tachesDePredecodage() []tacheDePredecodage {
	return []tacheDePredecodage{
		tacheDesAncres,
		tacheDesDatums,
		tacheDesEntetesExacts,
	}
}

// tacheDesDatums calcule la table des datums du payload ([TableDeDatums]) ; son rangement la pose dans
// le memo des tables ([FilmContext.tableDeDatumsDe]) si le payload n'y est pas deja.
func tacheDesDatums(_ entreesDuPredecodage, pay []byte) func(c *FilmContext) {
	table, ambigus := TableDeDatums(pay)
	return func(c *FilmContext) { c.poserTableDeDatumsSiAbsente(pay, table, ambigus) }
}

// tacheDesEntetesExacts calcule les slots d'en-tete exact de joueur gere du payload
// ([slotsDEntetesExacts]) ; son rangement les pose dans le memo des en-tetes si le payload n'y est pas deja.
func tacheDesEntetesExacts(_ entreesDuPredecodage, pay []byte) func(c *FilmContext) {
	slots := slotsDEntetesExacts(pay)
	return func(c *FilmContext) {
		cle := clePayload{debut: &pay[0], n: len(pay)}
		if _, ok := c.recup.entetes[cle]; ok {
			return
		}
		if c.recup.entetes == nil {
			c.recup.entetes = map[clePayload]map[int]bool{}
		}
		c.recup.entetes[cle] = slots
	}
}

// tacheDesAncres calcule la marche d ancres du payload sous la preuve du film ([MarcheDImageCle.Marcher] sans
// memo : une fonction pure des octets et de la preuve, que le fil du pre-passage ne fait que lire) ; son
// rangement la pose dans le memo des marches si le payload n y est pas deja.
func tacheDesAncres(e entreesDuPredecodage, pay []byte) func(c *FilmContext) {
	mp := MarcheDImageCle{preuve: e.preuve}.Marcher(pay)
	return func(c *FilmContext) { c.marches.poserSiAbsent(pay, mp) }
}

// predecoderLesImagesCles execute [tachesDePredecodage] sur tous les payloads d'image-cle, sur
// [FilmContext.Fils] fils, puis range les resultats depuis le fil appelant. Sans effet a un fil ou
// moins : les resultats se calculent alors a la demande. Le memo des marches doit exister.
func (c *FilmContext) predecoderLesImagesCles(preuve *PreuveDImageCle) {
	if c.Fils() <= 1 || c.marches == nil {
		return
	}
	payloads := c.payloadsDesImagesCles()
	taches := tachesDePredecodage()
	if len(payloads) == 0 || len(taches) == 0 {
		return
	}
	entrees := entreesDuPredecodage{preuve: preuve}
	rangements := make([]func(c *FilmContext), len(payloads)*len(taches))
	executerEnParallele(c.suivi, c.Fils(), len(rangements), func(i int) {
		rangements[i] = taches[i%len(taches)](entrees, payloads[i/len(taches)])
	})
	for _, ranger := range rangements {
		if ranger != nil {
			ranger(c)
		}
	}
}
