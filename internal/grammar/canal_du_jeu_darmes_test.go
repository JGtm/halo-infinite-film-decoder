package grammar

import (
	"reflect"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entitesDeTest est une table d entites fixe : un bipede au slot 3, un vehicule au slot 5.
type entitesDeTest map[uint32]lecture.Entite

func (e entitesDeTest) Entite(slot uint32) (lecture.Entite, bool) { x, ok := e[slot]; return x, ok }
func (e entitesDeTest) Liees() int                                { return len(e) }

// TestLeCanalDuJeuDArmesFiltreDedoublonneEtTrie : une lecture sur un slot lie au bipede est retenue ; une lecture
// sur un slot non lie ou lie a un autre archetype est ecartee et comptee ; la meme lecture (slot, instant)
// re-publiee est un doublon et non une lecture de plus ; la sortie est triee par (instant, slot) quel que soit
// l ordre d arrivee. MUTATIONS : filtre d archetype retire, cle de doublon sans slot, tri retire -> ROUGE.
func TestLeCanalDuJeuDArmesFiltreDedoublonneEtTrie(t *testing.T) {
	c := nouveauCanalDuJeuDArmes(Archetype{})
	c.st.Absent = false
	c.m = &MarcheDistribuee{Paquet: &lecture.Paquet{}, Entites: entitesDeTest{
		3: {TI: BipedTypeIndex}, 4: {TI: BipedTypeIndex}, 5: {TI: uint16(BipedTypeIndex + 1)}}}
	recoit := func(ts uint64, slot uint32, principal, second int) {
		c.m.Paquet.TS = ts
		c.recevoir(slot, JeuDArmes{Principal: principal, Second: second})
	}
	recoit(200, 4, 2, -1)
	recoit(100, 3, 0, 1)
	recoit(100, 3, 0, 1)  // seconde traversee du meme record
	recoit(100, 9, 1, -1) // slot non lie
	recoit(100, 5, 1, -1) // lie a un autre archetype
	recoit(100, 4, -1, -1)
	c.Clore(BilanDeMarche{})
	want := []types.DrawnWeaponRead{
		{TimestampUS: 100, Slot: 3, Principal: 0, Second: 1},
		{TimestampUS: 100, Slot: 4, Principal: -1, Second: -1},
		{TimestampUS: 200, Slot: 4, Principal: 2, Second: -1},
	}
	if !slices.Equal(c.out, want) {
		t.Errorf("lectures %+v, attendu %+v", c.out, want)
	}
	if st := c.st; st.Read != 3 || st.Duplicates != 1 || st.SlotUnbound != 2 || !st.Scanned || st.Absent {
		t.Errorf("comptes %+v", st)
	}
}

// TestLeCanalDuJeuDArmesSAbstientQuandLeComposantManque : un archetype sans `i42` ne pose ni interet ni crochet, et
// le canal le dit (`Absent`) au lieu de rendre « zero changement ».
func TestLeCanalDuJeuDArmesSAbstientQuandLeComposantManque(t *testing.T) {
	c := nouveauCanalDuJeuDArmes(Archetype{})
	if !c.st.Absent || c.Interets() != nil {
		t.Fatalf("un archetype vide doit etre absent et sans interet : %+v", c.st)
	}
	obs := NouvelleObservation()
	c.Brancher(obs, &MarcheDistribuee{})
	if obs.DesiredWeaponSetHook != nil {
		t.Error("un canal absent ne pose aucun crochet")
	}
}

// TestLaMarcheDesTramesRendLesMemesArmesTireesAUnEtAQuatreFils : sur la bobine de la marche, le canal lit des
// changements, tries, sur des slots lies au bipede, et rend les memes lectures et comptes a un et a quatre fils.
// MUTATION : un canal qui lit un etat que le pre-passage remplit sans le meme ordre -> ROUGE.
func TestLaMarcheDesTramesRendLesMemesArmesTireesAUnEtAQuatreFils(t *testing.T) {
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
		st := m.DrawnWeaponStats
		if !st.Scanned || st.Read != len(m.DrawnWeapons) {
			t.Errorf("%d fils : comptes %+v pour %d lectures", n, st, len(m.DrawnWeapons))
		}
		if !slices.IsSortedFunc(m.DrawnWeapons, func(a, b types.DrawnWeaponRead) int {
			if a.TimestampUS != b.TimestampUS {
				return int(int64(a.TimestampUS) - int64(b.TimestampUS))
			}
			return int(a.Slot) - int(b.Slot)
		}) {
			t.Errorf("%d fils : lectures non triees", n)
		}
		if n == 1 {
			ref = m
			t.Logf("bobine de la marche : %+v, %d lectures", st, len(m.DrawnWeapons))
			continue
		}
		if !reflect.DeepEqual(m.DrawnWeapons, ref.DrawnWeapons) || m.DrawnWeaponStats != ref.DrawnWeaponStats {
			t.Errorf("%d fils : armes tirees differentes de celles d un fil", n)
		}
	}
	if !ref.DrawnWeaponStats.Absent && len(ref.DrawnWeapons) == 0 {
		t.Error("la bobine porte i42 et aucune lecture n est rendue")
	}
}
