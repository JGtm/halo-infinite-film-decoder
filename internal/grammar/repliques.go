package grammar

// repliques.go — LES REPLIQUES DES SPARTANS ET LES ANNONCES DE LA VUE A (genres 75 et 76).
//
// Le genre 75 (`AIDialog`, [chargeDialogueIA]) est une replique prononcee par une unite : priorite,
// identifiant du son, indice de replique, index de joueur ; son locuteur est la reference 0 de
// l en-tete. Le genre 76 (`Dialogue2D`, [chargeDialogue2D]) est une annonce de l annonceur ou d un
// script de mode : identifiant du son et masque de 32 bits des destinataires. L attribution des deux
// genres est celle de la bibliotheque libfilmshell de Den (MIT), TEMOIN de cette lecture : le nom de
// chaque genre est celui de la table du jeu ([tableDesGenresVueA], `vue_a_genres_test.go`) ; l ordre et
// les largeurs des champs sont ceux des deux lecteurs du jeu, deja portes par les charges.
//
// Les deux genres se traversent toujours ; leurs valeurs ne se rangent que sous un lecteur inscrit
// ([canalDeRepliques]) : sans lui la lecture ne garde rien, et sa fin est la meme.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// Les genres de dialogue de la vue A.
const (
	// GenreDialogueIA est le genre `AIDialog`.
	GenreDialogueIA = 75
	// GenreDialogue2D est le genre `Dialogue2D`.
	GenreDialogue2D = 76
)

// canalDeRepliques est un canal de tete qui lit les messages des genres 75 et 76 : tant qu un canal
// l est dans une distribution, la lecture de la vue A les range dans [lecture.VueA.Repliques] et
// [lecture.VueA.Annonces] ; sans lui elle n en garde rien.
type canalDeRepliques interface {
	CanalDesTetes
	// LitLesRepliques dit que le canal lit les genres 75 et 76.
	LitLesRepliques() bool
}

// veutLesRepliques dit si l un des canaux de tete lit les genres 75 et 76.
func veutLesRepliques(tetes []CanalDesTetes) bool {
	for _, c := range tetes {
		if r, ok := c.(canalDeRepliques); ok && r.LitLesRepliques() {
			return true
		}
	}
	return false
}

// recueillirUnDialogue range, dans `out`, le message de genre 75 ou 76 que [lireLaVueA] vient de lire
// a la position `debut`, quand un lecteur est inscrit ; il rend vrai pour ces deux genres, inscrits ou
// non, pour que l appelant n en fasse rien d autre.
func recueillirUnDialogue(br *Lecteur, out *FluxVueA, genre, debut int) bool {
	if genre != GenreDialogueIA && genre != GenreDialogue2D {
		return false
	}
	if !br.vueA.repliques {
		return true
	}
	if genre == GenreDialogueIA {
		m := br.repliqueLue
		m.Debut = uint32(debut) //nolint:gosec // position dans un payload
		if r := br.refsDuMessage[0]; r.presente {
			m.Ref, m.Gen = uint16(r.index), uint8(r.gen) //nolint:gosec // index de domaine, generation 2 bits
			m.Drapeaux |= lecture.RepliqueRef0
		}
		out.Repliques = append(out.Repliques, m)
		return true
	}
	m := br.annonceLue
	m.Debut = uint32(debut) //nolint:gosec // position dans un payload
	out.Annonces = append(out.Annonces, m)
	return true
}
