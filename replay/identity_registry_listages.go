package replay

// identity_registry_listages.go — QUI TIENT UN SIEGE A UN INSTANT, LU DANS LA POPULATION DE LA SESSION ET DANS
// LE JOUEUR AUX COMMANDES.
//
// L index de participant d un corps (celui que son record de creation ecrit, celui que la tete d `unit-control`
// ecrit) n est pas une identite : un arrivant peut reprendre le siege d un partant, et un bot peut tenir le siege
// qu un humain a quitte. Les plages de listage des paquets de type 8 ([types.RosterListing]) donnent le joueur
// qui tient un siege A L INSTANT du corps : la table d index, elle, ne garde qu un occupant par index. Deux
// lectures en decoulent, qui se COMPTENT :
//
//	ParListage    l humain liste sous cet index a la creation du corps, autre que celui de la table, le nomme ;
//	HumainRetire  le xuid que la table donne a l index n est plus celui du corps : aucun humain n est liste sous cet
//	              index a cet instant, ou celui qui l est n a pas de xuid dans la table. Il n est pas pose (le bot
//	              declare au siege le nomme, quand il y en a un).
//
// Un film dont les paquets ne portent pas de siege (formats anterieurs a la porte de siege), ou dont la lecture
// n a pas eu lieu, n a pas de plages valides : la lecture se tait et la table d index nomme comme avant.
//
// LA TETE D `unit-control` EST LE COMPLEMENT DU RECORD DE CREATION : un corps dont aucun record ne ouvre le slot
// est nomme par le joueur que la premiere lecture du slot, depuis la fin de sa vie precedente, met aux commandes
// (`ParControle`).

import "github.com/JGtm/halo-infinite-film-decoder/types"

// siegesListes range les plages de listage par siege.
type siegesListes struct {
	plages []types.RosterListing
	// valide : la lecture a eu lieu et chaque plage porte un siege.
	valide bool
}

// nouveauxSiegesListes construit la lecture des sieges depuis les lectures de presence du film.
func nouveauxSiegesListes(p LecturesDePresence) siegesListes {
	if !p.ListingStats.Scanned || len(p.Listings) == 0 {
		return siegesListes{}
	}
	for _, l := range p.Listings {
		if l.Slot < 0 {
			return siegesListes{}
		}
	}
	return siegesListes{plages: p.Listings, valide: true}
}

// humainA rend le xuid de l humain liste sous `siege` a l instant `us`, et faux quand aucune plage ne le porte.
func (s siegesListes) humainA(siege int, us uint64) (uint64, bool) {
	for _, l := range s.plages {
		if plageTient(l, siege, us) {
			return l.XUID, true
		}
	}
	return 0, false
}

// imageCleAvantLeListageUS : l image-cle du debut du film precede le premier paquet de type 8 de quelques
// centaines de millisecondes ; une entite lue a cette image-cle est celle de l humain liste juste apres.
const imageCleAvantLeListageUS = 1_000_000

// plageTient dit que la plage liste son humain sous `siege` a l instant `us`, a l ecart de
// [imageCleAvantLeListageUS] pres a l ouverture de la plage : un corps ou une entite lus a l image-cle du debut
// du film, ou celle d un arrivant, precedent de quelques centaines de millisecondes le paquet qui le liste.
func plageTient(l types.RosterListing, siege int, us uint64) bool {
	return l.Slot == siege && l.FromUS <= us+imageCleAvantLeListageUS && (l.ToUS == 0 || us < l.ToUS)
}

// tenuPar dit qu une des plages d un humain le liste sous `siege` a l instant `us`.
func tenuPar(plages []types.RosterListing, siege int, us uint64) bool {
	for _, l := range plages {
		if plageTient(l, siege, us) {
			return true
		}
	}
	return false
}

// aucunHumain dit qu aucune plage ne tient `siege` pendant [de, a] (microsecondes), avec la meme tolerance
// d ouverture que [plageTient].
func (s siegesListes) aucunHumain(siege int, de, a uint64) bool {
	for _, l := range s.plages {
		if l.Slot == siege && l.FromUS <= a+imageCleAvantLeListageUS && (l.ToUS == 0 || de < l.ToUS) {
			return false
		}
	}
	return true
}

// controleDeLaVie rend la premiere lecture du slot de la vie `i` qui nomme un joueur, depuis la fin de la vie
// precedente du meme slot (`depuis`) jusqu a une seconde apres la fin de la vie. Faux sans lecture.
func controleDeLaVie(lus []types.UnitControlRead, depuis int64, l lifeSpan) (dateDeCreation, bool) {
	for _, c := range lus {
		t := int64(c.TimestampUS)
		if t < depuis || t > l.to+controleApresLaFinUS {
			continue
		}
		if c.Controller >= 0 {
			return dateDeCreation{tUS: t, index: uint32(c.Controller)}, true //nolint:gosec // R(5), 0..31
		}
	}
	return dateDeCreation{}, false
}

// controleApresLaFinUS : la fenetre d une vie prend les lectures d une seconde apres sa derniere position.
const controleApresLaFinUS = 1_000_000

// nommerParLeControle tente de nommer la vie `i`, que aucun record de creation n ouvre, par la tete d
// `unit-control` de son slot. `precedente` est la fin de la vie precedente du slot (0 sans vie avant). La cause
// d origine est gardee quand la lecture ne nomme pas.
func (r *creationReport) nommerParLeControle(lives []lifeSpan, i int, precedente int64,
	lus []types.UnitControlRead, t resolutionDIndex) {
	d, ok := controleDeLaVie(lus, precedente, lives[i])
	if !ok {
		return
	}
	cause, avait := r.causes[i]
	delete(r.causes, i)
	compteurs := r.compteurs()
	_, aLuIndex := r.indexLu[i]
	if !r.poser(lives, i, d, t) {
		// UNE TENTATIVE QUI NE NOMME PAS NE COMPTE RIEN : le rapport ne porte que des decisions appliquees.
		r.restaurer(compteurs)
		if !aLuIndex {
			delete(r.indexLu, i)
		}
		if avait {
			r.causes[i] = cause
		}
		return
	}
	lives[i].nomPar = NomParCreation
	r.Direct++
	r.ParControle++
}

// viePrecedente rend la fin de la vie qui precede la vie d indice `k` dans le groupe `vies` d un slot (0 pour la
// premiere).
func viePrecedente(lives []lifeSpan, vies []int, k int) int64 {
	if k == 0 {
		return 0
	}
	return lives[vies[k-1]].to
}

// compteursDeLecture est la photographie des compteurs de decisions d un rapport de lecture directe.
type compteursDeLecture struct{ indexBot, humainRetire, parListage, parEntite, parDeclaration int }

func (r *creationReport) compteurs() compteursDeLecture {
	return compteursDeLecture{r.IndexBot, r.HumainRetire, r.ParListage, r.ParEntite, r.ParDeclaration}
}

func (r *creationReport) restaurer(c compteursDeLecture) {
	r.IndexBot, r.HumainRetire, r.ParListage, r.ParEntite, r.ParDeclaration = c.indexBot, c.humainRetire,
		c.parListage, c.parEntite, c.parDeclaration
}
