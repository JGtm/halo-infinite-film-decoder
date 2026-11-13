package grammar

import "reflect"

// marche_commune.go — UNE MARCHE DES TRAMES POUR LA SOURCE DE DEGAT ET POUR LA CUISSON.
//
// La source de degat et la cuisson lisent le meme film sous le meme profil de balayage : la cuisson
// recoit le profil que la source de degat a calibre. Quand l appelant leur donne le MEME contexte, la
// source de degat distribue en une marche ses deux canaux et ceux de la cuisson
// ([LireLaMarcheCommune]) ; le contexte garde ce que la cuisson en attend, avec le profil sous lequel
// la marche a tourne, et [ScanMarcheDesTramesAvec] le rend au lieu de remarcher. Un profil qui a
// change entre-temps, et la cuisson marche comme si rien n avait ete garde.

// marcheGardee est le resultat d une marche des trames que le contexte garde pour la cuisson.
type marcheGardee struct {
	m MarcheDesTrames
	// morts : le canal des morts de la marche commune, que la cuisson lit quand elle demande les
	// morts ([MarcheDesTrames.prendreLesMorts]).
	morts *canalDesMorts
	// profil : le profil de balayage du contexte quand la marche a tourne.
	profil ProfilDeBalayage
	// rendue : la cuisson l a deja prise ; une seconde demande marche.
	rendue bool
}

// veutLesMorts dit si ces lectures demandent les morts d objet et l occupation sur le film de `fc`.
func (l LecturesDeLaMarche) veutLesMorts(fc *FilmContext) bool {
	return l.Morts || (l.MortsSiVehicules && len(ScanWorldObjectKeyframes(fc, VehicleTypeIndex).Band) > 0)
}

// prendreLesMorts range dans `m` les morts d objet et l occupation que le canal `morts` a lues, et verse
// au rapport du contexte les listes qu il a recuperees a largeur libre.
func (m *MarcheDesTrames) prendreLesMorts(fc *FilmContext, morts *canalDesMorts) {
	m.ObjectDeaths, m.Occupancy, m.ObjectDeathStats = morts.resultat()
	fc.NoterReplis(ComptesDesReplis{LocalisationsALargeurLibre: morts.largeurLibre})
}

// marcheGardee rend la marche que le contexte garde pour la cuisson, une fois, quand le profil de
// balayage du contexte est celui sous lequel elle a tourne ; faux sinon, et la cuisson marche.
func (c *FilmContext) marcheGardee(l LecturesDeLaMarche) (MarcheDesTrames, bool) {
	if !c.GardeUneMarchePourLaCuisson() {
		return MarcheDesTrames{}, false
	}
	g := c.recup.marche
	g.rendue = true
	m := g.m
	if l.veutLesMorts(c) {
		m.prendreLesMorts(c, g.morts)
	}
	return m, true
}

// GardeUneMarchePourLaCuisson dit si le contexte garde une marche des trames que la cuisson n a pas
// encore prise et que son profil de balayage courant est celui sous lequel elle a tourne.
func (c *FilmContext) GardeUneMarchePourLaCuisson() bool {
	g := c.recup.marche
	return g != nil && !g.rendue && reflect.DeepEqual(g.profil, c.ProfilDeBalayage())
}

// LireLaMarcheCommune est [LireLaMarcheDeKillsource] sur une marche qui porte aussi les canaux de la
// cuisson : ce que la cuisson en attend reste au contexte ([FilmContext.marcheGardee]). Quand la
// marche de la cuisson ne peut pas tourner sur ce film (pas de chunk, archetype bipede illisible), la
// marche de la source de degat seule.
func LireLaMarcheCommune(fc *FilmContext) (LectureDeKillsource, error) {
	reg, err := fc.Registry()
	if err != nil {
		return LectureDeKillsource{}, err
	}
	morts, kills := nouveauCanalDesMorts(reg), nouveauCanalDesKills(fc)
	morts.garder = true
	profil := fc.ProfilDeBalayage()
	m, _, err := marcherPourLaCuisson(fc, LecturesDeLaMarche{}, []Canal{morts, kills})
	if err != nil {
		return LireLaMarcheDeKillsource(fc)
	}
	fc.recup.marche = &marcheGardee{m: m, morts: morts, profil: profil}
	return lectureDeKillsource(morts, kills), nil
}
