package grammar

// film_context_voie_test.go — LE CONTEXTE DE VOIE : ce qu'il partage, ce qu'il possede, ce qu'il verse.

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/constat"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// entreeDeClassement dit le sort d'un champ du contexte dans une voie.
//
//	chauffe : partage en lecture ; `temoin` dit qu'il est prepare (la voie en herite)
//	prive   : propre a la voie ; `isole` dit que la voie n'a rien du parent
//	autre   : immuable (partage tel quel) ou ecarte (nul dans la voie) ; rien a prouver ici
type entreeDeClassement struct {
	classe string
	temoin func(c *FilmContext) bool
	isole  func(parent, voie *FilmContext) bool
}

func chauffe(t func(c *FilmContext) bool) entreeDeClassement {
	return entreeDeClassement{classe: "chauffe", temoin: t}
}

func prive(f func(parent, voie *FilmContext) bool) entreeDeClassement {
	return entreeDeClassement{classe: "prive", isole: f}
}

// classementDuContexte classe CHAQUE champ de [FilmContext], et chaque sous-champ de `recup`, `vies`,
// `duFilm` et `replis` (leur nom est « champ.sous-champ »). C'est le garde-rail de la brique : un champ
// ajoute au contexte sans entree ici fait echouer [TestChaufferRendTousLesMemosLus].
func classementDuContexte() map[string]entreeDeClassement {
	derive := func(c *FilmContext) bool { return c.duFilm.derive && (c.duFilm.lue || c.duFilm.compte) }
	creations := func(c *FilmContext) bool { return c.vies.creationsLues }
	autre := entreeDeClassement{classe: classeAutre}
	return map[string]entreeDeClassement{
		"film": autre, "impose": autre, "fils": autre,
		"chunks":    chauffe(func(c *FilmContext) bool { return c.chunksLus }),
		"chunksLus": chauffe(func(c *FilmContext) bool { return c.chunksLus }),
		"slots":     chauffe(func(c *FilmContext) bool { return c.slotsLus }),
		"slotsLus":  chauffe(func(c *FilmContext) bool { return c.slotsLus }),
		"lay":       chauffe(func(c *FilmContext) bool { return c.layLu }),
		"layErr":    chauffe(func(c *FilmContext) bool { return c.layLu }),
		"layLu":     chauffe(func(c *FilmContext) bool { return c.layLu }),
		"reg":       chauffe(func(c *FilmContext) bool { return c.regLu }),
		"regErr":    chauffe(func(c *FilmContext) bool { return c.regLu }),
		"regLu":     chauffe(func(c *FilmContext) bool { return c.regLu }),
		"prof":      chauffe(func(c *FilmContext) bool { return c.profLu }),
		"profLu":    chauffe(func(c *FilmContext) bool { return c.profLu }),
		"marches":   chauffe(func(c *FilmContext) bool { return c.marches != nil }),
		"bal": prive(func(p, v *FilmContext) bool {
			p.PoserMPP(profile.MPPWidths{Lead: 11, Index: 2})
			return v.bal.MPP != p.bal.MPP
		}),
		"obs":  prive(func(p, v *FilmContext) bool { return p.obs != nil && v.obs != nil && p.obs != v.obs }),
		"diag": prive(func(p, v *FilmContext) bool { return &p.diag != &v.diag && len(v.diag.Relever()) == 0 }),
		"voie": prive(func(p, v *FilmContext) bool { return v.voie != nil && v.voie.parent == p && p.voie == nil }),

		"duFilm.controle": chauffe(derive), "duFilm.moteurUn": chauffe(derive), "duFilm.malleable": chauffe(derive),
		"duFilm.creation": chauffe(derive), "duFilm.vueA": chauffe(derive), "duFilm.lue": chauffe(derive),
		"duFilm.derive": chauffe(derive), "duFilm.compte": chauffe(derive),

		"vies.creations": chauffe(creations), "vies.stats": chauffe(creations), "vies.err": chauffe(creations),
		"vies.creationsLues":      chauffe(creations),
		"vies.vivantes":           chauffe(func(c *FilmContext) bool { return c.vies.vivantes != nil }),
		"vies.enMarche":           chauffe(func(c *FilmContext) bool { return !c.vies.enMarche }),
		"vies.desImagesCles":      chauffe(func(c *FilmContext) bool { return c.vies.imagesClesRelevees }),
		"vies.imagesClesRelevees": chauffe(func(c *FilmContext) bool { return c.vies.imagesClesRelevees }),

		"replis.comptes": prive(func(_, v *FilmContext) bool { return v.replis.comptes == ComptesDesReplis{} }),
		"replis.i0Note":  prive(func(p, v *FilmContext) bool { return v.replis.i0Note == p.replis.i0Note }),
		"replis.regNote": prive(func(p, v *FilmContext) bool { return v.replis.regNote == p.replis.regNote }),

		"recup.ancres":          chauffe(func(c *FilmContext) bool { return c.recup.ancres != nil }),
		"recup.objets":          chauffe(func(c *FilmContext) bool { return c.recup.objets != nil }),
		"recup.lectures":        chauffe(func(c *FilmContext) bool { return c.recup.lectures != nil }),
		"recup.equipementPorte": chauffe(func(c *FilmContext) bool { return c.recup.equipementPorte != nil }),
		"recup.tables":          chauffe(func(c *FilmContext) bool { return c.tablesDesImagesClesChaudes() }),
		// Ecrit par le seul pre-passage, jamais par un lecteur : vide, les lecteurs calculent sans ranger.
		"recup.entetes": autre,
		// Se consomme (`rendue`, `prise`) : reste au parent, nul dans la voie.
		"recup.marche":    autre,
		"recup.pistes":    prive(func(_, v *FilmContext) bool { return v.recup.pistes == nil }),
		"recup.creations": prive(func(_, v *FilmContext) bool { return v.recup.creations == nil }),
	}
}

// champsDuContexte rend les noms que le classement doit couvrir, lus par reflexion.
func champsDuContexte() []string {
	var out []string
	t := reflect.TypeOf(FilmContext{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		switch f.Name {
		case "recup", "vies", "duFilm", "replis":
			for j := 0; j < f.Type.NumField(); j++ {
				out = append(out, f.Name+"."+f.Type.Field(j).Name)
			}
		default:
			out = append(out, f.Name)
		}
	}
	return out
}

// contexteChauffe ouvre un contexte sur la bobine `nom`, y fait tourner de quoi remplir les memos de
// la marche, et le chauffe. `fils` regle le pre-passage.
func contexteChauffe(t *testing.T, nom string, fils int) *FilmContext {
	t.Helper()
	fc := NewFilmContext(chargerUneBobine(t, nom))
	fc.PoserFils(fils)
	if _, _, err := fc.objetsDuMonde(); err != nil {
		t.Fatalf("objets du monde de %s : %v", nom, err)
	}
	if _, _, err := ScanEquipementPorte(fc, nil); err != nil {
		t.Fatalf("equipement porte de %s : %v", nom, err)
	}
	fc.Chauffer()
	return fc
}

// TestChaufferRendTousLesMemosLus : le classement couvre exactement les champs du contexte ; apres
// Chauffer, le temoin de chaque champ chauffe est vrai, chez le parent comme chez la voie ; chaque champ
// prive est isole dans la voie.
func TestChaufferRendTousLesMemosLus(t *testing.T) {
	classement := classementDuContexte()
	vus := map[string]bool{}
	for _, nom := range champsDuContexte() {
		vus[nom] = true
		if _, ok := classement[nom]; !ok {
			t.Errorf("champ %q du contexte sans classement : chauffe, prive a la voie, immuable ou ecarte ?", nom)
		}
	}
	for nom := range classement {
		if !vus[nom] {
			t.Errorf("classement de %q : ce champ n'existe plus", nom)
		}
	}
	for _, nom := range bobinesAEmpreinte {
		for _, fils := range []int{1, 4} {
			fc := contexteChauffe(t, nom, fils)
			if !fc.EstChaud() {
				t.Fatalf("%s a %d fils : EstChaud faux apres Chauffer", nom, fils)
			}
			voie := fc.Deriver()
			for champ, e := range classement {
				switch e.classe {
				case "chauffe":
					if !e.temoin(fc) || !e.temoin(voie) {
						t.Errorf("%s a %d fils : %q n'est pas chaud (parent %v, voie %v)", nom, fils, champ,
							e.temoin(fc), e.temoin(voie))
					}
				case "prive":
					if !e.isole(fc, voie) {
						t.Errorf("%s a %d fils : %q n'est pas isole dans la voie", nom, fils, champ)
					}
				}
			}
			if !voie.EstChaud() {
				t.Errorf("%s a %d fils : la voie d'un contexte chaud n'est pas chaude", nom, fils)
			}
			if voie.Fils() != 1 {
				t.Errorf("%s : une voie tourne a 1 fil, %d lu", nom, voie.Fils())
			}
		}
	}
}

// TestChaufferNeLanceAucuneMarche : sur un contexte dont la marche n'a pas tourne, Chauffer prepare ce
// qui ne la demande pas, laisse vides les memos qu'elle remplit, et EstChaud reste faux.
func TestChaufferNeLanceAucuneMarche(t *testing.T) {
	fc := NewFilmContext(chargerUneBobine(t, bobinesAEmpreinte[0]))
	if fc.EstChaud() {
		t.Fatal("contexte neuf : EstChaud vrai")
	}
	fc.Chauffer()
	if fc.recup.lectures != nil || fc.recup.objets != nil || fc.recup.equipementPorte != nil || fc.vies.vivantes != nil {
		t.Fatal("Chauffer a rempli un memo de la marche : il a lance une marche")
	}
	if !fc.memosDeBaseChauds() {
		t.Fatal("les memos qui ne dependent que du film ne sont pas prets")
	}
	if fc.EstChaud() {
		t.Fatal("EstChaud vrai sans marche")
	}
	if (*FilmContext)(nil).EstChaud() {
		t.Fatal("contexte nil : EstChaud vrai")
	}
}

// TestDeriverNePartagePasLeProfil : le profil de balayage, les largeurs MPP, l'observation et les
// diagnostics d'une voie ne sont pas ceux du parent, dans les deux sens.
func TestDeriverNePartagePasLeProfil(t *testing.T) {
	parent := NewFilmContext(nil)
	base := profile.MPPWidths{Lead: 7, Index: 3}
	parent.PoserMPP(base)
	voie := parent.Deriver()
	if voie.bal.MPP != base {
		t.Fatalf("la voie n'herite pas des largeurs du parent : %v", voie.bal.MPP)
	}
	prev := voie.PoserMPP(profile.MPPWidths{Lead: 9, Index: 5})
	if prev != base || parent.bal.MPP != base {
		t.Fatalf("PoserMPP sur la voie a touche le parent : precedent %v, parent %v", prev, parent.bal.MPP)
	}
	parent.PoserMPP(profile.MPPWidths{Lead: 1, Index: 1})
	if voie.bal.MPP != (profile.MPPWidths{Lead: 9, Index: 5}) {
		t.Fatalf("PoserMPP sur le parent a touche la voie : %v", voie.bal.MPP)
	}
	pb := voie.ProfilDeBalayage()
	pb.MPP = profile.MPPWidths{Lead: 2, Index: 2}
	voie.PoserProfilDeBalayage(pb)
	if parent.bal.MPP != (profile.MPPWidths{Lead: 1, Index: 1}) {
		t.Fatalf("PoserProfilDeBalayage sur la voie a touche le parent : %v", parent.bal.MPP)
	}
	if voie.Observation() == parent.Observation() {
		t.Fatal("la voie partage l'observation du parent")
	}
	voie.Diagnostics().Signaler(constat.Diagnostic{Code: "voie"})
	if len(parent.Diagnostics().Relever()) != 0 {
		t.Fatal("un diagnostic de la voie est apparu chez le parent avant la fusion")
	}
}

// TestVoieNEcritPasLeParent : deux voies qui posent des largeurs MPP en meme temps (sous -race) n'ecrivent
// ni le parent ni l'autre voie, et chacune relit ce qu'elle a pose.
func TestVoieNEcritPasLeParent(t *testing.T) {
	parent := NewFilmContext(nil)
	base := profile.MPPWidths{Lead: 7, Index: 3}
	parent.PoserMPP(base)
	voies := [2]*FilmContext{parent.Deriver(), parent.Deriver()}
	var ecarts atomic.Int32
	executerEnParallele(2, 2, func(i int) {
		w := profile.MPPWidths{Lead: 20 + i, Index: 30 + i}
		for k := 0; k < 2000; k++ {
			prev := voies[i].PoserMPP(w)
			if voies[i].bal.MPP != w || prev != base {
				ecarts.Add(1)
			}
			voies[i].PoserMPP(base)
			voies[i].NoterReplis(ComptesDesReplis{LargeursMPPParDefaut: 1})
			voies[i].Diagnostics().Signaler(constat.Diagnostic{Code: "voie"})
		}
	})
	if ecarts.Load() != 0 {
		t.Fatalf("%d lectures de largeurs inattendues dans les voies", ecarts.Load())
	}
	if parent.bal.MPP != base || parent.ComptesDesReplis() != (ComptesDesReplis{}) {
		t.Fatalf("une voie a ecrit le parent : %v, %+v", parent.bal.MPP, parent.ComptesDesReplis())
	}
}

// TestFusionnerSommeLesReplis : les comptes des voies s'ajoutent a ceux du parent, champ a champ, et les
// diagnostics suivent ceux du parent dans l'ordre des fusions.
func TestFusionnerSommeLesReplis(t *testing.T) {
	parent := NewFilmContext(nil)
	parent.NoterReplis(ComptesDesReplis{SlotsBipedesComblees: 2})
	parent.Diagnostics().Signaler(constat.Diagnostic{Code: "parent"})
	a, b := parent.Deriver(), parent.Deriver()
	a.NoterReplis(ComptesDesReplis{SlotsBipedesComblees: 3, LargeursMPPParDefaut: 1})
	b.NoterReplis(ComptesDesReplis{SlotsBipedesComblees: 5, AncresSansVieDelta: 4})
	a.Diagnostics().Signaler(constat.Diagnostic{Code: "a"})
	b.Diagnostics().Signaler(constat.Diagnostic{Code: "b"})
	parent.Fusionner(a)
	parent.Fusionner(b)
	voulu := ComptesDesReplis{SlotsBipedesComblees: 10, LargeursMPPParDefaut: 1, AncresSansVieDelta: 4}
	if got := parent.ComptesDesReplis(); got != voulu {
		t.Fatalf("replis fusionnes %+v, %+v attendus", got, voulu)
	}
	var codes []string
	for _, d := range parent.Diagnostics().Relever() {
		codes = append(codes, string(d.Code))
	}
	if !reflect.DeepEqual(codes, []string{"parent", "a", "b"}) {
		t.Fatalf("diagnostics fusionnes %v, [parent a b] attendus", codes)
	}
}

// compterTousLesReplisParFilm fait compter a `c` les trois replis « une fois par film », comme le
// contexte le fait lui-meme.
func compterTousLesReplisParFilm(c *FilmContext) {
	c.noterI0ParDefaut(nil)
	c.noterRegistre(&Registry{})
	if !c.duFilm.compte {
		c.duFilm.compte = true
		c.NoterReplis(ComptesDesReplis{ControleDeCorruptionNonDeclare: 1})
	}
}

func unParFilm() ComptesDesReplis {
	return ComptesDesReplis{I0PorteEtRegionParDefaut: 1, RegistreInconnu: 1, ControleDeCorruptionNonDeclare: 1}
}

// TestFusionnerNeCompteJamaisDeuxFoisUnReplisParFilm : que le repli soit compte par une voie, par deux, par
// le parent avant ou apres la derivation, le film le porte UNE fois. Une voie n'est fusionnee qu'une fois.
func TestFusionnerNeCompteJamaisDeuxFoisUnReplisParFilm(t *testing.T) {
	cas := map[string]func(parent *FilmContext){
		"deux voies": func(parent *FilmContext) {
			a, b := parent.Deriver(), parent.Deriver()
			compterTousLesReplisParFilm(a)
			compterTousLesReplisParFilm(b)
			parent.Fusionner(a)
			parent.Fusionner(b)
		},
		"parent avant la derivation": func(parent *FilmContext) {
			compterTousLesReplisParFilm(parent)
			a := parent.Deriver()
			compterTousLesReplisParFilm(a)
			parent.Fusionner(a)
		},
		"parent apres la derivation": func(parent *FilmContext) {
			a := parent.Deriver()
			compterTousLesReplisParFilm(a)
			compterTousLesReplisParFilm(parent)
			parent.Fusionner(a)
		},
		"parent apres la fusion": func(parent *FilmContext) {
			a := parent.Deriver()
			compterTousLesReplisParFilm(a)
			parent.Fusionner(a)
			compterTousLesReplisParFilm(parent)
		},
	}
	for nom, scenario := range cas {
		parent := NewFilmContext(nil)
		scenario(parent)
		if got := parent.ComptesDesReplis(); got != unParFilm() {
			t.Errorf("%s : replis %+v, %+v attendus", nom, got, unParFilm())
		}
	}
	parent := NewFilmContext(nil)
	voie := parent.Deriver()
	parent.Fusionner(voie)
	for nom, f := range map[string]func(){
		"seconde fusion":      func() { parent.Fusionner(voie) },
		"contexte quelconque": func() { parent.Fusionner(NewFilmContext(nil)) },
		"voie d'un autre":     func() { parent.Fusionner(NewFilmContext(nil).Deriver()) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s : pas de panique", nom)
				}
			}()
			f()
		}()
	}
}

// TestFusionnerReprendLesMemosDuMonde : les pistes et creations d'une voie rejoignent le parent, sans
// doublon de cle.
func TestFusionnerReprendLesMemosDuMonde(t *testing.T) {
	parent := NewFilmContext(nil)
	voie := parent.Deriver()
	voie.recup.pistes = []*pistesRelevees{{slots: []uint32{1, 2}}, {slots: []uint32{3}}}
	voie.recup.creations = []*creationsRelevees{{cle: cleDeCreation{ti: 7}}}
	parent.recup.pistes = []*pistesRelevees{{slots: []uint32{3}}}
	parent.Fusionner(voie)
	if len(parent.recup.pistes) != 2 || len(parent.recup.creations) != 1 {
		t.Fatalf("pistes %d (2 attendues), creations %d (1 attendue)", len(parent.recup.pistes), len(parent.recup.creations))
	}
}
