package depottest

// balayages_test.go — AUCUNE BOUCLE DE BALAYAGE BIT A BIT NEUVE HORS DE LA LISTE DATEE.
//
// La marche des trames est la seule lecture d'un paquet delta : chaque canal lit ce qu'elle a
// produit, et un balayage (une boucle qui avance d'une position de bit et essaie un motif a chaque
// position) ne tourne plus que derriere elle, sur un paquet qu'elle n'a pas ferme, comme recuperation
// comptee. Un balayage neuf sur le chemin de production referait un parcours bit a bit que la marche
// evite : le resultat serait le meme, aucun test de donnees ne le verrait, la cuisson ralentirait.
//
// Le test releve, par l'AST des sources de production du module (tests et `research/` exclus),
// chaque boucle `for` dont le pas est `v++` (ou `v += 1`) et dont le corps passe `v`, dans une
// expression, a un appel qui recoit aussi une tranche d'octets parametre de la fonction englobante :
// la forme de tous les balayages connus (`for p := 0; p <= limite; p++ { lire(pay, p) }`). Il exige
// que l'ensemble des fonctions qui en portent soit EXACTEMENT [balayagesNommes] : un balayage neuf
// rougit, un balayage retire aussi (la liste ne fait que diminuer). Une boucle de la meme forme qui
// ne balaye pas un payload (parcours d'une liste de records, de candidats) y figure avec sa raison.
//
// LIMITE : une boucle qui lit par un lecteur construit hors de l'appel (`r.At(p)`, sans la tranche
// en argument) ou un `range` sur un entier echappe au releve.
//
// MUTATION — `for p := 0; p < len(pay)*8; p++ { n += int(source.BitsStricts(pay, p, 8)) }` ajoute a une fonction
// de `internal/grammar/camo_state.go` : ROUGE.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// balayagesNommes : LA LISTE DATEE (2026-11-14). Cle : chemin relatif a la racine du module + « : »
// + fonction englobante (« Type.methode » pour une methode). Valeur : ce que la boucle balaye et
// pourquoi elle reste.
var balayagesNommes = map[string]string{
	// Paquet delta, derriere la marche : la passe ne juge que ce que la marche n'a pas prouve
	// (fin bornee par la fermeture de la trame, `rendParLAncrage`).
	"internal/grammar/biped_creation.go:bipedCreationWalk.scanPayloadJusqua": "creations de bipede, derriere la marche",
	"internal/grammar/creations_du_monde.go:passeDesCreations.payload":       "creations d'objets du monde, derriere la marche",
	"internal/grammar/grenade_events.go:scanGrenadeThrowsJusqua":             "lancers de grenade, derriere la marche",
	"internal/grammar/pistes_du_monde.go:echantillonsDesBandes":              "pistes des objets du monde, derriere la marche",
	"internal/grammar/offline_aim_only.go:scanBipedAimRecordsJusqua":         "visee seule du bipede, derriere la marche",
	"internal/grammar/delta_biped_walk.go:walkDeltaBipedPayloadJusqua": "ancrage des records de bipede : lit un record " +
		"et saute, n'avance d'un bit qu'apres un rejet ; borne par la fermeture de la trame",
	// Paquet delta, seulement sur un paquet que la lecture ne ferme pas (recuperations comptees).
	"internal/grammar/debut_de_liste.go:candidatsDeTete": "debut de liste d'un paquet non ferme " +
		"(`repli_debut_de_liste_ferme_au_bit`)",
	"internal/grammar/localisateur.go:marchLocateStrict": "signature du slot 123, paquet dont la vue B " +
		"n'est pas fermee",
	"internal/grammar/localisateur.go:marchLocateFallback": "localisation a largeur libre, paquet non ferme " +
		"(`repli_localisation_largeur_libre`)",
	"internal/grammar/kills_rattrapes.go:killsAvecArrets": "message de kill hors de la vue A " +
		"(`repli_kill_rattrape_hors_vue_a`), fenetre bornee",
	// Paquet delta, encore en pleine passe : a passer derriere la marche (sonde et recuperation
	// d'equipement, puis la voie de l'hote).
	"internal/grammar/equipment_creation_width.go:mppCalibProbe.scanPayload": "sonde de calibration des " +
		"creations d'equipement (passe pleine, cout mesure 1,5 % d'une cuisson)",
	"internal/grammar/equipment_recovery.go:scanEquipRecoveryPacket": "recuperation des records d'equipement " +
		"du bipede (passe pleine sur les paquets de sa fenetre)",
	"internal/grammar/equipment_state.go:equipmentWalk.scanPayload":           "etat d'equipement, voie de l'hote",
	"internal/grammar/etats_de_mort_balayes.go:balayerUnPayload":              "gabarits d'etat de mort, voie de l'hote",
	"internal/grammar/navpoint_radial_scan.go:navpointRadialWalk.scanPayload": "points de navigation, voie de l'hote",
	"internal/grammar/objective_scan.go:objectiveWalk.scanPayload":            "objectifs, voie de l'hote",
	"internal/grammar/zone_state_scan.go:managedPropertyWalk.scanPayload":     "zones, voie de l'hote",
	"internal/grammar/projectiles.go:scanProjectileRecords":                   "trajectoires de projectile, voie de l'hote",
	"internal/grammar/i0_layout.go:collectI0Samples":                          "detecteur de la mise en page de i0, voie de l'hote",
	// Paquet delta, dans les bornes d'un record ou d'une trame deja localises.
	"internal/grammar/frame_closure_temoins.go:temoinsDecales":   "temoins de fermeture a quelques decalages de la vue C",
	"internal/grammar/inventory_ammo_rules.go:invSolveAmmoBlock": "bloc de munitions, dans la fenetre du record d'inventaire",
	"internal/grammar/inventory_decode.go:invAbilityIn":          "ancre de capacite, dans la fenetre du record d'inventaire",
	"internal/grammar/inventory_decode.go:invFirstFamily":        "premiere famille d'arme, dans la fenetre du record d'inventaire",
	"internal/grammar/inventory_grenade_selection.go:invLastFamily": "derniere famille d'arme, dans la fenetre du " +
		"record d'inventaire",
	"internal/grammar/inventory_grenades_rules.go:invGrenadesAfter":    "grenades portees, dans la fenetre du record d'inventaire",
	"internal/grammar/inventory_grenades_rules.go:invGrenadesNearAmmo": "grenades portees, dans la fenetre du record d'inventaire",
	// Paquet delta, atteint par les seuls tests (aucun appelant de production).
	"internal/grammar/frame_harvest.go:scanForTargetDelta": "resynchronisation de `DecodeFrameResync`, sans appelant de production",
	"internal/grammar/frame_infer.go:validatedResync":      "resynchronisation, active par `inferResyncTargets` que seuls les tests posent",
	// Hors paquet delta.
	"internal/grammar/bot_metadata.go:balayerLesEntreesDeBots":        "paquet BOT_METADATA",
	"internal/grammar/highlight_events.go:findBitMarker":              "chunk des temps forts",
	"internal/grammar/highlight_events.go:scanHighlightEvents":        "chunk des temps forts",
	"internal/grammar/player_entities_entetes.go:slotsDEntetesExacts": "payload d'image-cle",
	"internal/grammar/positions/positions.go:combAt":                  "paquet TYPE_2 plein etat",
	"internal/grammar/positions/positions.go:findCombs":               "paquet TYPE_2 plein etat",
	"internal/grammar/signaux/pied_de_film.go:decodeTh10Block":        "chunk de pied de film",
	"internal/grammar/signaux/pied_de_film.go:scanTh10Events":         "chunk de pied de film",
	"internal/grammar/signaux/statborg.go:scanFrameAvecReplis":        "trames du statborg",
	"internal/grammar/weaponscan/scanner.go:ScanFireEventsB5":         "marqueur universel des tirs, outil de la facade hors cuisson",
}

func TestAucunBalayageNeufHorsDeLaListe(t *testing.T) {
	racine := racineDuModule(t)
	vus := balayagesDuModule(t, racine)
	var neufs, retires []string
	for cle := range vus {
		if _, ok := balayagesNommes[cle]; !ok {
			neufs = append(neufs, cle)
		}
	}
	for cle := range balayagesNommes {
		if !vus[cle] {
			retires = append(retires, cle)
		}
	}
	sort.Strings(neufs)
	sort.Strings(retires)
	for _, c := range neufs {
		t.Errorf("balayage bit a bit hors de la liste : %s (un canal lit ce que la marche a produit ; "+
			"une recuperation derriere elle s'inscrit dans balayagesNommes avec sa raison)", c)
	}
	for _, c := range retires {
		t.Errorf("balayage de la liste disparu : %s (retirer son entree de balayagesNommes)", c)
	}
}

// balayagesDuModule rend les cles (`chemin:fonction`) des fonctions de production qui portent une
// boucle de balayage.
func balayagesDuModule(t *testing.T, racine string) map[string]bool {
	t.Helper()
	vus := map[string]bool{}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(racine, chemin)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "research" || strings.HasPrefix(rel, "research/") || d.Name() == "testdata" ||
				(strings.HasPrefix(d.Name(), ".") && rel != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), chemin, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if porteUnBalayage(fn) {
				vus[rel+":"+nomDeFonction(fn)] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("releve des sources : %v", err)
	}
	return vus
}

// nomDeFonction rend `fonction` ou `Type.methode`.
func nomDeFonction(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	typ := fn.Recv.List[0].Type
	if etoile, ok := typ.(*ast.StarExpr); ok {
		typ = etoile.X
	}
	if gen, ok := typ.(*ast.IndexExpr); ok {
		typ = gen.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// porteUnBalayage dit si `fn` contient une boucle de balayage : pas `v++`, et un appel du corps qui
// recoit `v` dans une expression et une tranche d'octets parametre de `fn`.
func porteUnBalayage(fn *ast.FuncDecl) bool {
	octets := parametresOctets(fn)
	if len(octets) == 0 {
		return false
	}
	trouve := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		boucle, ok := n.(*ast.ForStmt)
		if !ok || trouve {
			return !trouve
		}
		v := variableDuPas(boucle.Post)
		if v == "" && boucle.Post == nil {
			v = pasDansLeCorps(boucle)
		}
		if v == "" {
			return true
		}
		ast.Inspect(boucle.Body, func(m ast.Node) bool {
			appel, ok := m.(*ast.CallExpr)
			if !ok || trouve {
				return !trouve
			}
			avecV, avecOctets := false, false
			for _, arg := range appel.Args {
				if contientIdent(arg, v) {
					avecV = true
				}
				if id, ok := arg.(*ast.Ident); ok && octets[id.Name] {
					avecOctets = true
				}
			}
			trouve = avecV && avecOctets
			return true
		})
		return true
	})
	return trouve
}

// parametresOctets rend les noms des parametres de `fn` de type `[]byte`.
func parametresOctets(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	for _, champ := range fn.Type.Params.List {
		tab, ok := champ.Type.(*ast.ArrayType)
		if !ok || tab.Len != nil {
			continue
		}
		if id, ok := tab.Elt.(*ast.Ident); !ok || (id.Name != "byte" && id.Name != "uint8") {
			continue
		}
		for _, nom := range champ.Names {
			out[nom.Name] = true
		}
	}
	return out
}

// variableDuPas rend `v` pour un pas `v++` ou `v += 1`, sinon "".
func variableDuPas(post ast.Stmt) string {
	switch s := post.(type) {
	case *ast.IncDecStmt:
		if id, ok := s.X.(*ast.Ident); ok && s.Tok == token.INC {
			return id.Name
		}
	case *ast.AssignStmt:
		if s.Tok != token.ADD_ASSIGN || len(s.Lhs) != 1 || len(s.Rhs) != 1 {
			return ""
		}
		lit, ok := s.Rhs[0].(*ast.BasicLit)
		if id, okID := s.Lhs[0].(*ast.Ident); okID && ok && lit.Value == "1" {
			return id.Name
		}
	}
	return ""
}

// pasDansLeCorps rend `v` pour une boucle sans pas `for v := ...; cond; {` dont le corps avance
// `v` d'une position par `v++` (le pas d'un rejet, la lecture d'un record sautant plus loin), sinon "".
func pasDansLeCorps(boucle *ast.ForStmt) string {
	init, ok := boucle.Init.(*ast.AssignStmt)
	if !ok || init.Tok != token.DEFINE || len(init.Lhs) != 1 {
		return ""
	}
	id, ok := init.Lhs[0].(*ast.Ident)
	if !ok {
		return ""
	}
	avance := false
	ast.Inspect(boucle.Body, func(n ast.Node) bool {
		if variableDuPas(asStmt(n)) == id.Name {
			avance = true
		}
		return !avance
	})
	if !avance {
		return ""
	}
	return id.Name
}

// asStmt rend `n` s'il est une instruction, sinon nil.
func asStmt(n ast.Node) ast.Stmt {
	s, _ := n.(ast.Stmt)
	return s
}

// contientIdent dit si l'expression `e` mentionne l'identifiant `nom`.
func contientIdent(e ast.Expr, nom string) bool {
	vu := false
	ast.Inspect(e, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == nom {
			vu = true
		}
		return !vu
	})
	return vu
}
