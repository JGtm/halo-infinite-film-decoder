package replay

// film_scan_voie_sieges.go — LA VOIE DES SIEGES ET DES OCCUPANTS : LES EQUIPES, PUIS LE TABLEAU DES SCORES.
//
// Elle lit, sur son contexte derive, la seule partie de la lecture des sieges qui passe par le contexte :
// l equipe de chaque index et les occupants par entite ([grammar.ScanPlayerTeams]), puis les entrees du
// tableau des scores ([grammar.ScanStatborgEntries]) — `lireLesOccupants`, la lecture que la chaine en serie
// et le document interne du sync emploient aussi. Chacune releve un corps d image-cle et pose les largeurs
// MPP du format le temps de sa boucle, d ou son contexte a elle.
//
// LA TABLE DES JOUEURS DU FILM, elle, ne dependant que du film, ne passe pas par la voie : la tache de fond
// la lit (film_scan_fond.go) et le fil principal l attend a son rang de publication, dans `lireSieges`,
// d ou il compose la lecture complete.

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar"

// voieDesSieges est la voie des sieges et des occupants.
type voieDesSieges struct {
	*voie
	occupants *attendue[lectureOccupants]
}

// lancerLaVoieDesSieges ouvre la voie des sieges sur `fc` (contexte derive, qu elle seule touche) et la lance.
func lancerLaVoieDesSieges(fc *grammar.FilmContext) *voieDesSieges {
	l := &voie{fc: fc}
	s := &voieDesSieges{voie: l, occupants: nouvelleAttendue[lectureOccupants](l)}
	l.lancer(func() {
		if !s.arretee() {
			livrer(s.occupants, lireLesOccupants(s.fc))
		}
	})
	return s
}

// occupants rend la lecture des equipes, des occupants et du tableau des scores que la voie a faite, et
// faux sans voie (la lecture se fait alors en serie, dans `lireSiegesEtOccupantsLecture`). Fil principal.
func (v *voiesDuBalayage) occupants() (lectureOccupants, bool) {
	if v == nil {
		return lectureOccupants{}, false
	}
	return attendreVoie(v, v.sieges.occupants), true
}
