package grammar

import (
	"encoding/hex"
	"reflect"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// lecteurDeRecordCapture ouvre un lecteur sur les bits d un record capture, du premier composant jusqu a la fin
// que la marche lui donne, le bit de poids fort d abord.
func lecteurDeRecordCapture(t *testing.T, h string) *Lecteur {
	t.Helper()
	b, err := hex.DecodeString(h)
	if err != nil {
		t.Fatal(err)
	}
	return LecteurSur(b)
}

// composantDeProjectileLu consomme un composant de projectile par le dispatch de production.
func composantDeProjectileLu(br *Lecteur, nom string) bool {
	_, _, _, porte := consumeByNameCapturing(br, nom, ProjectileTypeIndex, 0)
	consumeCorruptionCheck(br)
	return porte
}

// Les deux records ci-dessous viennent d un film d un contributeur, captures par la marche des trames : chacun est
// la suite des bits du record, de son premier composant jusqu au `FinBit` que la marche lui donne.

// TestLaLargeurDeI3SurUnRecordDeProjectileCapture : record ti=41 aux composants i0 i1 i2 i3 i13 i14 i15 i20 (209
// bits). i3 commence a +114 par un bit de porte a 0 : son deserialiseur lit R(1) puis R(19) puis R(8), 28 bits, et les
// quatre composants qui le suivent finissent exactement a la fin du record. Une autre largeur d i3 decalerait la fin.
func TestLaLargeurDeI3SurUnRecordDeProjectileCapture(t *testing.T) {
	br := lecteurDeRecordCapture(t, "0bd3a403a702d469cfc6b91f0a5fca420774734028000000208000")
	const debutI3, debutI13, fin = 114, 142, 209
	br.SetBitPos(debutI3)
	if !composantDeProjectileLu(br, "object-angular-velocity-component") {
		t.Fatal("i3 n est pas porte")
	}
	if br.BitPos() != debutI13 {
		t.Fatalf("i3 a pris %d bits, attendu 28 (porte a 0, R(19), R(8))", br.BitPos()-debutI3)
	}
	for _, nom := range []string{compObjectMaximumVitalities, compObjectDissolver, compObjectLowFrequency,
		"projectile-command_tick"} {
		if !composantDeProjectileLu(br, nom) {
			t.Fatalf("%s n est pas porte", nom)
		}
	}
	if br.BitPos() != fin {
		t.Fatalf("le record finit a +%d, le suivant de la marche commence a +%d", br.BitPos(), fin)
	}
}

// etatDeParenteCapture decode la branche « attache » du second record capture : un projectile ti=41 colle a un joueur,
// aux composants i1 i3 i10 i18 (134 bits), sans position.
func etatDeParenteCapture(t *testing.T) ObjectParentState {
	t.Helper()
	br := lecteurDeRecordCapture(t, "f18a002c1f75a12420445e4404f0732b20")
	const debutI10, debutI18, fin = 2, 113, 134
	for _, nom := range []string{composanteVitesseTranslation, "object-angular-velocity-component"} {
		if !composantDeProjectileLu(br, nom) {
			t.Fatalf("%s n est pas porte", nom)
		}
	}
	if br.BitPos() != debutI10 {
		t.Fatalf("i10 commence a +%d, attendu +%d", br.BitPos(), debutI10)
	}
	st := decodeObjectParentState(br, 3, ProjectileTypeIndex)
	if br.BitPos() != debutI18 {
		t.Fatalf("i10 finit a +%d, attendu +%d : la largeur de la branche n est pas celle du film", br.BitPos(), debutI18)
	}
	if !composantDeProjectileLu(br, "projectile-at-rest-state") {
		t.Fatal("i18 n est pas porte")
	}
	if br.BitPos() != fin {
		t.Fatalf("le record finit a +%d, attendu +%d", br.BitPos(), fin)
	}
	return st
}

// TestLaBrancheAttacheDeI10SurUnRecordDeProjectileCapture : la branche « attache » de i10 finit au bit ou i18 commence et
// le record finit ou la marche le termine : les largeurs de la branche (parent, noeud, trois valeurs de 16 bits) sont
// celles du film. Le parent est le slot 561, generation 1, le noeud 22.
func TestLaBrancheAttacheDeI10SurUnRecordDeProjectileCapture(t *testing.T) {
	st := etatDeParenteCapture(t)
	if !st.Attached {
		t.Fatal("la porte « attache » est fermee")
	}
	if slot := (st.Quant16 & parentHandleValueMask) + parentHandleBase; slot != 561 || st.Quant16>>30 != 1 {
		t.Fatalf("parent = slot %d generation %d, attendu 561 et 1", slot, st.Quant16>>30)
	}
	if st.Word16 != 22 || st.Mtx != [3]uint32{32214, 33936, 33041} {
		t.Fatalf("noeud %d et pose %v, attendus 22 et {32214 33936 33041}", st.Word16, st.Mtx)
	}
}

// enregistrementColle fabrique le record d un projectile (ti=41) dont i10 a les valeurs `st`, comme la couche de
// capture le rend.
func enregistrementColle(typ int, slot, gen uint32, st ObjectParentState, desalignement int) FrameRecord {
	st.TypeIndex = ProjectileTypeIndex
	return FrameRecord{Type: typ, ID: gen<<30 | slot, Slot: slot, TypeIndex: ProjectileTypeIndex, DesyncAt: desalignement,
		Trace: EntityTrace{Comps: []CompResult{{Payload: st, Ported: true}}}}
}

// TestLeCanalDesProjectilesCollesNeRetientQueLesCollagesAlignes : un delta ou une creation dont i10 ouvre la porte
// « attache » est une lecture, brute ; un i10 libre n est qu un record compte ; un record desaligne est ecarte et
// compte ; un autre archetype est ignore ; une seconde lecture de la meme vie est un repeat, non rendu. La sortie est triee par
// instant puis par projectile. MUTATIONS : la garde du desalignement retiree, les creations ignorees, le tri
// retire -> ROUGE.
func TestLeCanalDesProjectilesCollesNeRetientQueLesCollagesAlignes(t *testing.T) {
	colle := etatDeParenteCapture(t)
	libre := ObjectParentState{Attached: false}
	c := &canalDesProjectilesColles{vies: map[uint32]vieDeProjectile{}}
	lire := func(ts uint64, r FrameRecord) { c.lire(&lecture.Paquet{TS: ts}, &r) }

	lire(50, enregistrementColle(recNew, 1133, 1, libre, -1)) // la creation de la vie qui se colle plus tard
	lire(300, enregistrementColle(recDelta, 1133, 1, colle, -1))
	lire(100, enregistrementColle(recNew, 1200, 2, colle, -1))
	lire(100, enregistrementColle(recDelta, 1100, 0, libre, -1))
	lire(150, enregistrementColle(recDelta, 1133, 1, colle, 40)) // desaligne : rien ne prouve ce qu il porte
	lire(150, FrameRecord{Type: recDelta, TypeIndex: BipedTypeIndex, ID: 7, Slot: 7, DesyncAt: -1,
		Trace: EntityTrace{Comps: []CompResult{{Payload: colle, Ported: true}}}}) // pas un projectile
	lire(400, enregistrementColle(recDelta, 1133, 1, colle, -1)) // une seconde lecture de la meme vie
	c.Clore(BilanDeMarche{})

	parent := types.LifeKey{Slot: 561, Gen: 1}
	pose := [3]uint32{32214, 33936, 33041}
	want := []types.ProjectileAttachRead{
		{TimestampUS: 100, Life: types.LifeKey{Slot: 1200, Gen: 2}, BornUS: 100, HasBirth: true, Created: true, Parent: parent, Node: 22, Pose: pose},
		{TimestampUS: 300, Life: types.LifeKey{Slot: 1133, Gen: 1}, BornUS: 50, HasBirth: true, Parent: parent, Node: 22, Pose: pose},
	}
	if !slices.Equal(c.out, want) {
		t.Errorf("lectures %+v, attendu %+v", c.out, want)
	}
	attendu := types.ProjectileAttachStats{Scanned: true, Records: 5, Read: 2, Created: 1, Desynced: 1, Repeats: 1, NoTag: 2, NoCreator: 2}
	if c.st != attendu {
		t.Errorf("comptes %+v, attendu %+v", c.st, attendu)
	}
}

// TestLeCanalDesProjectilesCollesSAbstientQuandLeComposantManque : un registre sans i10 pour le projectile ne pose aucun
// interet et le canal le dit (`Absent`) au lieu de rendre « zero collage ».
func TestLeCanalDesProjectilesCollesSAbstientQuandLeComposantManque(t *testing.T) {
	c := nouveauCanalDesProjectilesColles(nil, &Registry{})
	if !c.st.Absent || c.Interets() != nil {
		t.Fatalf("un registre sans archetype projectile doit etre absent et sans interet : %+v", c.st)
	}
}

// TestLaMarcheDesTramesRendLesMemesProjectilesCollesAUnEtAQuatreFils : sur la bobine de la marche, le canal est joue
// (`Scanned`), ses comptes sont coherents avec ses lectures, aucun record lu n est desaligne, et un fil comme quatre
// rendent les memes lectures et les memes comptes.
func TestLaMarcheDesTramesRendLesMemesProjectilesCollesAUnEtAQuatreFils(t *testing.T) {
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
		st := m.ProjectileAttachStats
		if !st.Scanned || st.Read != len(m.ProjectileAttaches) || st.Read > st.Records || st.Created > st.Read {
			t.Errorf("%d fils : comptes %+v pour %d lectures", n, st, len(m.ProjectileAttaches))
		}
		if st.Desynced != 0 {
			t.Errorf("%d fils : %d record(s) de projectile desaligne(s) sur la bobine", n, st.Desynced)
		}
		if n == 1 {
			ref = m
			t.Logf("bobine de la marche : %+v", st)
			continue
		}
		if !reflect.DeepEqual(m.ProjectileAttaches, ref.ProjectileAttaches) || m.ProjectileAttachStats != ref.ProjectileAttachStats {
			t.Errorf("%d fils : projectiles colles differents de ceux d un fil", n)
		}
	}
	if ref.ProjectileAttachStats.Read == 0 {
		t.Error("la bobine porte des projectiles colles et aucune lecture n est rendue")
	}
	for _, l := range ref.ProjectileAttaches {
		if !l.HasBirth || !l.HasTag || l.BornUS > l.TimestampUS || (l.Created && l.BornUS != l.TimestampUS) {
			t.Errorf("la naissance d une lecture de la bobine est incoherente : %+v", l)
		}
		if l.HasCreator && l.Creator.Slot < parentHandleBase {
			t.Errorf("le createur n est pas dans l espace des slots d unites : %+v", l)
		}
	}
}

// TestUneCleReutiliseNeHeriteJamaisDeLaNaissanceDUneVieAncienne : la cle (slot, generation) revient apres quatre vies. Quand la
// creation de la vie courante n a pas ete lue, son collage sort sans naissance ; la suppression du slot et la creation d un autre
// type sous ce slot finissent aussi la vie. MUTATIONS : la suppression ignoree, la creation d un autre type ignoree, la vie
// non remplacee a la creation -> ROUGE.
func TestUneCleReutiliseNeHeriteJamaisDeLaNaissanceDUneVieAncienne(t *testing.T) {
	colle := etatDeParenteCapture(t)
	libre := ObjectParentState{Attached: false}
	c := &canalDesProjectilesColles{vies: map[uint32]vieDeProjectile{}}
	lire := func(ts uint64, r FrameRecord) { c.lire(&lecture.Paquet{TS: ts}, &r) }
	autre := func(typ int, slot, gen uint32) FrameRecord {
		return FrameRecord{Type: typ, ID: gen<<30 | slot, Slot: slot, TypeIndex: BipedTypeIndex, DesyncAt: -1}
	}

	lire(10, enregistrementColle(recNew, 1133, 1, libre, -1)) // vie A, creation lue
	lire(20, FrameRecord{Type: recDel, ID: 1<<30 | 1133, Slot: 1133, DesyncAt: -1})
	lire(900, enregistrementColle(recDelta, 1133, 1, colle, -1)) // meme cle, creation jamais lue : sans naissance
	lire(910, autre(recNew, 1133, 2))                            // un autre type prend le slot
	lire(920, enregistrementColle(recDelta, 1133, 1, colle, -1)) // la vie de la cle n existe plus : vie inconnue, sans naissance
	lire(930, enregistrementColle(recNew, 1200, 3, libre, -1))
	lire(940, enregistrementColle(recDelta, 1200, 3, colle, -1)) // creation lue : naissance
	lire(950, enregistrementColle(recNew, 1200, 3, libre, 40))   // creation desalignee de la cle suivante, meme generation
	lire(960, enregistrementColle(recDelta, 1200, 3, colle, -1)) // vie nouvelle sans naissance, pas celle de 930
	c.Clore(BilanDeMarche{})

	var sans, avec int
	for _, l := range c.out {
		if l.HasBirth {
			avec++
			if l.BornUS != 930 {
				t.Errorf("naissance %d, attendue 930 : %+v", l.BornUS, l)
			}
		} else {
			sans++
		}
	}
	if avec != 1 || sans != 3 || c.st.NoBirth != 3 {
		t.Errorf("avec naissance %d, sans %d, NoBirth %d ; attendus 1, 3 et 3", avec, sans, c.st.NoBirth)
	}
}
