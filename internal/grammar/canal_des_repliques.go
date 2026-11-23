package grammar

// canal_des_repliques.go — LES REPLIQUES DES SPARTANS ET LES ANNONCES DU FILM (genres 75 et 76 de la vue
// A), UN CANAL DE TETE DE LA MARCHE DES TRAMES.
//
// La lecture des deux genres est celle de [chargeDialogueIA] et [chargeDialogue2D] ; ce canal est le
// lecteur inscrit qui la fait ranger ([canalDeRepliques]). Il rend, dans l ordre du flux, ce que les
// messages portent avec l horodatage de leur trame, sans l interpreter : le locuteur d une replique est une
// REFERENCE d unite (la meme forme que la victime d un message de degat), le joueur un index de film ;
// les relier a un joueur du roster se fait a la publication, qui connait les pistes et les presences.
//
// Le canal ne depend ni du nombre de fils ni d un autre canal : il ne lit que la vue A rangee de chaque
// trame, comme les canaux des degats et des rechargements. Une trame dont la vue A s arrete sur un message
// non lu perd les messages qui le suivent : elle se compte ([StatsRepliques.Arretees]).

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"

// RepliqueLue est une replique de Spartan (genre 75) d une trame delta, avec l horodatage de la trame
// (microsecondes, horloge du film). Les valeurs ne valent que sous leur drapeau.
type RepliqueLue struct {
	TS uint64
	// Son est l identifiant du son (32 bits), valable quand ASon. Le film ne dit pas quelle variante
	// audio a joue.
	Son uint32
	// Locuteur et GenLocuteur sont l index d unite de la reference 0 de l en-tete et sa generation,
	// valables quand ALocuteur.
	Locuteur uint16
	// Ligne est l indice de replique (12 bits), valable quand ALigne.
	Ligne uint16
	// GenLocuteur : voir Locuteur.
	GenLocuteur uint8
	// Priorite est la classe de priorite de la replique (5 bits).
	Priorite uint8
	// Joueur est l index de joueur de la charge, ou [lecture.RefAbsente].
	Joueur                  int8
	ASon, ALigne, ALocuteur bool
}

// AnnonceLue est une annonce (genre 76) d une trame delta, avec l horodatage de la trame.
type AnnonceLue struct {
	TS uint64
	// Son est l identifiant du son (32 bits), valable quand ASon.
	Son uint32
	// Destinataires est le masque de 32 bits que la charge ecrit, tel que lu : le premier bit lu en est
	// le bit de poids fort.
	Destinataires uint32
	// Joueur est l index de joueur de la charge, ou [lecture.RefAbsente].
	Joueur int8
	ASon   bool
}

// StatsRepliques dit ce que le canal a vu : les trames delta, celles dont la vue A est lue jusqu a son
// terminateur, celles qu un message non lu arrete (les messages derriere cet arret sont perdus), et les
// messages rendus.
type StatsRepliques struct {
	Trames, Terminees, Arretees int
	Repliques, Annonces         int
}

// canalDesRepliques recueille les repliques et les annonces des trames delta d un film.
type canalDesRepliques struct {
	repliques []RepliqueLue
	annonces  []AnnonceLue
	stats     StatsRepliques
}

func nouveauCanalDesRepliques() *canalDesRepliques { return &canalDesRepliques{} }

// Interets : le canal ne lit que la vue A rangee de chaque trame.
func (c *canalDesRepliques) Interets() []Interet { return nil }

// Clore : rien, le canal se lit par ses champs.
func (c *canalDesRepliques) Clore(BilanDeMarche) {}

// LitLesRepliques inscrit le canal aupres de la lecture de la vue A : sans lui, les genres 75 et 76 se
// traversent sans rien ranger ([canalDeRepliques]).
func (c *canalDesRepliques) LitLesRepliques() bool { return true }

// Tete range les repliques et les annonces de la vue A de la trame `p`.
func (c *canalDesRepliques) Tete(p *lecture.Paquet) {
	st := &c.stats
	st.Trames++
	switch p.VueA.Etat {
	case lecture.VueTerminee:
		st.Terminees++
	case lecture.VueArretee:
		st.Arretees++
	}
	for _, m := range p.VueA.Repliques {
		c.repliques = append(c.repliques, RepliqueLue{TS: p.TS, Son: m.Son, Locuteur: m.Ref, Ligne: m.Ligne,
			GenLocuteur: m.Gen, Priorite: m.Priorite, Joueur: m.Joueur,
			ASon: m.Drapeaux&lecture.RepliqueSon != 0, ALigne: m.Drapeaux&lecture.RepliqueLigne != 0,
			ALocuteur: m.Drapeaux&lecture.RepliqueRef0 != 0})
		st.Repliques++
	}
	for _, m := range p.VueA.Annonces {
		c.annonces = append(c.annonces, AnnonceLue{TS: p.TS, Son: m.Son, Destinataires: m.Destinataires,
			Joueur: m.Joueur, ASon: m.Drapeaux&lecture.AnnonceSon != 0})
		st.Annonces++
	}
}

// RepliquesLues est ce que la marche des trames rend des genres 75 et 76.
type RepliquesLues struct {
	Repliques []RepliqueLue
	Annonces  []AnnonceLue
	Stats     StatsRepliques
}

func (c *canalDesRepliques) resultat() RepliquesLues {
	return RepliquesLues{Repliques: c.repliques, Annonces: c.annonces, Stats: c.stats}
}
