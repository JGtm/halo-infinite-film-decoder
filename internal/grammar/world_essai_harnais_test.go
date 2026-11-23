package grammar

// world_essai_harnais_test.go — LE HARNAIS DIFFERENTIEL DES ESSAIS DU MONDE.
//
// Un essai rend le monde tel qu il etait a son ouverture. Deux methodes le font : la copie complete
// de la table des slots ([World.Snapshot] puis [World.Restore], la methode de reference) et le
// journal d annulation ([World.ouvrirUnEssai] puis [World.annulerLEssai]). Le harnais joue le MEME
// corps sur deux clones du meme monde, un par methode, et compare le monde ENTIER, champ par champ,
// une fois l essai clos. Il compte aussi les essais qui ont reellement modifie la table des slots
// avant la restauration : un essai qui n ecrit rien ne prouve rien.

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

// methodeDEssai : une facon de rendre le monde. `ouvrir` rend la fermeture de l essai.
type methodeDEssai struct {
	nom    string
	ouvrir func(w *World) func()
}

var (
	methodeAncienne = methodeDEssai{"copie complete", func(w *World) func() {
		snap := w.Snapshot()
		return func() { w.Restore(snap) }
	}}
	methodeNouvelle = methodeDEssai{"journal d annulation", func(w *World) func() {
		e := w.ouvrirUnEssai()
		return func() { w.annulerLEssai(e) }
	}}
)

// bilanDEssais : les comptes d une campagne de comparaisons.
type bilanDEssais struct {
	essais, modifiants int
}

// cloneDuMonde rend un monde independant de `w` : table copiee, NEW suspendus copies, journal vide.
func cloneDuMonde(w *World) *World {
	c := *w
	c.slots = w.cloneSlots()
	c.neufsSuspendus = slices.Clone(w.neufsSuspendus)
	c.journal, c.essaisOuverts = nil, 0
	return &c
}

// memeSlot compare deux liaisons, les positions par leurs bits (une position NaN est egale a elle-meme).
func memeSlot(a, b slotState) bool {
	for i := range a.Pos {
		if math.Float32bits(a.Pos[i]) != math.Float32bits(b.Pos[i]) {
			return false
		}
	}
	a.Pos, b.Pos = [3]float32{}, [3]float32{}
	return a == b
}

func memesSlots(a, b map[uint32]slotState) bool {
	if len(a) != len(b) {
		return false
	}
	for k, x := range a {
		y, ok := b[k]
		if !ok || !memeSlot(x, y) {
			return false
		}
	}
	return true
}

// champsDuMonde rend, pour CHAQUE champ de [World], s il est le meme dans `a` et `b`. Un champ
// ajoute au monde sans l etre ici fait echouer [TestLeHarnaisCompareChaqueChampDuMonde].
func champsDuMonde(a, b *World) map[string]bool {
	return map[string]bool{
		"Reg":              a.Reg == b.Reg,
		"slots":            memesSlots(a.slots, b.slots),
		"vueCourante":      a.vueCourante == b.vueCourante,
		"nsImageCle":       a.nsImageCle == b.nsImageCle,
		"anticipee":        a.anticipee == b.anticipee,
		"chunkCourant":     a.chunkCourant == b.chunkCourant,
		"anticipationDite": a.anticipationDite == b.anticipationDite,
		"debutNonProuve":   reflect.DeepEqual(a.debutNonProuve, b.debutNonProuve),
		"neufsSuspendus":   slices.Equal(a.neufsSuspendus, b.neufsSuspendus),
		"journal":          len(a.journal) == 0 && len(b.journal) == 0,
		"essaisOuverts":    a.essaisOuverts == 0 && b.essaisOuverts == 0,
	}
}

// ecartsDeMondes nomme les champs qui different entre `a` et `b`.
func ecartsDeMondes(a, b *World) []string {
	var out []string
	for nom, egal := range champsDuMonde(a, b) {
		if !egal {
			out = append(out, nom)
		}
	}
	slices.Sort(out)
	return out
}

// comparer joue `corps` sur deux clones de `w`, un par methode, et rend les ecarts. `production`, s il
// n est pas nil, est la fonction de production qui fait elle-meme son essai : jouee sur un troisieme
// clone, elle doit rendre le meme resultat et le meme monde que la reference.
func (b *bilanDEssais) comparer(w *World, corps func(*World, methodeDEssai) any, production func(*World) any) []string {
	avant := w.cloneSlots()
	jouer := func(m methodeDEssai) (*World, any, bool) {
		c := cloneDuMonde(w)
		fermer := m.ouvrir(c)
		r := corps(c, m)
		modifie := !memesSlots(c.slots, avant)
		fermer()
		return c, r, modifie
	}
	ancien, rAncien, modifieAncien := jouer(methodeAncienne)
	nouveau, rNouveau, modifieNouveau := jouer(methodeNouvelle)
	b.essais++
	if modifieAncien {
		b.modifiants++
	}
	ecarts := ecartsDeMondes(ancien, nouveau)
	if !memesSlots(nouveau.slots, avant) {
		ecarts = append(ecarts, "slots != etat de l ouverture")
	}
	if modifieAncien != modifieNouveau || !reflect.DeepEqual(rAncien, rNouveau) {
		ecarts = append(ecarts, "resultat ou modification differents")
	}
	if production != nil {
		c := cloneDuMonde(w)
		if rProd := production(c); !reflect.DeepEqual(rProd, rAncien) {
			ecarts = append(ecarts, "resultat de la production")
		}
		for _, e := range ecartsDeMondes(ancien, c) {
			ecarts = append(ecarts, "production : "+e)
		}
	}
	return ecarts
}

// exiger echoue au premier ecart.
func (b *bilanDEssais) exiger(t *testing.T, w *World, nom string, corps func(*World, methodeDEssai) any,
	production func(*World) any) {
	t.Helper()
	if e := b.comparer(w, corps, production); len(e) > 0 {
		t.Fatalf("%s : le monde restaure differe de la reference : %v", nom, e)
	}
}

// --- les corps d essai de production, sans leur gestion d essai ---------------------------------

// corpsDeLectureDEssai est le corps de [lectureDEssai].
func corpsDeLectureDEssai(pay []byte, cfg FrameConfig, debut int) func(*World, methodeDEssai) any {
	return func(w *World, _ methodeDEssai) any {
		essai := cfg
		obs := NouvelleObservation()
		var l LectureVueC
		obs.VueControleHook = func(x LectureVueC) { l = x }
		essai.Obs = obs
		DecodeFrameViewsCurseur(pay, w, essai, MovementStateViews, debut)
		return l
	}
}

// corpsDeRecordsBipedes est le corps de [recordsBipedesPropres].
func corpsDeRecordsBipedes(pay []byte, cadre FrameConfig, vues int) func(*World, methodeDEssai) any {
	return func(w *World, _ methodeDEssai) any {
		br := LecteurSur(pay)
		br.Skip(2)
		n := 0
		for v := 0; v < vues && len(pay)*8-br.BitPos() >= 8; v++ {
			recs, err := DecodeFrameRecords(br, w, cadre)
			for k := range recs {
				if recs[k].DesyncAt == -1 && recs[k].TypeIndex == BipedTypeIndex {
					n++
				}
			}
			if err != nil {
				break
			}
		}
		return n
	}
}

// corpsDeListeRecuperee est le corps de [MarcheDistribuee.lireLaListeRecuperee] depuis le bit `s`.
func corpsDeListeRecuperee(pay []byte, cfg FrameConfig, s int) func(*World, methodeDEssai) any {
	return func(w *World, _ methodeDEssai) any {
		c := cfg
		c.Obs = nil
		br := LecteurSur(pay)
		br.poserCadre(c)
		br.Skip(s)
		recs, _ := DecodeFrameRecords(br, w, c)
		return len(recs)
	}
}
