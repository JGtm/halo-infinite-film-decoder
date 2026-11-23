package grammar

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// canal_des_repliques_test.go — le canal des repliques et des annonces : ce qu il range d une trame, ce qu il compte, et la
// marche des mini-bobines a un et a quatre fils.

// TestLeCanalDesRepliquesRangeCeQueLaVueAPorteEtCompteLesTrames : une trame rend ses repliques et ses annonces avec son
// horodatage et leurs drapeaux, les trames se comptent par etat de leur vue A, et le canal s inscrit comme lecteur des genres 75
// et 76. MUTATIONS : un drapeau lu sur le mauvais bit, l horodatage de la trame precedente, une annonce rangee comme replique,
// l inscription retiree -> ROUGE.
func TestLeCanalDesRepliquesRangeCeQueLaVueAPorteEtCompteLesTrames(t *testing.T) {
	c := nouveauCanalDesRepliques()
	if !veutLesRepliques([]CanalDesTetes{c}) {
		t.Fatal("le canal ne s inscrit pas comme lecteur des genres 75 et 76")
	}
	trame := func(ts uint64, etat lecture.EtatDeVue, rs []lecture.MessageDeReplique, as []lecture.MessageDAnnonce) {
		p := &lecture.Paquet{TS: ts}
		p.VueA.Etat, p.VueA.Repliques, p.VueA.Annonces = etat, rs, as
		c.Tete(p)
	}
	complete := lecture.MessageDeReplique{Son: 0xcafe, Ref: 300, Gen: 2, Ligne: 77, Priorite: 7, Joueur: 3,
		Drapeaux: lecture.RepliqueRef0 | lecture.RepliqueSon | lecture.RepliqueLigne}
	nue := lecture.MessageDeReplique{Priorite: 13, Joueur: lecture.RefAbsente}
	trame(100, lecture.VueTerminee, []lecture.MessageDeReplique{complete, nue},
		[]lecture.MessageDAnnonce{{Son: 0x1234, Destinataires: 0x01000000, Joueur: 0, Drapeaux: lecture.AnnonceSon}})
	trame(200, lecture.VueArretee, nil, []lecture.MessageDAnnonce{{Destinataires: 0x80000000, Joueur: lecture.RefAbsente}})
	trame(300, lecture.VueTerminee, nil, nil)

	wantR := []RepliqueLue{
		{TS: 100, Son: 0xcafe, Locuteur: 300, GenLocuteur: 2, Ligne: 77, Priorite: 7, Joueur: 3, ASon: true, ALigne: true, ALocuteur: true},
		{TS: 100, Priorite: 13, Joueur: lecture.RefAbsente},
	}
	wantA := []AnnonceLue{
		{TS: 100, Son: 0x1234, Destinataires: 0x01000000, Joueur: 0, ASon: true},
		{TS: 200, Destinataires: 0x80000000, Joueur: lecture.RefAbsente},
	}
	got := c.resultat()
	if !reflect.DeepEqual(got.Repliques, wantR) || !reflect.DeepEqual(got.Annonces, wantA) {
		t.Errorf("repliques %+v, annonces %+v\nattendu %+v, %+v", got.Repliques, got.Annonces, wantR, wantA)
	}
	if want := (StatsRepliques{Trames: 3, Terminees: 2, Arretees: 1, Repliques: 2, Annonces: 2}); got.Stats != want {
		t.Errorf("comptes %+v, attendu %+v", got.Stats, want)
	}
}

// TestLeCanalDesRepliquesLitLesOctetsDUneTrame : de bout en bout, une vue A ecrite bit a bit (une replique, un genre vide, une
// annonce) passe par la lecture inscrite puis par le canal, et rend les valeurs ecrites. Sans canal inscrit, rien ne se range.
func TestLeCanalDesRepliquesLitLesOctetsDUneTrame(t *testing.T) {
	var w bitWriter
	w.bit(1)
	w.ecrireReplique(300, 2, 13, 0xcafebabe, 0xabc, 17, 2)
	w.ecrireEnTeteDeMessage(103)
	w.ecrireAnnonce(0x01020304, 0x0a0b0c0d, 0xa0000001)
	w.bit(0)
	w.bits(0, 16)
	flux := lireSous(w.buf, grammaireAvecRepliques())
	p := lecture.Paquet{TS: 4242, Payload: w.buf}
	rangerLaVueA(&p, &flux)
	c := nouveauCanalDesRepliques()
	c.Tete(&p)
	got := c.resultat()
	wantR := RepliqueLue{TS: 4242, Son: 0xcafebabe, Locuteur: 300, GenLocuteur: 2, Ligne: 0xabc, Priorite: 13, Joueur: 17,
		ASon: true, ALigne: true, ALocuteur: true}
	if len(got.Repliques) != 1 || got.Repliques[0] != wantR {
		t.Errorf("repliques %+v, attendu %+v", got.Repliques, wantR)
	}
	wantA := AnnonceLue{TS: 4242, Son: 0x01020304, Destinataires: 0xa0000001, Joueur: lecture.RefAbsente, ASon: true}
	if len(got.Annonces) != 1 || got.Annonces[0] != wantA {
		t.Errorf("annonces %+v, attendu %+v", got.Annonces, wantA)
	}
}

// TestLaMarcheDesTramesRendLesMemesRepliquesAUnEtAQuatreFils : sur la bobine de la marche, celle-ci rend les memes repliques,
// annonces et comptes a un et a quatre fils, et le canal a vu toutes les trames delta que celui des rechargements a vues. Le
// canal ne depend pas du pre-passage parallele des images-cles. MUTATION : un canal qui lit un etat que le pre-passage
// remplit sans le meme ordre -> ROUGE.
func TestLaMarcheDesTramesRendLesMemesRepliquesAUnEtAQuatreFils(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	var ref MarcheDesTrames
	for _, n := range []int{1, 4} {
		fc := NewFilmContext(film)
		fc.PoserFils(n)
		m, err := ScanMarcheDesTramesAvec(fc, LecturesDeLaMarche{})
		if err != nil {
			t.Fatalf("%d fils : %v", n, err)
		}
		if m.Repliques.Stats.Trames == 0 || m.Repliques.Stats.Trames != m.ReloadStats.Frames {
			t.Errorf("%d fils : le canal des repliques voit %d trames, celui des rechargements %d", n,
				m.Repliques.Stats.Trames, m.ReloadStats.Frames)
		}
		if n == 1 {
			ref = m
			t.Logf("bobine de la marche : %+v", m.Repliques.Stats)
			continue
		}
		if !reflect.DeepEqual(m.Repliques, ref.Repliques) {
			t.Errorf("%d fils : repliques differentes de celles d un fil", n)
		}
	}
}
