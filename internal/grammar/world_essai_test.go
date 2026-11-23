package grammar

// world_essai_test.go — LES ESSAIS DU MONDE RENDENT LE MONDE EXACTEMENT : mondes construits, mutants,
// garde des sources, mini-bobine. Le harnais est dans `world_essai_harnais_test.go`.

import (
	"math/rand/v2"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

const (
	premierSlotDEssai = 500
	slotsDEssai       = 8 // peu de slots : les ecritures se rencontrent, creations, ecrasements et suppressions
)

// operationAleatoire rend une ecriture du monde (ou d un champ que ni l une ni l autre methode ne
// restaure) a arguments tires une fois : la meme sequence se rejoue a l identique sur deux mondes.
func operationAleatoire(r *rand.Rand) func(*World) {
	slot := uint32(premierSlotDEssai + r.IntN(slotsDEssai))
	id := slot | uint32(r.IntN(4))<<30
	ti := uint32(r.IntN(40))
	switch r.IntN(14) {
	case 0:
		return func(w *World) { w.BindFull(id, ti) }
	case 1:
		return func(w *World) { w.BindWildcard(slot, ti) }
	case 2:
		ns := uint32(r.IntN(4))
		return func(w *World) { w.BindImageCle(ns, slot, ti) }
	case 3:
		return func(w *World) { w.BindEntreeDeTable(id, ti) }
	case 4:
		return func(w *World) { w.lierCommeDatum(slot, ti, lecture.LiaisonAnticipation) }
	case 5:
		return func(w *World) { w.BindSoft(id, ti) }
	case 6:
		return func(w *World) { w.Unbind(slot) }
	case 7:
		portes := map[uint32]bool{}
		for range 1 + r.IntN(slotsDEssai) {
			portes[uint32(premierSlotDEssai+r.IntN(slotsDEssai))] = true
		}
		return func(w *World) { w.OublierLesSlotsNonPortes(portes) }
	case 8, 9:
		v := [3]float32{r.Float32(), r.Float32(), r.Float32()}
		return func(w *World) { w.SetPos(slot, v) }
	case 10:
		// Un Restore COMPLET au milieu d un essai : la table est remplacee, le journal doit l annuler.
		ref := NewWorld(nil)
		for range r.IntN(slotsDEssai) {
			ref.BindFull(uint32(premierSlotDEssai+r.IntN(slotsDEssai)), uint32(r.IntN(40)))
		}
		return func(w *World) { w.Restore(ref.Snapshot()) }
	case 11:
		v := r.IntN(3)
		return func(w *World) { w.PoserVueCourante(v) }
	case 12:
		pay := []byte{1, 2, 3}
		bit := r.IntN(20)
		return func(w *World) { w.marquerDebutNonProuve(pay, bit) }
	default:
		rec := &FrameRecord{ID: id, TypeIndex: ti, HeaderBit: r.IntN(100)}
		return func(w *World) { w.suspendreUnNeuf(rec) }
	}
}

func sequenceAleatoire(r *rand.Rand, n int) []func(*World) {
	ops := make([]func(*World), n)
	for i := range ops {
		ops[i] = operationAleatoire(r)
	}
	return ops
}

func monde8Slots(r *rand.Rand) *World {
	w := NewWorld(nil)
	for _, op := range sequenceAleatoire(r, r.IntN(2*slotsDEssai)) {
		op(w)
	}
	return w
}

func jouer(w *World, ops []func(*World)) {
	for _, op := range ops {
		op(w)
	}
}

// TestLeHarnaisCompareChaqueChampDuMonde : un champ ajoute a [World] sans l etre dans le harnais
// fait echouer ici, et non en silence dans une restauration fausse.
func TestLeHarnaisCompareChaqueChampDuMonde(t *testing.T) {
	compares := champsDuMonde(NewWorld(nil), NewWorld(nil))
	typ := reflect.TypeFor[World]()
	for i := range typ.NumField() {
		if _, ok := compares[typ.Field(i).Name]; !ok {
			t.Errorf("le champ %q de World n est pas compare par le harnais", typ.Field(i).Name)
		}
	}
	if len(compares) != typ.NumField() {
		t.Errorf("le harnais compare %d champs, World en porte %d", len(compares), typ.NumField())
	}
}

// TestEssaiSurMondesConstruits : des milliers de sequences d ecritures, a plat et emboitees, sur des
// mondes construits ; le monde restaure est celui de la reference, champ par champ.
func TestEssaiSurMondesConstruits(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 11))
	var b bilanDEssais
	for range 4000 {
		w := monde8Slots(r)
		ops, dedans, apres := sequenceAleatoire(r, r.IntN(14)), sequenceAleatoire(r, r.IntN(8)), sequenceAleatoire(r, r.IntN(6))
		b.exiger(t, w, "essai a plat", func(c *World, _ methodeDEssai) any { jouer(c, ops); return nil }, nil)
		b.exiger(t, w, "essais emboites", func(c *World, m methodeDEssai) any {
			jouer(c, ops)
			fermer := m.ouvrir(c)
			jouer(c, dedans)
			fermer()
			jouer(c, apres)
			return nil
		}, nil)
	}
	t.Logf("%d essais compares (monde complet), dont %d modifient la table avant la restauration", b.essais, b.modifiants)
	if b.modifiants < b.essais/2 {
		t.Fatalf("trop peu d essais ecrivent (%d sur %d) : le test ne garde rien", b.modifiants, b.essais)
	}
}

// TestChaqueSiteDEcritureEstJournalise : chaque ecriture de la table, sur un slot present, absent, et
// pour la suppression sur un slot absent, laisse le monde de la reference apres restauration.
func TestChaqueSiteDEcritureEstJournalise(t *testing.T) {
	sites := map[string]func(w *World, slot uint32){
		"BindFull":                 func(w *World, s uint32) { w.BindFull(s|1<<30, 3) },
		"BindWildcard":             func(w *World, s uint32) { w.BindWildcard(s, 4) },
		"BindImageCle":             func(w *World, s uint32) { w.BindImageCle(1, s, 5) },
		"BindEntreeDeTable":        func(w *World, s uint32) { w.BindEntreeDeTable(s|2<<30, 6) },
		"lierCommeDatum":           func(w *World, s uint32) { w.lierCommeDatum(s, 7, lecture.LiaisonAnticipation) },
		"BindSoft":                 func(w *World, s uint32) { w.BindSoft(s, 8) },
		"Unbind":                   func(w *World, s uint32) { w.Unbind(s) },
		"OublierLesSlotsNonPortes": func(w *World, _ uint32) { w.OublierLesSlotsNonPortes(map[uint32]bool{999: true}) },
		"SetPos":                   func(w *World, s uint32) { w.SetPos(s, [3]float32{1, 2, 3}) },
		"Restore":                  func(w *World, _ uint32) { w.Restore(WorldSnapshot{slots: map[uint32]slotState{9: {TypeIndex: 1}}}) },
	}
	var b bilanDEssais
	for nom, site := range sites {
		for _, slot := range []uint32{510, 520} { // 510 est lie dans le monde de depart, 520 ne l est pas
			w := NewWorld(nil)
			w.BindFull(510, 1)
			w.SetPos(510, [3]float32{9, 8, 7})
			w.BindSoft(511, 2)
			b.exiger(t, w, nom, func(c *World, _ methodeDEssai) any { site(c, slot); return nil }, nil)
		}
	}
	if b.modifiants == 0 {
		t.Fatal("aucun site n a ecrit")
	}
}

// TestLeHarnaisDetecteUneEcritureHorsJournal : un mutant, une ecriture de la table qui ne passe pas
// par le journal, est vu, quelle que soit sa forme, seule ou parmi des ecritures journalisees.
func TestLeHarnaisDetecteUneEcritureHorsJournal(t *testing.T) {
	mutants := map[string]func(w *World){
		"creation directe":    func(w *World) { w.slots[777] = slotState{TypeIndex: 9} },
		"ecrasement direct":   func(w *World) { s := w.slots[510]; s.TypeIndex++; w.slots[510] = s },
		"suppression directe": func(w *World) { delete(w.slots, 510) },
		"remplacement":        func(w *World) { w.slots = map[uint32]slotState{} },
	}
	for nom, mutant := range mutants {
		w := NewWorld(nil)
		w.BindFull(510, 1)
		var b bilanDEssais
		seul := func(c *World, _ methodeDEssai) any { mutant(c); return nil }
		if len(b.comparer(w, seul, nil)) == 0 {
			t.Errorf("mutant %q : non detecte", nom)
		}
		melange := func(c *World, _ methodeDEssai) any { c.BindSoft(511, 2); mutant(c); c.Unbind(511); return nil }
		if len(b.comparer(w, melange, nil)) == 0 {
			t.Errorf("mutant %q parmi des ecritures journalisees : non detecte", nom)
		}
	}
}

// TestLesEcrituresDeLaTableNePassentQueParLeJournal : garde des sources. Hors `world_essai.go`, aucun
// fichier de production n ecrit la table des slots d un monde, n en garde un alias, et seuls les
// fichiers listes la lisent.
func TestLesEcrituresDeLaTableNePassentQueParLeJournal(t *testing.T) {
	lecteurs := map[string]bool{"world.go": true, "world_essai.go": true, "frame_infer.go": true, "marche_trames_entites.go": true}
	monde := `(?:\b(?:w|monde)|\.(?:w|monde))\.slots\b`
	ecriture := regexp.MustCompile(monde + `\[[^\]]*\]\s*[-+*/|&^]?=[^=]|` + monde + `\s*(?:[-+*/|&^]?=[^=]|\+\+|--)|` +
		`\b(?:delete|clear)\(\s*` + monde + `|\bmaps\.\w+\(\s*` + monde + `|(?::=|=)\s*&?` + monde + `\s*$`)
	mention := regexp.MustCompile(monde)
	entrees, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	lus := 0
	for _, e := range entrees {
		nom := e.Name()
		if !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		brut, err := os.ReadFile(nom) //nolint:gosec // sources du paquet
		if err != nil {
			t.Fatal(err)
		}
		for i, l := range strings.Split(string(brut), "\n") {
			if c := strings.Index(l, "//"); c >= 0 {
				l = l[:c]
			}
			if !mention.MatchString(l) {
				continue
			}
			lus++
			if !lecteurs[nom] {
				t.Errorf("%s:%d : la table des slots est lue hors des fichiers listes : %s", nom, i+1, strings.TrimSpace(l))
			}
			if nom != "world_essai.go" && ecriture.MatchString(l) {
				t.Errorf("%s:%d : ecriture de la table hors du journal : %s", nom, i+1, strings.TrimSpace(l))
			}
		}
	}
	if lus == 0 {
		t.Fatal("la garde ne lit rien")
	}
}

// TestEssaiSurLaBobineReelle : sur la mini-bobine du depot, chaque essai que la production fait sur
// un paquet a evenements (lecture d essai, records de bipedes de la calibration, liste recuperee) est
// joue avec la reference et avec le journal, depuis le monde reel de la marche, aux positions que
// la production essaie et a un echantillon des autres. La fonction de production est aussi comparee.
func TestEssaiSurLaBobineReelle(t *testing.T) {
	var b bilanDEssais
	parcourirPaquetsAEvenements(t, func(p paquetDeLocalisation, cfg FrameConfig) {
		nb, extra := len(p.pay)*8, motFacultatifDEnTete(cfg)
		positions := map[int]bool{}
		for _, s := range []int{marchLocateStrict(p.pay, p.w, cfg), marchLocateFallback(p.pay, p.w, cfg)} {
			if s >= 0 {
				positions[s] = true
			}
		}
		if d, _ := localiserLaListe(p.pay, p.w, cfg); d >= 0 {
			positions[d] = true
		}
		for _, c := range candidatsDeTete(p.pay, nb, p.w) {
			if c-extra >= 0 {
				positions[c-extra] = true
			}
		}
		for s := 2; s < nb; s += 1 + nb/40 {
			if precedeDuTerminateur(p.pay, s) {
				positions[s] = true
			}
		}
		for s := range positions {
			b.exiger(t, p.w, "lecture d essai", corpsDeLectureDEssai(p.pay, cfg, s),
				func(w *World) any { return lectureDEssai(p.pay, w, cfg, s) })
			b.exiger(t, p.w, "liste recuperee", corpsDeListeRecuperee(p.pay, cfg, s), nil)
		}
		b.exiger(t, p.w, "records de bipedes", corpsDeRecordsBipedes(p.pay, cfg, 3),
			func(w *World) any { return recordsBipedesPropres(p.pay, w, cfg, 3) })
	})
	t.Logf("bobine : %d essais compares (monde complet), dont %d modifient la table avant la restauration",
		b.essais, b.modifiants)
	if b.essais == 0 || b.modifiants == 0 {
		t.Fatalf("la bobine n a donne aucun essai qui ecrive (%d essais, %d modifiants)", b.essais, b.modifiants)
	}
}
