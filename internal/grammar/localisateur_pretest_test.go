package grammar

// localisateur_pretest_test.go — LE PRE-TEST DE TETE DU LOCALISATEUR NE CHANGE AUCUNE DECISION.
//
// [marchLocateStrictReference] et [marchLocateFallbackReference] sont les formes du localisateur
// SANS pre-test : un essai complet de [TryDeltaAt] a chaque position precedee d un bit nul. Les
// formes de production doivent rendre la meme position, et ne jamais ecarter une position que la
// reference accepte. Les payloads viennent de la bobine reelle : les signatures et les repliers
// que ses paquets portent, recopiees dans du bruit, avec des leurres (tete juste, corps casse),
// des bits qui precedent mal, une generation fausse, un slot voisin.

import (
	"math/rand/v2"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

func marchLocateStrictReference(pay []byte, w *World, cfg FrameConfig) int {
	defer cfg.Obs.neutraliserLesCrochetsDeCanal()()
	nb, largeur := len(pay)*8, largeurDeSignature(cfg)
	for s := 2; s+largeur < nb; s++ {
		if !precedeDuTerminateur(pay, s) {
			continue
		}
		if marchSignature123(pay, s, largeur, w, cfg) {
			return s
		}
	}
	return -1
}

func marchLocateFallbackReference(pay []byte, w *World, cfg FrameConfig) int {
	defer cfg.Obs.neutraliserLesCrochetsDeCanal()()
	nb := len(pay) * 8
	for s := 2; s+resteMinimalALargeurLibre < nb; s++ {
		if !precedeDuTerminateur(pay, s) {
			continue
		}
		rec, _, ok := TryDeltaAt(pay, s, w, cfg)
		if !ok || rec.Slot != marchSignatureSlot ||
			!w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte) {
			continue
		}
		return s
	}
	return -1
}

// bitsDe rend les bits [debut, fin) de `pay`, un par octet (0 ou 1).
func bitsDuPayload(pay []byte, debut, fin int) []uint8 {
	out := make([]uint8, 0, fin-debut)
	for p := debut; p < fin; p++ {
		out = append(out, uint8(source.BitAt(pay, p)))
	}
	return out
}

// empaqueter range des bits (un par octet) en octets, MSB d abord.
func empaqueter(bits []uint8) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, b := range bits {
		if b != 0 {
			out[i/8] |= 0x80 >> (uint(i) & 7)
		}
	}
	return out
}

// bruit rend `n` bits au hasard.
func bruit(r *rand.Rand, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(r.IntN(2))
	}
	return out
}

// graine d une signature recueillie : les bits qui precedent la tete (le bit nul compris) et la
// signature elle-meme, et la longueur de sa tete.
type graineDeSignature struct {
	bits  []uint8 // le bit nul, puis le record
	tete  int     // bits de la tete (prefixe de champs, type, identifiant) apres le bit nul
	marge int     // bits du bit nul : toujours 1
}

// variantes construit les payloads d une graine. Chacun place la graine (ou un leurre) a une
// position aleatoire dans du bruit.
func variantes(r *rand.Rand, g graineDeSignature, cfg FrameConfig) [][]byte {
	pose := func(dst []uint8, pos int, src []uint8) {
		copy(dst[pos:], src)
	}
	nouveau := func(extra int) []uint8 { return bruit(r, len(g.bits)+extra) }
	var out [][]byte
	for range 6 {
		avant := 2 + r.IntN(300)
		apres := r.IntN(300)

		// V1 : la graine seule dans du bruit.
		b := bruit(r, avant+len(g.bits)+apres)
		pose(b, avant, g.bits)
		out = append(out, empaqueter(b))

		// V2 : un leurre AVANT la graine (tete juste, un bit du corps casse), qui peut la
		// chevaucher ; la graine, ecrite apres, gagne le chevauchement.
		b = bruit(r, avant+len(g.bits)+apres+80)
		leurre := append([]uint8(nil), g.bits...)
		k := 1 + g.tete + r.IntN(len(g.bits)-g.tete-1)
		leurre[k] ^= 1
		pose(b, avant, leurre)
		pose(b, avant+1+r.IntN(70), g.bits)
		out = append(out, empaqueter(b))

		// V3 : la graine, generation de l identifiant fausse (les deux bits qui suivent le
		// champ bas).
		b = bruit(r, avant+len(g.bits)+apres)
		fausse := append([]uint8(nil), g.bits...)
		gen := g.marge + motFacultatifDEnTete(cfg) + prefixeDeltaBits + cfg.IDLowBits
		fausse[gen] ^= 1
		pose(b, avant, fausse)
		out = append(out, empaqueter(b))

		// V4 : la graine precedee d un bit a UN, ni terminateur ni position essayable ; un
		// leurre bien precede la suit plus loin.
		b = bruit(r, avant+len(g.bits)+apres+len(g.bits)+4)
		sans := append([]uint8(nil), g.bits...)
		sans[0] = 1
		pose(b, avant, sans)
		pose(b, avant+len(g.bits)+2, g.bits)
		out = append(out, empaqueter(b))

		// V5 : la tete seule, suivie de bruit : une pluie de leurres qui passent le pre-test.
		b = nouveau(apres + 200)
		for range 4 {
			pose(b, 2+r.IntN(len(b)-len(g.bits)-3), g.bits[:g.marge+g.tete])
		}
		out = append(out, empaqueter(b))

		// V6 : deux signatures ; la premiere decide.
		b = bruit(r, avant+2*len(g.bits)+apres+9)
		pose(b, avant, g.bits)
		pose(b, avant+len(g.bits)+1+r.IntN(8), g.bits)
		out = append(out, empaqueter(b))

		// V7 : le slot voisin (un bit du champ bas change) : la tete ne porte pas la signature.
		b = bruit(r, avant+len(g.bits)+apres)
		voisin := append([]uint8(nil), g.bits...)
		voisin[g.marge+motFacultatifDEnTete(cfg)+prefixeDeltaBits+r.IntN(cfg.IDLowBits)] ^= 1
		pose(b, avant, voisin)
		out = append(out, empaqueter(b))
	}
	return out
}

// bilanDeComparaison compte ce qu une comparaison a exerce.
type bilanDeComparaison struct {
	payloads         int
	strictes, libres int // localisations rendues (>= 0) par les references
	acceptees        int // positions acceptees par les references, stricte ou libre
	leursStricts     int // positions dont la tete passe le pre-test et que la signature stricte rejette
	leursLibres      int // idem pour le repli
}

func (b *bilanDeComparaison) ajouter(o bilanDeComparaison) {
	b.payloads += o.payloads
	b.strictes += o.strictes
	b.libres += o.libres
	b.acceptees += o.acceptees
	b.leursStricts += o.leursStricts
	b.leursLibres += o.leursLibres
}

// comparerAuxReferences confronte les formes de production aux references sur `pay` : meme
// position rendue, et aucune position acceptee par une reference n est ecartee par le pre-test.
func comparerAuxReferences(t *testing.T, pay []byte, w *World, cfg FrameConfig) bilanDeComparaison {
	t.Helper()
	b := bilanDeComparaison{payloads: 1}
	if got, want := marchLocateStrict(pay, w, cfg), marchLocateStrictReference(pay, w, cfg); got != want {
		t.Fatalf("signature stricte : %d, reference %d", got, want)
	} else if want >= 0 {
		b.strictes++
	}
	if got, want := marchLocateFallback(pay, w, cfg), marchLocateFallbackReference(pay, w, cfg); got != want {
		t.Fatalf("repli a largeur libre : %d, reference %d", got, want)
	} else if want >= 0 {
		b.libres++
	}
	nb, largeur := len(pay)*8, largeurDeSignature(cfg)
	br := LecteurSur(pay)
	for s := 2; s+resteMinimalALargeurLibre < nb; s++ {
		if !precedeDuTerminateur(pay, s) {
			continue
		}
		id, tete := teteDeLaSignature(br, s, cfg)
		rec, _, ok := TryDeltaAt(pay, s, w, cfg)
		libre := ok && rec.Slot == marchSignatureSlot &&
			w.GenerationMatches(rec.ID, cfg.Profil.Grammaire.GenerationStricte)
		strict := s+largeur < nb && marchSignature123(pay, s, largeur, w, cfg)
		if libre || strict {
			b.acceptees++
			if !tete {
				t.Fatalf("position %d acceptee par la reference, ecartee par le pre-test", s)
			}
			if ok && id != rec.ID {
				t.Fatalf("position %d : identifiant du pre-test %#x, de l essai %#x", s, id, rec.ID)
			}
		}
		if tete && !strict && s+largeur < nb {
			b.leursStricts++
		}
		if tete && !libre {
			b.leursLibres++
		}
	}
	return b
}

// TestLePretestDeTeteNeChangePasLeLocalisateur : sur la bobine et sur des payloads construits
// depuis ses signatures, les formes de production rendent la position de la reference, et le
// pre-test n ecarte aucune position acceptee. Le test compte les acceptations exercees : il echoue
// s il ne compare que des rejets.
func TestLePretestDeTeteNeChangePasLeLocalisateur(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 11))
	var reels, construits bilanDeComparaison
	graines := 0
	parcourirPaquetsAEvenements(t, func(p paquetDeLocalisation, cfg FrameConfig) {
		for _, stricte := range []bool{true, false} {
			cfg.Profil.Grammaire.GenerationStricte = stricte
			reels.ajouter(comparerAuxReferences(t, p.pay, p.w, cfg))
		}
		cfg.Profil.Grammaire.GenerationStricte = true
		largeur := largeurDeSignature(cfg)
		var g graineDeSignature
		s := marchLocateStrict(p.pay, p.w, cfg)
		if s >= 1 {
			g = graineDeSignature{bits: bitsDuPayload(p.pay, s-1, s+largeur), tete: largeur - 19, marge: 1}
		} else if s = marchLocateFallback(p.pay, p.w, cfg); s >= 1 {
			if _, fin, ok := TryDeltaAt(p.pay, s, p.w, cfg); ok && fin-s > largeur-19+4 {
				g = graineDeSignature{bits: bitsDuPayload(p.pay, s-1, fin), tete: largeur - 19, marge: 1}
			}
		}
		if g.bits == nil || graines >= 60 {
			return
		}
		graines++
		// Deux mondes : le slot lie a une generation inconnue (celui de la marche, ou le controle
		// de generation ne rejette rien), puis lie a la generation de la graine.
		for _, lie := range []bool{false, true} {
			snap := p.w.Snapshot()
			if lie {
				br := LecteurSur(p.pay)
				br.SetBitPos(s)
				id, _, _ := lireLaTeteDeDelta(br, cfg)
				p.w.BindFull(id, p.w.slots[marchSignatureSlot].TypeIndex)
			}
			for _, stricte := range []bool{true, false} {
				cfg.Profil.Grammaire.GenerationStricte = stricte
				for _, pay := range variantes(r, g, cfg) {
					construits.ajouter(comparerAuxReferences(t, pay, p.w, cfg))
				}
			}
			p.w.Restore(snap)
		}
	})
	t.Logf("paquets reels : %+v", reels)
	t.Logf("payloads construits (%d graines tirees de la bobine) : %+v", graines, construits)
	if graines < 20 || construits.strictes < 100 || construits.libres < 100 || reels.strictes == 0 ||
		reels.libres == 0 || construits.leursStricts < 50 || construits.leursLibres < 50 {
		t.Fatalf("le test n exerce pas assez d acceptations ou de leurres : graines %d, reels %+v, construits %+v",
			graines, reels, construits)
	}
}
