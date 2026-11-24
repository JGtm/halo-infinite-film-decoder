package grammar

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// repliques_test.go — les vecteurs des genres 75 (`AIDialog`) et 76 (`Dialogue2D`) de la vue A, ecrits
// d apres les deux lecteurs du jeu (`FUN_140f2e634`, `FUN_140f2e87c`) avec leurs largeurs en litteraux.

// grammaireAvecRepliques est la grammaire d un film recent sous un lecteur de repliques inscrit.
func grammaireAvecRepliques() grammaireDeLaVueA {
	g := grammaireRecente()
	g.repliques = true
	return g
}

// ecrireReplique ecrit un message de genre 75 : la reference 0 portee (domaine 1 : bit de sonde a 1, donc
// neuf bits, puis deux bits de generation), la priorite, le son, l indice, l index de joueur et deux bits
// de queue.
func (w *bitWriter) ecrireReplique(ref, gen, priorite, son, ligne, joueur, queue uint64) {
	w.bit(1)
	w.bits(75, 7)
	w.bit(1)
	w.bit(1)
	w.bits(ref, 9)
	w.bits(gen, 2)
	w.bits(0, 2) // references 1 et 2 absentes
	w.bits(priorite, 5)
	w.bit(1)
	w.bits(son, 32)
	w.bit(0)
	w.bits(ligne, 12)
	w.bit(0)
	w.bits(joueur, 5)
	w.bit(1) // R(1) = 1 : deux bits de queue
	w.bits(queue, 2)
}

// ecrireAnnonce ecrit un message de genre 76 : aucune reference, le son, une seconde valeur gardee de 32
// bits, un index de joueur absent et le masque des destinataires.
func (w *bitWriter) ecrireAnnonce(son, opaque, masque uint64) {
	w.bit(1)
	w.bits(76, 7)
	w.bits(0, 3)
	w.bit(1)
	w.bits(son, 32)
	w.bit(1)
	w.bits(opaque, 32)
	w.bit(1) // index de joueur absent
	w.bits(masque, 32)
}

// TestLesRepliquesEtLesAnnoncesSeLisentAvecLeursValeurs : un message de chaque genre, entoure d un genre
// vide, se lit avec ses valeurs, et la lecture s arrete au meme bit que la lecture sans lecteur inscrit
// (le terminateur, ecrit d apres l ecrivain). MUTATIONS — la priorite lue sur 4 bits, l indice de
// replique sur 11, le masque des destinataires sur 31, la queue de la replique sur un seul bit : ROUGES.
func TestLesRepliquesEtLesAnnoncesSeLisentAvecLeursValeurs(t *testing.T) {
	var w bitWriter
	w.bit(1) // bit de configuration
	w.ecrireReplique(300, 2, 13, 0xcafebabe, 0xabc, 17, 2)
	w.ecrireEnTeteDeMessage(103) // genre vide
	w.ecrireAnnonce(0x01020304, 0x0a0b0c0d, 0xa0000001)
	w.bit(0) // terminateur
	fin := w.n
	w.bits(0, 16)
	repliqueAttendue := lecture.MessageDeReplique{Debut: 1, Son: 0xcafebabe, Ref: 300, Ligne: 0xabc, Gen: 2,
		Priorite: 13, Joueur: 17, Drapeaux: lecture.RepliqueRef0 | lecture.RepliqueSon | lecture.RepliqueLigne,
		Queue: 2, LargeurQueue: 2}

	inscrit := lireSous(w.buf, grammaireAvecRepliques())
	nu := lireSous(w.buf, grammaireRecente())
	for nom, a := range map[string]FluxVueA{"inscrit": inscrit, "sans lecteur": nu} {
		if !a.Porte || a.Fin != fin || !slices.Equal(a.Genres, []int{75, 103, 76}) {
			t.Errorf("%s : vue A %+v, attendu portee jusqu au bit %d avec les genres 75, 103, 76", nom, a, fin)
		}
	}
	if len(nu.Repliques) != 0 || len(nu.Annonces) != 0 {
		t.Errorf("sans lecteur inscrit, rien ne se range : %+v %+v", nu.Repliques, nu.Annonces)
	}
	if len(inscrit.Repliques) != 1 || inscrit.Repliques[0] != repliqueAttendue {
		t.Errorf("repliques %+v, attendu %+v", inscrit.Repliques, repliqueAttendue)
	}
	if len(inscrit.Annonces) != 1 {
		t.Fatalf("annonces %+v, attendu une", inscrit.Annonces)
	}
	got := inscrit.Annonces[0]
	if got.Son != 0x01020304 || got.Opaque != 0x0a0b0c0d || got.Destinataires != 0xa0000001 ||
		got.Joueur != lecture.RefAbsente || got.Drapeaux != lecture.AnnonceSon|lecture.AnnonceOpaque {
		t.Errorf("annonce %+v", got)
	}
	var p lecture.Paquet
	p.Payload = w.buf
	rangerLaVueA(&p, &inscrit)
	if len(p.VueA.Repliques) != 1 || len(p.VueA.Annonces) != 1 {
		t.Errorf("rangees : %d repliques, %d annonces, attendu 1 et 1", len(p.VueA.Repliques), len(p.VueA.Annonces))
	}
}

// TestUneChargeDeDialogueTronqueeArreteLaLecture : un message dont la charge deborde le payload arrete la
// lecture apres son genre, et ne range rien.
func TestUneChargeDeDialogueTronqueeArreteLaLecture(t *testing.T) {
	var w bitWriter
	w.bit(1)
	w.bit(1)
	w.bits(75, 7)
	w.bits(0, 3)
	w.bits(13, 5)
	w.bit(1)
	w.bits(0xcafe, 16) // le son de 32 bits s arrete a mi-chemin
	fin := 1 + 1 + 7
	a := lireSous(w.buf, grammaireAvecRepliques())
	if a.Porte || a.Fin != fin || !slices.Equal(a.Genres, []int{75}) || len(a.Repliques) != 0 {
		t.Errorf("genre 75 tronque : %+v, attendu arretee au bit %d, sans replique", a, fin)
	}
	var v bitWriter
	v.bit(1)
	v.bit(1)
	v.bits(76, 7)
	v.bits(0, 3)
	v.bit(1)
	v.bits(0xcafe, 16)
	if a := lireSous(v.buf, grammaireAvecRepliques()); a.Porte || a.Fin != fin || len(a.Annonces) != 0 {
		t.Errorf("genre 76 tronque : %+v, attendu arretee au bit %d, sans annonce", a, fin)
	}
}

// TestLaQueueDeLaRepliqueSeLitSurUnOuDeuxBits : le R(1) de la queue donne sa largeur moins un ; avec 0, la
// queue tient sur un bit et le message suivant se lit au bit qui la suit.
func TestLaQueueDeLaRepliqueSeLitSurUnOuDeuxBits(t *testing.T) {
	var w bitWriter
	w.bit(1)
	w.bit(1)
	w.bits(75, 7)
	w.bits(0, 3)
	w.bits(5, 5)
	w.bit(0) // son absent
	w.bit(1) // indice absent
	w.bit(1) // joueur absent
	w.bit(0) // queue sur un bit
	w.bit(1) // la valeur de la queue
	w.bit(0) // terminateur
	fin := w.n
	w.bits(0, 16)
	a := lireSous(w.buf, grammaireAvecRepliques())
	attendue := lecture.MessageDeReplique{Debut: 1, Priorite: 5, Joueur: lecture.RefAbsente, Queue: 1, LargeurQueue: 1}
	if !a.Porte || a.Fin != fin || len(a.Repliques) != 1 || a.Repliques[0] != attendue {
		t.Errorf("vue A %+v, attendu portee jusqu au bit %d avec la replique %+v", a, fin, attendue)
	}
}

// fauxCanalDeRepliques est un canal de tete qui dit lire (ou non) les genres 75 et 76.
type fauxCanalDeRepliques struct{ lit bool }

func (fauxCanalDeRepliques) Interets() []Interet     { return nil }
func (fauxCanalDeRepliques) Clore(BilanDeMarche)     {}
func (fauxCanalDeRepliques) Tete(*lecture.Paquet)    {}
func (c fauxCanalDeRepliques) LitLesRepliques() bool { return c.lit }

// TestSeulUnCanalInscritFaitRangerLesRepliques : sans canal de repliques, ou avec un canal qui dit ne pas
// les lire, la lecture de la vue A n en range rien.
func TestSeulUnCanalInscritFaitRangerLesRepliques(t *testing.T) {
	cas := []struct {
		nom    string
		canaux []CanalDesTetes
		veut   bool
	}{
		{"aucun canal", nil, false},
		{"un canal de tete ordinaire", []CanalDesTetes{canalDesDegatsPourTest()}, false},
		{"un canal qui ne lit pas", []CanalDesTetes{fauxCanalDeRepliques{lit: false}}, false},
		{"un canal qui lit", []CanalDesTetes{canalDesDegatsPourTest(), fauxCanalDeRepliques{lit: true}}, true},
	}
	for _, c := range cas {
		if got := veutLesRepliques(c.canaux); got != c.veut {
			t.Errorf("%s : %v, attendu %v", c.nom, got, c.veut)
		}
	}
}

func canalDesDegatsPourTest() CanalDesTetes { return nouveauCanalDesDegats() }
