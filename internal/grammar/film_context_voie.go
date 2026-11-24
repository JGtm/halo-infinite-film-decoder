package grammar

// film_context_voie.go — LE CONTEXTE DE VOIE : de quoi faire tourner plusieurs balayages d'apres la
// marche en meme temps sur un meme film, sans que l'un ecrive ce que l'autre lit.
//
// # LE PRINCIPE
//
// Apres la marche des trames, un contexte porte des memos que les balayages LISENT (chunks, bande de
// slots, decoupage d'i0, registre, profil, grammaire du film, vies, lectures de la marche, tables des
// datums, ...) et un petit etat qu'ils ECRIVENT (profil de balayage, observation, rapport des replis,
// diagnostics, memos des pistes et des creations du monde). Une voie partage les premiers en lecture
// seule, possede les seconds en prive, et les verse au parent a la jointure :
//
//	Chauffer   le fil qui possede le contexte calcule d'avance ce que les voies liraient ;
//	EstChaud   dit que tout est pret (sinon : le chemin sans voie, comme avant) ;
//	Deriver    rend un contexte de voie ;
//	Fusionner  verse ce que la voie a accumule, depuis le fil qui possede le parent.
//
// Un contexte, parent ou voie, ne sert qu'UN fil a la fois. Aucune de ces fonctions ne lance de
// goroutine ni ne prend de verrou : l'appelant joint ses voies (cf. executerEnParallele) avant de
// fusionner.
//
// # LA LISTE FERMEE, ET CE QUI LA TIENT
//
// Chaque champ de [FilmContext] est classe : immuable, chauffe (partage en lecture), prive a la voie, ou
// ecarte (nul dans la voie). `TestChaufferRendTousLesMemosLus` echoue sur un champ, ou un sous-champ de
// `recup`, `vies`, `duFilm` ou `replis`, qui n'est pas classe : ajouter un memo paresseux au contexte
// oblige a decider de son sort ici.
//
// # CE QUE LA VOIE NE RECOIT PAS
//
//   - les crochets du parent : son observation est NEUVE. Un instrument qui pose des crochets sur
//     l'observation du parent avant les voies ne les verrait pas dans une voie ;
//   - la marche gardee pour la cuisson ([memoDesRecuperations.marche]) : elle se consomme (`rendue`,
//     `prise`), elle reste au parent ;
//   - les memos des pistes et des creations du monde : ils se mutent apres leur creation
//     (`finales`, `fondues`) ; la voie qui les demande en est l'unique proprietaire. Le parent peut les
//     avoir demandes avant de deriver : la voie les releve alors de nouveau, pour lire les memes valeurs,
//     et [FilmContext.Fusionner] n'ajoute pas le repli « apres la marche » que le parent a deja compte ;
//   - plus d'un fil : une voie est une file serielle, [FilmContext.Fils] y vaut 1.

import (
	"errors"
	"maps"
)

// etatDeVoie est ce qu'un contexte de voie sait de sa derivation.
type etatDeVoie struct {
	// parent : le contexte dont la voie est derivee, le seul qui peut la fusionner.
	parent *FilmContext
	// fusionnee : la voie a deja ete versee ; une seconde fusion compterait deux fois ses replis.
	fusionnee bool
}

// laMarcheEstFaite dit si la marche des trames a rendu ce que les balayages suivants relisent : les
// lectures bipedes, les objets du monde et l'equipement porte. Faux, un balayage relancerait une
// marche de repli, ce qu'aucune voie ne doit faire.
func (c *FilmContext) laMarcheEstFaite() bool {
	return c.recup.lectures != nil && c.recup.objets != nil && c.recup.equipementPorte != nil
}

// Chauffer calcule, depuis le fil qui possede le contexte, les memos paresseux que des balayages
// simultanes liraient : l'observation, les chunks, la bande de slots, le decoupage d'i0, le registre,
// le profil, la grammaire du film, la marche d'ancres des images-cles (et, a plusieurs fils, son
// pre-passage), la table des datums de chaque image-cle ; puis, si la marche des trames est faite, les
// generations vivantes (donc les creations de bipede et les vies des images-cles) et l'ancrage bipede.
//
// Chauffer ne lance JAMAIS la marche des trames : sur un contexte dont la marche n'a pas tourne, les
// memos qu'elle remplit restent vides et [FilmContext.EstChaud] rend faux. Idempotent. Rend les erreurs du
// decoupage d'i0 et du registre : elles sont rangees dans le memo, chaque lecteur les retrouve, et la
// chauffe continue (un film sans registre ou sans decoupage est chaud, ses lecteurs rendent l'erreur). Sans effet sur
// un contexte nil.
func (c *FilmContext) Chauffer() error {
	if c == nil {
		return nil
	}
	c.Observation()
	c.ChunkNumbers()
	c.BipedSlots()
	_, errLay := c.I0Layout()
	_, errReg := c.Registry()
	c.Profile()
	c.controleDeCorruptionDuFilm()
	c.MarcheDImageCle()
	for _, pay := range c.payloadsDesImagesCles() {
		c.tableDeDatumsDe(pay)
	}
	if c.laMarcheEstFaite() {
		// Sans decoupage d'i0 ou sans archetype bipede, la cloture de la marche ne range aucun ancrage :
		// un lecteur le creerait. Borne sur les trames de la marche, comme la cloture le fait.
		c.ancresBipedes(c.recup.lectures.trames)
		// Seul le rangement du memo compte ; l instant passe ne juge aucun record.
		c.GenerationsVivantesA(0)
	}
	return errors.Join(errLay, errReg)
}

// EstChaud dit si aucun balayage de lecture ne peut plus ecrire dans un memo partage : tous les memos
// que [FilmContext.Chauffer] prepare sont prets ET la marche des trames est faite. Faux, les balayages
// tournent l'un apres l'autre sur ce contexte, comme avant. Calcule a chaque appel : il ne se perime pas.
func (c *FilmContext) EstChaud() bool {
	return c != nil && c.memosDeBaseChauds() && c.memosDeLaMarcheChauds()
}

// memosDeBaseChauds dit si les memos qui ne dependent que du film sont prets.
func (c *FilmContext) memosDeBaseChauds() bool {
	d := &c.duFilm
	return c.obs != nil && c.chunksLus && c.slotsLus && (c.impose != nil || c.layLu) && c.regLu &&
		c.profLu && d.derive && (d.lue || d.compte) && c.marches != nil && c.tablesDesImagesClesChaudes()
}

// memosDeLaMarcheChauds dit si les memos que la marche des trames remplit, et ce qui s'en derive, sont prets.
func (c *FilmContext) memosDeLaMarcheChauds() bool {
	v := &c.vies
	return c.laMarcheEstFaite() && c.recup.ancres != nil && v.creationsLues && v.vivantes != nil &&
		v.imagesClesRelevees && !v.enMarche
}

// tablesDesImagesClesChaudes dit si la table des datums de chaque image-cle est rangee dans le memo.
func (c *FilmContext) tablesDesImagesClesChaudes() bool {
	for _, pay := range c.payloadsDesImagesCles() {
		if m, ok := c.recup.tables[&pay[0]]; !ok || m.bits != len(pay) {
			return false
		}
	}
	return true
}

// Deriver rend un contexte de voie : il partage en lecture le film et les memos chauds de `c`, et
// possede en PRIVE ce qu'un balayage ecrit (cf. l'en-tete). A appeler depuis le fil qui possede `c`,
// apres [FilmContext.Chauffer], et seulement quand [FilmContext.EstChaud] : sur un contexte froid la voie
// recalcule en prive ce qui manque, sans course, mais refait le travail.
//
//   - partage (lecture) : le film, les chunks, la bande de slots, le decoupage d'i0, le registre, le
//     profil, les memos des marches d'ancres et des en-tetes exacts, les lectures de la marche
//     (bipedes, objets du monde, equipement porte), l'ancrage, les vies et les generations vivantes ;
//   - copie privee : le profil de balayage (dont les largeurs MPP que [FilmContext.PoserMPP] pose puis
//     restaure), la grammaire du film, et le memo des tables de datums (copie superficielle : une table
//     absente se range chez la voie, jamais chez le parent ni chez une voie soeur) ;
//   - neuf : l'observation, le rapport des replis (zero, les drapeaux « une fois par film » du parent
//     repris), les diagnostics, les pistes et les creations du monde.
func (c *FilmContext) Deriver() *FilmContext {
	return &FilmContext{
		film: c.film, chunks: c.chunks, chunksLus: c.chunksLus, slots: c.slots, slotsLus: c.slotsLus,
		impose: c.impose, lay: c.lay, layErr: c.layErr, layLu: c.layLu,
		reg: c.reg, regErr: c.regErr, regLu: c.regLu, prof: c.prof, profLu: c.profLu,
		bal: c.bal, duFilm: c.duFilm, vies: c.vies, marches: c.marches,
		recup: memoDesRecuperations{ancres: c.recup.ancres, objets: c.recup.objets, lectures: c.recup.lectures,
			equipementPorte: c.recup.equipementPorte, tables: maps.Clone(c.recup.tables), entetes: c.recup.entetes},
		obs:    NouvelleObservation(),
		replis: replisDuContexte{i0Note: c.replis.i0Note, regNote: c.replis.regNote},
		fils:   1, suivi: c.suivi,
		voie: &etatDeVoie{parent: c},
	}
}

// Fusionner verse dans `c` ce que la voie `voie` (derivee de `c` par [FilmContext.Deriver]) a
// accumule, depuis le fil qui possede `c` et APRES la jointure de la voie :
//
//   - les replis : sommes champ a champ ([ComptesDesReplis.Plus], commutative). Les trois replis
//     comptes « une fois par film » (decoupage d'i0 par defaut, registre inconnu, controle de corruption
//     non declare) ne s'ajoutent que si `c` ne les a pas deja comptes : le premier qui compte gagne, les
//     autres sont ecartes, et les drapeaux se combinent par OU ;
//   - les diagnostics, a la suite de ceux de `c` : l'ordre des fusions est celui des diagnostics ;
//   - les pistes et les creations du monde que `c` ne porte pas encore ; ce que la voie a compte au repli
//     pour un rendu que `c` portait deja n'est pas ajoute : le compte est celui d'un fil, que `c` ait ou
//     non demande ces memos avant de deriver.
//
// Rien d'autre ne remonte : le profil de balayage de la voie et son observation (ses compteurs ne
// sont lus par aucune production) sont abandonnes. Une voie ne se fusionne qu'une fois, dans son
// parent ; autrement c'est une erreur de programmation et Fusionner panique.
func (c *FilmContext) Fusionner(voie *FilmContext) {
	if c == nil || voie == nil || voie.voie == nil || voie.voie.parent != c || voie.voie.fusionnee {
		panic("grammar: Fusionner veut une voie de ce contexte, jamais fusionnee")
	}
	voie.voie.fusionnee = true
	d := voie.replis.comptes
	pistes, creations := c.reprendreLesMemosDuMonde(voie)
	d.PistesDuMondeApresLaMarche -= pistes
	d.CreationsDuMondeApresLaMarche -= creations
	if c.replis.i0Note {
		d.I0PorteEtRegionParDefaut = 0
	}
	if c.replis.regNote {
		d.RegistreInconnu = 0
	}
	if c.duFilm.compte {
		d.ControleDeCorruptionNonDeclare = 0
	}
	c.replis.comptes = c.replis.comptes.Plus(d)
	c.replis.i0Note = c.replis.i0Note || voie.replis.i0Note
	c.replis.regNote = c.replis.regNote || voie.replis.regNote
	c.duFilm.compte = c.duFilm.compte || voie.duFilm.compte
	voie.diag.Verser(&c.diag)
}

// reprendreLesMemosDuMonde range dans `c` les pistes et les creations du monde que la voie a relevees et
// que `c` ne porte pas deja (meme cle), et rend ce que la voie a compte au repli pour des rendus que `c`
// portait deja : le contexte parent les avait comptes, un seul fil ne les aurait pas comptes de nouveau.
// La voie en est l'unique proprietaire : on prend ses pointeurs. Un rendu que `c` porte sans l'avoir fait
// (entree relevee, rendu d'un archetype ou des creations absent) est repris de la voie, avec son compte.
func (c *FilmContext) reprendreLesMemosDuMonde(voie *FilmContext) (dejaComptePistes, dejaCompteCreations int) {
	for _, p := range voie.recup.pistes {
		mien, deja := c.recup.pistesDe(p.wr, p.lg, p.slots)
		if !deja {
			c.recup.pistes = append(c.recup.pistes, p)
			continue
		}
		for ti, f := range p.finales {
			if _, fait := mien.finales[ti]; fait {
				dejaComptePistes += p.recuperes[ti]
			} else if mien.finales != nil && mien.recuperes != nil {
				mien.finales[ti], mien.recuperes[ti] = f, p.recuperes[ti]
			}
		}
	}
	for _, r := range voie.recup.creations {
		mien, deja := c.recup.creationsDe(r.cle)
		switch {
		case !deja:
			c.recup.creations = append(c.recup.creations, r)
		case mien.fondues:
			dejaCompteCreations += r.repli
		case r.fondues:
			mien.fondues, mien.finales, mien.statsFinales, mien.repli = true, r.finales, r.statsFinales, r.repli
		}
	}
	return dejaComptePistes, dejaCompteCreations
}
