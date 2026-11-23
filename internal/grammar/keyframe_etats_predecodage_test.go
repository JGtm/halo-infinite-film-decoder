package grammar

// keyframe_etats_predecodage_test.go — le pre-calcul parallele des mots par record rend, a tous les
// nombres de fils, ce que la fenetre de bits calcule seule, et ne lance aucune goroutine a un fil.

import (
	"reflect"
	"runtime"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// payloadsDeLaBobine rend les payloads d image-cle d un contexte, avec leurs records (identite
// seule, dans l ordre des bits) tels que la distribution les range.
func payloadsDeLaBobine(fc *FilmContext) (pays [][]byte, recs [][]lecture.Record) {
	marche := fc.MarcheDImageCle()
	for _, num := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(num)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe || len(pk.Payload(data)) == 0 {
				continue
			}
			pay := pk.Payload(data)
			var rs []lecture.Record
			for _, b := range keyframeBornesDe(marche.Marcher(pay).Records) {
				rs = append(rs, lecture.Record{TI: int16(b.TI), Debut: uint32(b.Bit)}) //nolint:gosec // bornes du walker
			}
			pays, recs = append(pays, pay), append(recs, rs)
		}
	}
	return pays, recs
}

// TestMotsParRecordIndependantDuMuet : le resultat de [motsParRecord] pour un record ne depend pas
// du mutisme des autres records (le lemme du pre-calcul). Sur chaque payload, les records bipedes
// sont lus tous parlants, puis sous plusieurs masques de records muets ; pour chaque record resté
// parlant, ses familles et sa marque sont celles de la lecture sans mutisme.
func TestMotsParRecordIndependantDuMuet(t *testing.T) {
	jeux := []map[uint32]bool{catalogueDesFamilles(), carrierMarkViews}
	lus, records := 0, 0
	for _, nom := range bobinesAEmpreinte {
		fc := contexteDeBobine(chargerUneBobine(t, nom))
		pays, recs := payloadsDeLaBobine(fc)
		for i, pay := range pays {
			ref := motsParRecord(pay, recs[i], keyframeBipedTI, jeux...)
			refDe := func(k int) map[uint32][]uint32 {
				m := map[uint32][]uint32{}
				for _, rf := range ref[k] {
					m[rf.Rec.Debut] = rf.Families
				}
				return m
			}
			refFam, refMar := refDe(0), refDe(1)
			lus += len(refFam)
			for _, masque := range []func(j int) bool{
				func(j int) bool { return j%2 == 0 }, func(j int) bool { return j%2 == 1 },
				func(j int) bool { return j%3 != 0 }, func(int) bool { return true }, func(j int) bool { return j == 0 },
			} {
				muets := slices.Clone(recs[i])
				for j := range muets {
					if masque(j) {
						muets[j].TI = lecture.TINonResolu
					}
				}
				mots := motsParRecord(pay, muets, keyframeBipedTI, jeux...)
				for k, vus := range [2]map[uint32][]uint32{refFam, refMar} {
					got := map[uint32][]uint32{}
					for _, rf := range mots[k] {
						got[rf.Rec.Debut] = rf.Families
					}
					for j, r := range recs[i] {
						if masque(j) || int(r.TI) != keyframeBipedTI {
							continue
						}
						records++
						if !slices.Equal(got[r.Debut], vus[r.Debut]) {
							t.Fatalf("%s payload %d record %d jeu %d : %v sous mutisme, %v sans", nom, i, j, k,
								got[r.Debut], vus[r.Debut])
						}
					}
				}
			}
		}
	}
	if lus == 0 || records == 0 {
		t.Fatalf("%d records avec des mots, %d comparaisons : le test ne mord plus", lus, records)
	}
}

// TestFilsMotsIdentiques : le balayage des etats rend le meme resultat, compteurs de replis compris,
// a 1, 2, 4 et 8 fils ; les armes seules aussi.
func TestFilsMotsIdentiques(t *testing.T) {
	for _, court := range closureMiniFilms() {
		var refE EtatsDesImagesCles
		var refR ComptesDesReplis
		var refA any
		for _, n := range []int{1, 2, 4, 8} {
			fc := contexteDeLaBobine(t, court)
			fc.PoserFils(n)
			e, err := ScanEtatsDesImagesCles(fc, catalogueDesFamilles(), 0)
			if err != nil {
				t.Fatalf("%s %d fils : %v", court, n, err)
			}
			fa := contexteDeLaBobine(t, court)
			fa.PoserFils(n)
			armes, err := ScanArmesDesImagesCles(fa, catalogueDesFamilles())
			if err != nil {
				t.Fatalf("%s %d fils (armes) : %v", court, n, err)
			}
			if n == 1 {
				refE, refR, refA = e, fc.ComptesDesReplis(), armes
				if e.Admission.FenetresArmes == 0 {
					t.Fatalf("%s : aucun record non admis, le test ne mord pas", court)
				}
				continue
			}
			if !reflect.DeepEqual(e, refE) {
				t.Errorf("%s %d fils : etats differents de ceux d un fil", court, n)
			}
			if r := fc.ComptesDesReplis(); r != refR {
				t.Errorf("%s %d fils : replis %+v, un fil %+v", court, n, r, refR)
			}
			if !reflect.DeepEqual(armes, refA) {
				t.Errorf("%s %d fils : armes differentes de celles d un fil", court, n)
			}
		}
	}
}

// canalEspion observe, paquet par paquet, que la fenetre consomme le resultat pre-calcule.
type canalEspion struct {
	*canalDeLEtatCompletBipede
	consommes, calculesDirect int
}

func (c *canalEspion) ImageCle(p *lecture.Paquet, m *MarcheDistribuee) {
	cle := clePayload{debut: &p.Payload[0], n: len(p.Payload)}
	_, avait := c.motsAvance[cle]
	avant := c.out.Admission.FenetresArmes
	c.canalDeLEtatCompletBipede.ImageCle(p, m)
	if c.out.Admission.FenetresArmes == avant {
		return // aucun record non admis : la fenetre ne demande rien
	}
	if _, reste := c.motsAvance[cle]; avait && !reste {
		c.consommes++
	} else {
		c.calculesDirect++
	}
}

// TestFilsMotsPredecodesSontConsommes : a plusieurs fils, chaque payload dont un record est non admis
// est servi par le pre-calcul, jamais recalcule.
func TestFilsMotsPredecodesSontConsommes(t *testing.T) {
	fc := contexteDeLaBobine(t, closureMiniFilms()[0])
	fc.PoserFils(4)
	c := &canalEspion{canalDeLEtatCompletBipede: nouveauCanalDeLEtatComplet(fc, catalogueDesFamilles(), DefaultGrenadeMax)}
	c.predecoderLesMots()
	if len(c.motsAvance) == 0 {
		t.Fatal("rien de pre-calcule a 4 fils")
	}
	distribuerLesImagesClesSeules(fc, []Canal{c})
	if c.consommes == 0 || c.calculesDirect != 0 {
		t.Fatalf("%d payloads servis par le pre-calcul, %d calcules directement", c.consommes, c.calculesDirect)
	}
	if c.motsAvance != nil {
		t.Fatal("le pre-calcul doit etre vide a la cloture du canal")
	}
}

// TestFilsMotsAUnFilAucuneGoroutine : a un fil, le pre-calcul ne range rien et ne lance aucune
// goroutine.
func TestFilsMotsAUnFilAucuneGoroutine(t *testing.T) {
	fc := contexteDeLaBobine(t, closureMiniFilms()[0])
	c := nouveauCanalDeLEtatComplet(fc, catalogueDesFamilles(), DefaultGrenadeMax)
	avant := runtime.NumGoroutine()
	c.predecoderLesMots()
	if c.motsAvance != nil || runtime.NumGoroutine() > avant {
		t.Fatalf("a un fil : %d resultats ranges, %d goroutines (avant %d)", len(c.motsAvance), runtime.NumGoroutine(), avant)
	}
	if _, err := marcherLesImagesCles(fc, c); err != nil {
		t.Fatal(err)
	}
	if runtime.NumGoroutine() > avant {
		t.Fatalf("le balayage a un fil a laisse des goroutines : %d, avant %d", runtime.NumGoroutine(), avant)
	}
}
