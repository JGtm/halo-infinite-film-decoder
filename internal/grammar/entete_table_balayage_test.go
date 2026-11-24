package grammar

// entete_table_balayage_test.go — [prochainEnteteDeTable] ET LES DEUX RECHERCHES QUI S'EN SERVENT
// ([EntetesDeTable], [slotsDEntetesExacts]), CONTRE LEUR FORME POSITION PAR POSITION.
//
// La reference de chaque test est l'ancienne boucle, gardee ici telle quelle : un test qui
// recoderait la regle de l'eid la verifierait contre sa propre copie.

import (
	"math/bits"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// eidPossibleReference est le test d'eid d'une position, lu bit a bit (zero hors du tampon) : la
// generation voulue et l'emplacement sous la borne de la table.
func eidPossibleReference(pay []byte, q int, genNulle bool) bool {
	id := source.BitsTolerants(pay, q, 32)
	return id&masqueDeSlot < kfTableCap && (id>>30 == 0) == genNulle
}

// prochainEnteteDeTableReference est la recherche position par position.
func prochainEnteteDeTableReference(pay []byte, p, limite int, genNulle bool) int {
	for q := p; q < limite; q++ {
		if eidPossibleReference(pay, q, genNulle) {
			return q
		}
	}
	return limite
}

// TestLargeurDuSlotDeTable : la constante de largeur du rejet decoule de la borne de la table.
func TestLargeurDuSlotDeTable(t *testing.T) {
	if 1<<(30-largeurHauteDuSlot) != kfTableCap {
		t.Fatalf("%d bits d'emplacement nuls ne disent pas un emplacement sous %d", largeurHauteDuSlot, kfTableCap)
	}
}

// payloadsDeLaRechercheDeTable rend des payloads de 0 a 40 octets a trois densites, dont des
// payloads plus courts qu'un mot, semes de plages nulles de 19 a 40 bits (les candidats du rejet).
func payloadsDeLaRechercheDeTable(rng *rand.Rand) [][]byte {
	var out [][]byte
	for n := range 41 {
		for range 8 {
			pay := bruitDeTest(rng, n)
			for range rng.IntN(3) {
				if n == 0 {
					break
				}
				zeros := &bitWriter{}
				zeros.bits(uint64(rng.IntN(4)), 2) // generation
				zeros.bits(0, 17+rng.IntN(22))
				effacerLesBits(pay, rng.IntN(n*8), zeros.n)
			}
			out = append(out, pay)
		}
	}
	return out
}

// effacerLesBits met a zero `n` bits de `dst` a partir du bit `off`.
func effacerLesBits(dst []byte, off, n int) {
	for k := range n {
		if at := off + k; at>>3 < len(dst) {
			dst[at>>3] &^= 1 << (7 - uint(at&7))
		}
	}
}

// comparerLaRechercheDeTable compare `f` a la recherche position par position, pour toute position de
// depart, plusieurs limites (au-dela du tampon comprises) et les deux generations. Rend le nombre de
// recherches et celui de celles qui trouvent une position.
func comparerLaRechercheDeTable(
	t *testing.T, f func(pay []byte, p, limite int, genNulle bool) int,
) (trouvees, appels int) {
	t.Helper()
	rng := rand.New(rand.NewPCG(21, 4))
	for _, pay := range payloadsDeLaRechercheDeTable(rng) {
		total := len(pay) * 8
		for p := 0; p <= total+3; p++ {
			for _, limite := range []int{p, p + 1, p + 20, p + 45, p + 46, p + 64, total, total - 63, total + 9} {
				for _, genNulle := range []bool{false, true} {
					want := prochainEnteteDeTableReference(pay, p, limite, genNulle)
					appels++
					if got := f(pay, p, limite, genNulle); got != want {
						t.Fatalf("len=%d p=%d limite=%d genNulle=%v : %d, attendu %d", len(pay), p, limite, genNulle, got, want)
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

func TestProchainEnteteDeTableEgaleLaRechercheParPosition(t *testing.T) {
	trouvees, appels := comparerLaRechercheDeTable(t, prochainEnteteDeTable)
	if trouvees == 0 {
		t.Fatal("aucune position trouvee : le test ne mesure rien")
	}
	t.Logf("%d recherches, dont %d trouvent une position", appels, trouvees)
}

// defautsDeLaRechercheDeTable regle les defauts qu'une copie de [prochainEnteteDeTable] peut porter ;
// la valeur nulle est la forme de production.
type defautsDeLaRechercheDeTable struct {
	pasEnPlus      int  // positions sautees en changeant de mot (bord de mot)
	sansBorneBasse bool // positions avant p non ecartees
	generationLue  bool // la generation voulue est ignoree
	plageCourte    bool // seize bits d'emplacement nuls au lieu de dix-sept
	sansLimite     bool // limite ignoree
}

// rechercheDeTableAvecDefauts est [prochainEnteteDeTable] recopiee, avec les defauts demandes.
func rechercheDeTableAvecDefauts(d defautsDeLaRechercheDeTable) func(pay []byte, p, limite int, g bool) int {
	return func(pay []byte, p, limite int, genNulle bool) int {
		for p < limite {
			i := p >> 3
			w := source.BitsAt(pay, i<<3, 64)
			z := ^w
			z2 := z & (z << 1)
			z4 := z2 & (z2 << 2)
			z8 := z4 & (z4 << 4)
			z16 := z8 & (z8 << 8)
			plage := (z16 & (z << 16)) << premierBitHautDuSlot
			if d.plageCourte {
				plage = z16 << premierBitHautDuSlot
			}
			switch {
			case d.generationLue:
			case genNulle:
				plage &= z2
			default:
				plage &= w | w<<1
			}
			if !d.sansBorneBasse {
				plage &= ^uint64(0) >> uint(p&7)
			}
			if plage != 0 {
				q := i<<3 + bits.LeadingZeros64(plage)
				if d.sansLimite {
					return q
				}
				return min(q, limite)
			}
			p = i<<3 + positionsParMotDeTable + d.pasEnPlus
		}
		return limite
	}
}

// TestProchainEnteteDeTableDetecteSesMutants : le test differentiel voit un bord de mot mal gere, une
// borne basse ou une limite oubliee, une generation ignoree, une plage trop courte ; et la copie sans
// defaut ne diverge pas, de sorte que les detections ne viennent que du defaut.
func TestProchainEnteteDeTableDetecteSesMutants(t *testing.T) {
	diverge := func(f func(pay []byte, p, limite int, g bool) int) bool {
		rng := rand.New(rand.NewPCG(21, 4))
		for _, pay := range payloadsDeLaRechercheDeTable(rng) {
			total := len(pay) * 8
			for p := 0; p <= total+3; p++ {
				for _, limite := range []int{p + 1, p + 30, total} {
					for _, genNulle := range []bool{false, true} {
						if f(pay, p, limite, genNulle) != prochainEnteteDeTableReference(pay, p, limite, genNulle) {
							return true
						}
					}
				}
			}
		}
		return false
	}
	if diverge(rechercheDeTableAvecDefauts(defautsDeLaRechercheDeTable{})) {
		t.Fatal("la copie sans defaut diverge de la reference")
	}
	mutants := map[string]defautsDeLaRechercheDeTable{
		"bord du mot (trois positions sautees)": {pasEnPlus: 3},
		"sans borne basse":                      {sansBorneBasse: true},
		"generation ignoree":                    {generationLue: true},
		"plage d'emplacement trop courte":       {plageCourte: true},
		"limite ignoree":                        {sansLimite: true},
	}
	for nom, d := range mutants {
		if !diverge(rechercheDeTableAvecDefauts(d)) {
			t.Errorf("le mutant %q n'est pas detecte", nom)
		}
	}
}

// entetesDeTableReference est l'ancienne forme d'[EntetesDeTable] : un test d'en-tete complet a CHAQUE
// bit, dans les deux passes.
func entetesDeTableReference(pay []byte) ([]EnteteDeTable, int) {
	b := nouveauBalayageDeTable(pay)
	var premiere []entreeSituee
	suivant := uint32(0)
	for at := debutDeLaTableBit; at < b.fin; {
		e, ok := b.enteteA(at, false)
		switch {
		case !ok:
			at++
		case e.Slot() < suivant:
			b.ecartes[e.Slot()] = true
			at++
		default:
			premiere = append(premiere, entreeSituee{e, at})
			suivant = e.Slot() + 1
			at += teteDEntreeBits
		}
	}
	zeroEntre := func(a, c borneDeTable) []EnteteDeTable {
		var out []EnteteDeTable
		suivant := a.slot + 1
		for at := a.at + teteDEntreeBits; at < c.at-teteDEntreeBits && at < b.fin; {
			e, ok := b.enteteA(at, true)
			if !ok || e.Gen() != 0 {
				at++
				continue
			}
			if s := int(e.Slot()); s < suivant || s >= c.slot {
				b.ecartes[e.Slot()] = true
				at++
				continue
			}
			out = append(out, e)
			suivant = int(e.Slot()) + 1
			at += teteDEntreeBits
		}
		return out
	}
	out := make([]EnteteDeTable, 0, len(premiere)+16)
	prec := borneDeTable{slot: -1, at: debutDeLaTableBit - teteDEntreeBits}
	for i := 0; i <= len(premiere); i++ {
		suiv := borneDeTable{slot: kfTableCap, at: b.total}
		if i < len(premiere) {
			suiv = borneDeTable{slot: int(premiere[i].Slot()), at: premiere[i].bit}
		}
		out = append(out, zeroEntre(prec, suiv)...)
		if i < len(premiere) {
			out = append(out, premiere[i].EnteteDeTable)
			prec = suiv
		}
	}
	return out, len(b.ecartes)
}

// slotsDEntetesExactsReference est l'ancienne forme de [slotsDEntetesExacts] : une lecture de l'eid a
// chaque bit.
func slotsDEntetesExactsReference(pay []byte) map[int]bool {
	out := map[int]bool{}
	total := len(pay) * 8
	for q := 0; q+64 <= total; q++ {
		id := source.BitsBourres(pay, q, 32)
		if id>>30 == 0 {
			continue
		}
		slot := int(id & 0x3FFFFFFF)
		if slot >= kfTableCap {
			continue
		}
		if source.BitsBourres(pay, q+32, 32) == managedPlayerTypeIndex {
			out[slot] = true
		}
	}
	return out
}

// bilanDeLaComparaison compte ce qu'un test differentiel a exerce.
type bilanDeLaComparaison struct {
	payloads, positions       int    // positions : bits juges par la reference, par generation
	candidats                 [2]int // positions que le rejet garde, par generation (non nulle, nulle)
	entrees, ecartes, joueurs int    // sorties non vides de la reference
}

// comparerLesDeuxRecherches joue les deux formes de chaque recherche sur `pay` et compte ce qui a ete
// exerce. Les candidats sont enumeres par le rejet (premiere passe, generation non nulle ; seconde passe,
// generation nulle) et confrontes a ceux de la reference, position par position.
func comparerLesDeuxRecherches(t *testing.T, nom string, pay []byte, bilan *bilanDeLaComparaison) {
	t.Helper()
	wantE, wantA := entetesDeTableReference(pay)
	gotE, gotA := EntetesDeTable(pay)
	if !reflect.DeepEqual(wantE, gotE) || wantA != gotA {
		t.Fatalf("%s len=%d : EntetesDeTable %v (ecartes %d), reference %v (%d)", nom, len(pay), gotE, gotA, wantE, wantA)
	}
	wantJ := slotsDEntetesExactsReference(pay)
	if gotJ := slotsDEntetesExacts(pay); !reflect.DeepEqual(wantJ, gotJ) {
		t.Fatalf("%s len=%d : slotsDEntetesExacts %v, reference %v", nom, len(pay), gotJ, wantJ)
	}
	total := len(pay) * 8
	for _, genNulle := range []bool{false, true} {
		var want, got []int
		for q := 0; q < total; q++ {
			if eidPossibleReference(pay, q, genNulle) {
				want = append(want, q)
			}
		}
		for q := prochainEnteteDeTable(pay, 0, total, genNulle); q < total; q = prochainEnteteDeTable(pay, q+1, total, genNulle) {
			got = append(got, q)
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("%s len=%d genNulle=%v : candidats %v, reference %v", nom, len(pay), genNulle, got, want)
		}
		bilan.positions += total
		bilan.candidats[unSi(genNulle)] += len(got)
	}
	bilan.payloads++
	bilan.entrees += len(wantE)
	bilan.ecartes += wantA
	bilan.joueurs += len(wantJ)
}

// TestLesRecherchesDeTableEgalentLeurFormeSurLesBobines : sur chaque payload d'image-cle de deux bobines
// (les plus peuplees), ses decalages, ses troncatures et ses precedes de zeros, les deux recherches
// rendent ce que rendait la boucle position par position, et le rejet garde exactement les positions
// dont l'eid passe le test de la reference.
func TestLesRecherchesDeTableEgalentLeurFormeSurLesBobines(t *testing.T) {
	rng := rand.New(rand.NewPCG(33, 9))
	var bilan bilanDeLaComparaison
	for _, b := range []bobineEquipes{bobinesEquipes()[0], bobinesEquipes()[4]} {
		fc := NewFilmContext(bobineFilm(t, b.film))
		for _, c := range fc.ChunkNumbers() {
			data, pks, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				for _, pay := range variantesDeBord(rng, pk.Payload(data)) {
					comparerLesDeuxRecherches(t, "bobine", pay, &bilan)
				}
			}
		}
	}
	if bilan.entrees == 0 || bilan.joueurs == 0 {
		t.Fatalf("rien a comparer : %+v", bilan)
	}
	t.Logf("%d payloads, %d positions jugees (par generation), dont %d gardees par le rejet (generation non nulle), %d (generation nulle) ; "+
		"%d entrees de table, %d emplacements ecartes, %d en-tetes de joueur", bilan.payloads, bilan.positions/2,
		bilan.candidats[0], bilan.candidats[1], bilan.entrees, bilan.ecartes, bilan.joueurs)
}

// TestLesRecherchesDeTableEgalentLeurFormeAuxBords : une table et des en-tetes de joueur ecrits a tous
// les alignements (le candidat a cheval sur deux mots de 64 bits, au dernier bit du payload, ou le
// payload plus court qu'un mot), complets, puis tronques d'un a neuf octets.
func TestLesRecherchesDeTableEgalentLeurFormeAuxBords(t *testing.T) {
	var bilan bilanDeLaComparaison
	table := payloadDeTable(func(w *bitWriter) {
		w.ecrireEntreeVivante(eidDeTest(0, 0), 36)
		w.ecrireEntreeVide()
		w.ecrireEntreeVivante(eidDeTest(1, 2), 3)
		w.ecrireEntreeSupprimee(eidDeTest(0, 4), 41)
		w.ecrireEntreeVivante(eidDeTest(3, 6), 5)
		w.ecrireEntreeVivante(eidDeTest(0, 7), 10)
		w.ecrireEntreeVivante(eidDeTest(2, 8191), 9)
	})
	rng := rand.New(rand.NewPCG(5, 6))
	for k := range 136 {
		decale := decalerLesBits(rng, table, k)
		for cut := 0; cut <= 9; cut++ {
			comparerLesDeuxRecherches(t, "table", decale[:len(decale)-cut], &bilan)
		}
		comparerLesDeuxRecherches(t, "table-zeros", precederDeZeros(table, k), &bilan)
	}
	for _, gen := range []uint32{0, 1, 3} {
		for _, slot := range []uint32{0, 7, 8191, 8192, 1 << 20} {
			rec := &bitWriter{}
			rec.bits(uint64(eidDeTest(gen, slot)), 32)
			rec.bits(managedPlayerTypeIndex, 32)
			for off := range 136 {
				juste := (off + rec.n + 7) / 8
				for _, longueur := range []int{juste, juste - 1, juste + 1, 1, 7, 8, 9} {
					pay := make([]byte, longueur)
					poserLesBits(pay, off, rec)
					comparerLesDeuxRecherches(t, "joueur", pay, &bilan)
				}
			}
		}
	}
	if bilan.joueurs == 0 || bilan.entrees == 0 {
		t.Fatalf("rien a comparer : %+v", bilan)
	}
	t.Logf("%d payloads, %d positions jugees (par generation), dont %d gardees par le rejet (generation non nulle), %d (generation nulle) ; "+
		"%d entrees de table, %d emplacements ecartes, %d en-tetes de joueur", bilan.payloads, bilan.positions/2,
		bilan.candidats[0], bilan.candidats[1], bilan.entrees, bilan.ecartes, bilan.joueurs)
}
