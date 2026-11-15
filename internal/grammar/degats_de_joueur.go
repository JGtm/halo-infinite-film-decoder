package grammar

// degats_de_joueur.go — LES DEGATS `damage_aftermath` DU FILM (genre 0 de la vue A).
//
// Un message `damage_aftermath` est un effet de degat qui touche une unite : il nomme l'unite blessee
// (reference 0), celle qui a cause le degat (reference 1, du meme domaine que la reference 0 d'un tir),
// l'effet de degat par son tag `jpt!` et l'index de joueur de l'auteur. Le canal les rend dans l'ordre du
// flux avec l'horodatage de leur trame, sans les interpreter : le tag ne se lit qu'avec la table des effets
// de degat, l'index de joueur qu'avec le roster.

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"

// GenreDegatsApres est le genre `damage_aftermath` de la vue A.
const GenreDegatsApres = 0

// DegatLu est un message `damage_aftermath` d'une trame delta, avec l'horodatage de la trame (microsecondes,
// horloge du film).
type DegatLu struct {
	TS uint64
	// Tag est l'identifiant de l'effet de degat, valable quand ATag.
	Tag uint32
	// Victime et Responsable sont les index d'unite des deux references, valables quand AVictime,
	// AResponsable.
	Victime, Responsable uint16
	// GenVictime est la generation de l unite blessee ; avec Victime elle designe sa vie.
	GenVictime uint8
	// Joueur est l'index de joueur de l'auteur, ou [lecture.RefAbsente].
	Joueur                       int8
	Echelle                      uint8
	ATag, AVictime, AResponsable bool
}

// StatsDegats dit ce que le canal a vu : les trames delta, celles dont la vue A est lue jusqu a son terminateur et celles qu un message non lu
// arrete (les messages derriere cet arret sont perdus), et les messages rendus.
type StatsDegats struct {
	Trames, Terminees, Arretees, Messages int
	// VitalesNonLues : les trames dont les records de vie des bipedes restent illisibles sans que la vue A s y arrete (le canal des vitalites les
	// compte, [StatsVitalites.TramesNonLues]) ; leurs points de degat se chiffrent sur une vie incomplete.
	VitalesNonLues int
}

// messageDeDegat range le message `damage_aftermath` que [chargeDegatsApres] vient de lire a la position
// `debut`, avec les deux references de son en-tete.
func messageDeDegat(br *Lecteur, debut int) lecture.DegatDeJoueur {
	d := br.degatLu
	d.Debut = uint32(debut) //nolint:gosec // position dans un payload
	if r := br.refsDuMessage[0]; r.presente {
		d.Victime, d.GenVictime = uint16(r.index), uint8(r.gen) //nolint:gosec // index d'unite, generation 2 bits
		d.Drapeaux |= lecture.DegatVictime
	}
	if r := br.refsDuMessage[1]; r.presente {
		d.Responsable, d.GenResponsable = uint16(r.index), uint8(r.gen) //nolint:gosec // idem
		d.Drapeaux |= lecture.DegatResponsable
	}
	return d
}

// canalDesDegats recueille les degats des trames delta d'un film.
type canalDesDegats struct {
	lus   []DegatLu
	stats StatsDegats
}

func nouveauCanalDesDegats() *canalDesDegats { return &canalDesDegats{} }

// Interets : le canal ne lit que la vue A rangee de chaque trame.
func (c *canalDesDegats) Interets() []Interet { return nil }

// Clore : rien — le canal se lit par son champ.
func (c *canalDesDegats) Clore(BilanDeMarche) {}

// Tete range les degats de la vue A de la trame `p`.
func (c *canalDesDegats) Tete(p *lecture.Paquet) {
	c.stats.Trames++
	switch p.VueA.Etat {
	case lecture.VueTerminee:
		c.stats.Terminees++
	case lecture.VueArretee:
		c.stats.Arretees++
	}
	for _, d := range p.VueA.Degats {
		c.lus = append(c.lus, DegatLu{TS: p.TS, Tag: d.Tag, Victime: d.Victime, GenVictime: d.GenVictime, Responsable: d.Responsable,
			Joueur: d.Joueur, Echelle: d.Echelle, ATag: d.Drapeaux&lecture.DegatTag != 0,
			AVictime: d.Drapeaux&lecture.DegatVictime != 0, AResponsable: d.Drapeaux&lecture.DegatResponsable != 0})
		c.stats.Messages++
	}
}
