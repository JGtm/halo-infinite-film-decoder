package grammar

// entete_delta_balayage_test.go — [prochainEnteteDelta] ET LE MARCHEUR DES BIPEDES QUI S'EN SERT,
// CONTRE LEUR FORME POSITION PAR POSITION.
//
// La reference de chaque test est l'ancienne boucle, gardee ici telle quelle : un test qui
// recoderait la regle de l'en-tete la verifierait contre sa propre copie.

import (
	"math/bits"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// enteteDeltaPossibleReference est le test de tete d'une position, lu bit a bit (zero hors du
// tampon).
func enteteDeltaPossibleReference(pay []byte, q, mcMin int) bool {
	return source.BitsTolerants(pay, q, 1) == 1 &&
		source.BitsTolerants(pay, q+porteDeLEnteteDelta, 2) == 0 &&
		int(source.BitsTolerants(pay, q+compteurDeLEnteteDelta, 3)) >= mcMin
}

// prochainEnteteDeltaReference est la recherche position par position.
func prochainEnteteDeltaReference(pay []byte, p, limite, mcMin int) int {
	for q := p; q < limite; q++ {
		if enteteDeltaPossibleReference(pay, q, mcMin) {
			return q
		}
	}
	return limite
}

// payloadsDuTestDeLaRecherche rend des payloads de 0 a 40 octets a trois densites, dont des
// payloads plus courts qu'un mot.
func payloadsDuTestDeLaRecherche(rng *rand.Rand) [][]byte {
	var out [][]byte
	for n := range 41 {
		for range 6 {
			pay := bruitDeTest(rng, n)
			// Un en-tete idealement place, pour que la recherche trouve autre chose que du bruit.
			if n > 0 {
				poserLesBits(pay, rng.IntN(n*8), fragmentDEnteteDeTest(rng, slotsDesPayloads[rng.IntN(5)]))
			}
			out = append(out, pay)
		}
	}
	return out
}

// comparerLaRecherche compare `f` a la recherche position par position, pour toute position de
// depart et plusieurs limites, y compris au-dela du tampon. Rend le nombre de positions trouvees.
func comparerLaRecherche(t *testing.T, f func(pay []byte, p, limite, mcMin int) int) (trouvees, appels int) {
	t.Helper()
	rng := rand.New(rand.NewPCG(11, 3))
	for _, pay := range payloadsDuTestDeLaRecherche(rng) {
		total := len(pay) * 8
		for p := 0; p <= total+3; p++ {
			for _, limite := range []int{p, p + 1, p + 20, p + 37, p + 44, p + 45, p + 64, total, total + 9} {
				for _, mcMin := range []int{1, 2} {
					want := prochainEnteteDeltaReference(pay, p, limite, mcMin)
					appels++
					if got := f(pay, p, limite, mcMin); got != want {
						t.Fatalf("len=%d p=%d limite=%d mcMin=%d : %d, attendu %d", len(pay), p, limite, mcMin, got, want)
					}
					if want < limite {
						trouvees++
					}
				}
			}
		}
	}
	return trouvees, appels
}

func TestProchainEnteteDeltaEgaleLaRechercheParPosition(t *testing.T) {
	trouvees, appels := comparerLaRecherche(t, prochainEnteteDelta)
	if trouvees == 0 {
		t.Fatal("aucune position trouvee : le test ne mesure rien")
	}
	t.Logf("%d recherches, dont %d trouvent une position", appels, trouvees)
}

// defautsDeLaRecherche regle les defauts qu'une copie de [prochainEnteteDelta] peut porter ; la
// valeur nulle est la forme de production.
type defautsDeLaRecherche struct {
	pasEnPlus      int  // positions sautees en changeant de mot (bord de mot)
	porteDecalee   bool // porte lue un bit trop tot
	sansBorneBasse bool // positions avant p non ecartees
	compteurDeUn   bool // compteur minimal 1 quel que soit mcMin
	sansLimite     bool // limite ignoree
}

// rechercheAvecDefauts est [prochainEnteteDelta] recopiee, avec les defauts demandes.
func rechercheAvecDefauts(d defautsDeLaRecherche) func(pay []byte, p, limite, mcMin int) int {
	return func(pay []byte, p, limite, mcMin int) int {
		for p < limite {
			i := p >> 3
			w := source.BitsAt(pay, i<<3, 64)
			porte := uint(porteDeLEnteteDelta)
			if d.porteDecalee {
				porte--
			}
			c := w &^ (w << porte) &^ (w << (porte + 1))
			if mcMin >= 2 && !d.compteurDeUn {
				c &= w<<compteurDeLEnteteDelta | w<<(compteurDeLEnteteDelta+1)
			} else {
				c &= w<<compteurDeLEnteteDelta | w<<(compteurDeLEnteteDelta+1) | w<<(compteurDeLEnteteDelta+2)
			}
			if !d.sansBorneBasse {
				c &= ^uint64(0) >> uint(p&7)
			}
			c &^= 1<<derniereLectureDelta - 1
			if c != 0 {
				q := i<<3 + bits.LeadingZeros64(c)
				if d.sansLimite {
					return q
				}
				return min(q, limite)
			}
			p = i<<3 + 64 - derniereLectureDelta + d.pasEnPlus
		}
		return limite
	}
}

// TestProchainEnteteDeltaDetecteSesMutants — le test differentiel voit un bord de mot mal gere, une
// porte decalee d'un bit, une borne basse ou une limite oubliee, un compteur de masque mal lu ; et la
// copie sans defaut ne diverge pas, de sorte que les detections ne viennent que du defaut.
func TestProchainEnteteDeltaDetecteSesMutants(t *testing.T) {
	if n := divergeLaRecherche(rechercheAvecDefauts(defautsDeLaRecherche{})); n != 0 {
		t.Fatalf("la copie sans defaut diverge %d fois de la reference", n)
	}
	mutants := map[string]defautsDeLaRecherche{
		"bord du mot (trois positions sautees)": {pasEnPlus: 3},
		"porte decalee d'un bit":                {porteDecalee: true},
		"sans borne basse":                      {sansBorneBasse: true},
		"compteur minimal ignore":               {compteurDeUn: true},
		"limite ignoree":                        {sansLimite: true},
	}
	for nom, d := range mutants {
		if divergeLaRecherche(rechercheAvecDefauts(d)) == 0 {
			t.Errorf("le mutant %q n'est pas detecte", nom)
		}
	}
}

// divergeLaRecherche compte les recherches ou `f` s'ecarte de la reference.
func divergeLaRecherche(f func(pay []byte, p, limite, mcMin int) int) int {
	rng := rand.New(rand.NewPCG(11, 3))
	n := 0
	for _, pay := range payloadsDuTestDeLaRecherche(rng) {
		total := len(pay) * 8
		for p := 0; p <= total+3; p++ {
			for _, limite := range []int{p + 1, p + 30, total} {
				for _, mcMin := range []int{1, 2} {
					if f(pay, p, limite, mcMin) != prochainEnteteDeltaReference(pay, p, limite, mcMin) {
						n++
					}
				}
			}
		}
	}
	return n
}

// enregistrementBipede est ce que le marcheur publie d'un record, sans le payload.
type enregistrementBipede struct {
	I0, Total int
	Slot, Gen uint32
	Mask      []int
}

func (r enregistrementBipede) de(d deltaBipedRecord) enregistrementBipede {
	return enregistrementBipede{I0: d.I0, Total: d.Total, Slot: d.Slot, Gen: d.Gen, Mask: d.Mask}
}

// walkDeltaBipedPayloadJusquaReference est l'ancienne boucle du marcheur : un test d'en-tete complet
// a chaque position.
func walkDeltaBipedPayloadJusquaReference(
	pay []byte, slots SlotBand, lay profile.I0Layout, gens *GenerationsVivantes, fin int,
	visit func(deltaBipedRecord),
) {
	total := len(pay) * 8
	i0Bits := lay.TotalBits()
	minRecord := deltaBipedMinRecord(i0Bits)
	for p := 0; p+minRecord <= total && p < fin; {
		i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, gens, lay)
		if !ok {
			p++
			continue
		}
		gen := LireHandleDelta(pay, i0-bipedHeaderBits-bipedIndexBits*len(idx)).Gen
		visit(deltaBipedRecord{Payload: pay, Total: total, I0: i0, Slot: slot, Gen: gen, Mask: idx})
		p = i0 + i0Bits // pas de re-scan chevauchant
	}
}

// comparerLeMarcheur joue les deux formes du marcheur sur `pay`, pour plusieurs bornes `fin` et
// plusieurs filtres de generation, et rend le nombre de records publies par la reference.
func comparerLeMarcheur(t *testing.T, pay []byte, slots SlotBand, gensDeTest []*GenerationsVivantes) int {
	t.Helper()
	total := len(pay) * 8
	acceptes := 0
	for _, gens := range gensDeTest {
		for _, fin := range []int{total, total + 5, total / 2, 1, 0, -3, total - 1} {
			var want, got []enregistrementBipede
			walkDeltaBipedPayloadJusquaReference(pay, slots, cliffLayout, gens, fin, func(d deltaBipedRecord) {
				want = append(want, enregistrementBipede{}.de(d))
			})
			walkDeltaBipedPayloadJusqua(pay, slots, cliffLayout, gens, fin, func(d deltaBipedRecord) {
				got = append(got, enregistrementBipede{}.de(d))
			})
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("len=%d fin=%d : %+v, reference %+v", len(pay), fin, got, want)
			}
			acceptes += len(want)
		}
	}
	return acceptes
}

// gensDeTest : aucun filtre, le repli nomme, tous les slots acceptes, des generations vivantes
// variees.
func gensDeTest() []*GenerationsVivantes {
	return []*GenerationsVivantes{
		nil,
		ToutesLesGenerations(),
		NouvellesGenerationsVivantes(nil),
	}
}

func TestMarcheurDeltaBipedeEgaleSaFormePositionParPosition(t *testing.T) {
	rng := rand.New(rand.NewPCG(5, 8))
	slots := bandeDesPayloads(t)
	gens := gensDeTest()
	acceptes, payloads := 0, 0
	for _, pay := range payloadsDeBord(rng, 517) {
		acceptes += comparerLeMarcheur(t, pay, slots, gens)
		payloads++
	}
	bord := acceptes
	for _, pay := range payloadsAleatoires(rng, 4000) {
		acceptes += comparerLeMarcheur(t, pay, slots, gens)
		payloads++
	}
	if bord == 0 || acceptes-bord == 0 {
		t.Fatalf("acceptations exercees : %d de bord, %d aleatoires", bord, acceptes-bord)
	}
	t.Logf("%d payloads, %d records publies (%d sur les payloads de bord)", payloads, acceptes, bord)
}

// TestMarcheurDeltaBipedeEgaleSaFormeSurLaMiniBobine : sur chaque payload delta de la bobine, ses
// decalages, ses troncatures et ses precedes de zeros, avec la bande de slots, le decoupage d'i0 et
// les generations vivantes (datees au paquet) du film.
func TestMarcheurDeltaBipedeEgaleSaFormeSurLaMiniBobine(t *testing.T) {
	fc, _ := contexteDesObjetsDuMonde(t)
	lay, err := fc.I0Layout()
	if err != nil {
		t.Fatalf("decoupage d'i0 : %v", err)
	}
	slots := fc.BipedSlots()
	rng := rand.New(rand.NewPCG(61, 7))
	payloads, acceptes := 0, 0
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			gens := fc.GenerationsVivantesA(pk.TimestampUS)
			for _, pay := range variantesDeBord(rng, pk.Payload(data)) {
				total := len(pay) * 8
				for _, fin := range []int{total, total / 2, 100} {
					var want, got []enregistrementBipede
					walkDeltaBipedPayloadJusquaReference(pay, slots, lay, gens, fin, func(d deltaBipedRecord) {
						want = append(want, enregistrementBipede{}.de(d))
					})
					walkDeltaBipedPayloadJusqua(pay, slots, lay, gens, fin, func(d deltaBipedRecord) {
						got = append(got, enregistrementBipede{}.de(d))
					})
					if !reflect.DeepEqual(want, got) {
						t.Fatalf("chunk %d paquet %d len=%d fin=%d : %d records, reference %d", c, pk.Index, len(pay), fin, len(got), len(want))
					}
					acceptes += len(want)
				}
				payloads++
			}
		}
	}
	if acceptes == 0 {
		t.Fatal("aucun record sur la bobine : le test ne prouve rien")
	}
	t.Logf("%d payloads (variantes comprises), %d records publies (tous `fin` confondus)", payloads, acceptes)
}
