package grammar

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// recuperationDeReference refait la recuperation d une liste comme le faisaient les canaux avant
// qu elle soit partagee : le localisateur complet ([debutRecupere]), puis la lecture de la vue B sous
// le monde de la marche, rendu intact.
func recuperationDeReference(m *marcheurDesTrames, pay []byte) (recs []FrameRecord, lus, libre bool) {
	cfg := m.cfg
	cfg.Obs = nil
	s, libre := debutRecupere(pay, m.monde, cfg)
	if s < 0 {
		return nil, false, false
	}
	defer m.monde.Restore(m.monde.Snapshot())
	br := LecteurSur(pay)
	br.poserCadre(cfg)
	br.Skip(s)
	recs, _ = DecodeFrameRecords(br, m.monde, cfg)
	return recs, true, libre
}

// TestLaListeRecupereeEstLueUneFoisParTrame : sur la bobine, chaque liste que la marche ne localise
// pas est recuperee une seule fois quel que soit le nombre de canaux qui la demandent, et tous recoivent
// ce que la recuperation complete rendait. Garde aussi ce que le raccourci suppose : sur une telle trame,
// la signature stricte est absente, donc le localisateur complet et le repli seul rendent le meme debut.
func TestLaListeRecupereeEstLueUneFoisParTrame(t *testing.T) {
	film, err := source.LoadDir(bobineMarcheDir(), nil)
	if err != nil {
		t.Fatalf("bobine illisible : %v", err)
	}
	m, err := NewFilmContext(film).nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	recuperees, vides := 0, 0
	m.parcourir(func(tr *trameLue) bool {
		if tr.debut >= 0 || !listeAnnoncee(&tr.paquet.VueA) {
			return true
		}
		pay := tr.paquet.Payload
		cfg := m.cfg
		cfg.Obs = nil
		if s := marchLocateStrict(pay, m.monde, cfg); s >= 0 {
			t.Fatalf("paquet %d du chunk %d : liste non localisee alors que la signature stricte est en %d",
				tr.paquet.Index, tr.paquet.Chunk, s)
		}
		complet, completLibre := debutRecupere(pay, m.monde, cfg)
		if seul, seulLibre := debutApresLaStricte(pay, m.monde, cfg); seul != complet || seulLibre != completLibre {
			t.Fatalf("paquet %d : repli seul (%d, %v), localisateur complet (%d, %v)", tr.paquet.Index, seul,
				seulLibre, complet, completLibre)
		}
		attendu, attenduLus, attenduLibre := recuperationDeReference(m, pay)
		md := &MarcheDistribuee{Paquet: tr.paquet, marche: m}
		premier, lus1, libre1 := md.recupererLaListe()
		second, lus2, libre2 := md.recupererLaListe()
		for i, g := range [][]FrameRecord{premier, second} {
			if !reflect.DeepEqual(g, attendu) {
				t.Fatalf("paquet %d, canal %d : records differents de la recuperation complete", tr.paquet.Index, i+1)
			}
		}
		if lus1 != attenduLus || lus2 != attenduLus || libre1 != attenduLibre || libre2 != attenduLibre {
			t.Fatalf("paquet %d : verdict (%v, %v) et (%v, %v) ; attendu (%v, %v)", tr.paquet.Index, lus1, libre1,
				lus2, libre2, attenduLus, attenduLibre)
		}
		if len(premier) > 0 {
			recuperees++
			if &premier[0] != &second[0] {
				t.Fatalf("paquet %d : la recuperation a ete refaite pour le second canal", tr.paquet.Index)
			}
		} else {
			vides++
		}
		return true
	})
	if recuperees == 0 {
		t.Fatal("aucune liste recuperee avec des records sur la bobine : le test ne garde rien")
	}
	t.Logf("%d liste(s) recuperee(s) avec des records, %d sans", recuperees, vides)
}
