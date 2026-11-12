package grammar

// film_infos_de_match_limites_test.go — les vecteurs de bord des infos de match : messages Bond qui tombent,
// bornes des longueurs, tampon coupe, chemins, et les sept bobines (un film par build). Les constructeurs de
// messages sont ceux de `film_infos_de_match_test.go`.

import (
	"encoding/hex"
	"testing"
)

// avecUnChampCasse ecrit une variante valide (score 50) puis, a un champ de tete que la lecture ne suit pas,
// un champ que `casse` ecrit : un message dont seule cette partie est fausse.
func avecUnChampCasse(casse func(*bondEcrit)) bondEcrit {
	var b bondEcrit
	b.structure(func(t *bondEcrit) {
		t.listeDeStructures(1, uneVarianteEcrite{score: p32(50)}.ecrire)
		casse(t)
	})
	return b
}

// TestUneLectureQuiTombeNeRendRien : une longueur de structure fausse, un compte hors du tampon, un type
// inconnu, un message trop profond et un entier de plus de dix octets, a cote d une variante valide, ne
// rendent aucune valeur de variante, sans paniquer. Un tampon qui s arrete avant l octet de fin ne rend rien.
// MUTATIONS — la verification de l octet de fin, la borne des comptes, la profondeur maximale, la borne de
// l entier variable, la garde de debordement : ROUGES.
func TestUneLectureQuiTombeNeRendRien(t *testing.T) {
	bon := bondDeLaVarianteDesInfos(uneVarianteEcrite{score: p32(50), nom: "n"})
	if got := infosDe(t, bon); !got.VarianteLue {
		t.Fatalf("le message sain doit etre lu : %+v", got)
	}
	for _, c := range []struct {
		nom string
		v   bondEcrit
	}{
		{"longueur + 1", func() bondEcrit { b := append(bondEcrit{}, bon...); b[0]++; return append(b, 0) }()},
		{"type inconnu", avecUnChampCasse(func(t *bondEcrit) { t.enTete(9, 0x1D) })},
		{"liste plus longue que le tampon", avecUnChampCasse(func(t *bondEcrit) {
			t.enTete(9, bondListe)
			*t = append(*t, bondStructure)
			t.entier(1 << 28)
		})},
		{"chaine plus longue que le tampon", avecUnChampCasse(func(t *bondEcrit) { t.enTete(9, bondChaine); t.entier(1 << 28) })},
		{"table plus longue que le tampon", avecUnChampCasse(func(t *bondEcrit) {
			t.enTete(9, bondTable)
			*t = append(*t, bondOctet, bondOctet)
			t.entier(1 << 28)
		})},
		{"structure de longueur 0", avecUnChampCasse(func(t *bondEcrit) { t.enTete(9, bondStructure); *t = append(*t, 0) })},
		{"structure sans octet de fin", avecUnChampCasse(func(t *bondEcrit) {
			t.enTete(9, bondStructure)
			*t = append(*t, 2, 0x20, 7) // longueur 2 : un champ puis un octet qui n est pas la fin
		})},
		{"trop profond", avecUnChampCasse(func(t *bondEcrit) {
			t.enTete(9, bondListe)
			for range profondeurBondMax + 3 {
				*t = append(*t, byte(bondListe|2<<5)) // une liste d un element, lui-meme une liste
			}
			*t = append(*t, byte(bondOctet|1<<5)) // une liste vide d octets
		})},
		{"entier de onze octets", avecUnChampCasse(func(t *bondEcrit) {
			t.enTete(9, bondEntierNS64)
			*t = append(*t, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x01)
		})},
		{"longueur de chaine sur six octets", avecUnChampCasse(func(t *bondEcrit) {
			t.enTete(9, bondChaine)
			*t = append(*t, 0x81, 0x80, 0x80, 0x80, 0x80, 0x00, 'a') // la valeur 1, sur six octets
		})},
	} {
		if got := infosDe(t, c.v); got.VarianteLue || got.ScoreDeVictoire != nil || got.Nom != nil {
			t.Errorf("%s : %+v", c.nom, got)
		}
	}
	// Le dernier octet du message est l octet de fin de la structure de tete, un zero : lu hors du tampon il
	// vaudrait zero aussi, et seule la longueur annoncee de la structure le dit.
	l := &lecteurDuCorps{br: LecteurSur(bon[:len(bon)-1]), ok: true}
	if got := lireLaVarianteDesInfos(l); got.VarianteLue {
		t.Errorf("message coupe avant son octet de fin : %+v", got)
	}
}

// TestUnCorpsCoupeNePaniquePas : un `chunk_00` coupe a chaque octet du message ne fait jamais paniquer ;
// et ne rend la variante que s il tient entier.
func TestUnCorpsCoupeNePaniquePas(t *testing.T) {
	v := bondDeLaVarianteDesInfos(uneVarianteEcrite{score: p32(50), nom: "Slayer", options: [][2]any{
		{int32(1), []byte{0, 0, 0, 0x3f, 3, 0, 0, 0}}}, chargements: []chargementEcrit{{armes: []int32{3}}}, bruit: true})
	buf, id := chunkDesInfos(unGUID(1), unGUID(2), 9, v, true)
	if got := lireLesInfosDeMatch(buf, id); !got.VarianteLue {
		t.Fatalf("le vecteur entier doit etre lu : %+v", got)
	}
	// Apres la variante, le corps tient encore une chaine, un mot et un bloc de 0x6C0 bits : un tampon coupe
	// avant la fin de la variante ne la rend jamais.
	const apresLaVariante = 4 + 1 + 4 + largeurQueueDuCorps/8 + 2
	for n := 0; n < len(buf); n++ {
		got := lireLesInfosDeMatch(buf[:n], id)
		if got.VarianteLue && n < len(buf)-apresLaVariante-1 {
			t.Fatalf("coupe a %d octets sur %d : variante rendue", n, len(buf))
		}
	}
}

// TestLesJokersDeCheminAcceptentToutRang : un rang quelconque dans un modele de chemin, et le prefixe d un
// modele suivi.
func TestLesJokersDeCheminAcceptentToutRang(t *testing.T) {
	m := []int{1, jokerDeChemin, 2}
	for _, c := range []struct {
		chemin []int
		pref   bool
		egal   bool
	}{
		{[]int{1}, true, false}, {[]int{1, 9}, true, false}, {[]int{1, 9, 2}, true, true},
		{[]int{1, 9, 3}, false, false}, {[]int{2, 9, 2}, false, false}, {[]int{1, 9, 2, 0}, false, false}, {nil, true, false},
	} {
		if got := cheminAvecJokers(c.chemin, m); got != c.pref {
			t.Errorf("%v prefixe : %v", c.chemin, got)
		}
		if got := cheminEgal(c.chemin, m); got != c.egal {
			t.Errorf("%v egal : %v", c.chemin, got)
		}
	}
}

// ecrireDuBruit ecrit, dans une structure suivie, des champs de tous les types lus a des id que la lecture ne
// suit pas : ils sont lus puis jetes, et le champ suivi qui les suit se lit.
func ecrireDuBruit(e *bondEcrit) {
	e.enTete(20, bondFlottant32)
	*e = append(*e, 0, 0, 0x80, 0x3f)
	e.enTete(21, bondFlottant64)
	*e = append(*e, 1, 2, 3, 4, 5, 6, 7, 8)
	e.chaine(22, "Multiplayer/Test")
	e.enTete(23, bondChaineLarge)
	e.entier(2)
	*e = append(*e, 'a', 0, 'b', 0)
	e.enTete(24, bondEntierNS64)
	*e = append(*e, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01) // dix octets
	e.enTete(25, bondOctetSigne)
	*e = append(*e, 0xff)
	e.enTete(26, bondTable)
	*e = append(*e, byte(bondOctet), byte(bondChaine))
	e.entier(1)
	*e = append(*e, 7, 2, 'h', 'i')
	e.enTete(27, bondEnsemble)
	e.enTeteDeListe(bondEntierNS16, 2)
	*e = append(*e, 5, 0x81, 0x01)
	e.enTete(28, bondListe)
	e.enTeteDeListe(bondListe, 1)
	e.enTeteDeListe(bondOctet, 2)
	*e = append(*e, 1, 2)
	e.listeOctets(29, []byte{1, 2, 3, 4, 5, 6, 7})
}

// TestLesLongueursBondSontBornees : sans structure englobante qui rattrape l erreur, une longueur de structure,
// un compte de liste ou de table plus grands que le tampon, ou un entier de longueur sur six octets, font
// tomber le curseur. Un compte enorme de booleens ne boucle pas : il tombe avant le premier element.
// MUTATIONS — la borne d une longueur de structure, d un compte de liste, d un compte de table, la largeur
// d une longueur : ROUGES.
func TestLesLongueursBondSontBornees(t *testing.T) {
	zeros := make([]byte, 16)
	for _, c := range []struct {
		nom string
		ty  int
		oct []byte
	}{
		{"structure plus longue que le tampon", bondStructure, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x0F}},
		{"longueur de structure sur six octets", bondStructure, []byte{0x82, 0x80, 0x80, 0x80, 0x80, 0x00, 0, 0}},
		{"liste de booleens plus longue que le tampon", bondListe, []byte{byte(bondBooleen), 0x80, 0x80, 0x80, 0x80, 0x01}},
		{"table plus longue que le tampon", bondTable, []byte{byte(bondOctet), byte(bondOctet), 0x80, 0x80, 0x80, 0x80, 0x01}},
		{"chaine plus longue que le tampon", bondChaine, []byte{0x80, 0x80, 0x80, 0x80, 0x01}},
		{"chaine large plus longue que le tampon", bondChaineLarge, []byte{0x80, 0x80, 0x80, 0x80, 0x01}},
	} {
		l := &lecteurDuCorps{br: LecteurSur(append(append([]byte{}, c.oct...), zeros...)), ok: true}
		l.valeurBond(c.ty, nil, visiteQuiIgnore(), 0)
		if l.ok {
			t.Errorf("%s : le curseur ne tombe pas", c.nom)
		}
	}
	// Une structure saine de deux octets, elle, s enjambe.
	l := &lecteurDuCorps{br: LecteurSur([]byte{2, 0x20, 0}), ok: true}
	l.valeurBond(bondStructure, nil, visiteQuiIgnore(), 0)
	if !l.ok {
		t.Errorf("structure saine non enjambee")
	}
}

// TestLesInfosDeMatchDesBobines : le `chunk_00` des sept bobines (un film par build) rend les infos de match
// que le decodeur tiers rend sur les films de HI_1_12_0 et HI_1_13_0, et que la table de chaines et les
// registres du match confirment sur les autres (identifiant de carte, nom de variante, score).
// MUTATIONS — un decalage de bit sur les deux identifiants ou sur l heure, un chemin de champ decale : ROUGES.
func TestLesInfosDeMatchDesBobines(t *testing.T) {
	type attendu struct {
		debut                 uint32
		asset, version        string
		nom, description      string
		score, duree, respawn int64
		prolongation          int64 // -1 : absente
		armes                 []uint32
		aleatoire             bool
		options               int
	}
	for film, a := range map[string]attendu{
		"a521164d": {1693653278, "529a840ddbfeea4ab5a3caee268f1f49", "1a552a36476da440a0e0c7a844c2315d", "Total Control",
			"Control all three zones at once to score.", 3, 900, 10, -1, []uint32{723002581, 1220648237}, false, 0},
		"60ae07c4": {1728939350, "f8539230757aff4883e1e7fb3db2ac47", "330d091e653c6344bf4c41d79c0c6644", "Ranked Oddball",
			"Oddball with precision weapons, friendly fire, and no radar.", 100, 300, 10, 60, []uint32{1791870029}, false, 0},
		"11de8353": {1731432075, "28aca3289df6a94f9ebfa0449c89c8da", "48d32a3428ec8b41b8fd601cf38b5005", "Big Team Fiesta",
			"Slayer with Random weapons. This is Madness!", 100, 900, 10, -1, nil, true, 0},
		"111fa685": {1743012059, "90349f2ce26b904d901502095651e91e", "cbf8671bbe346f47b5aad625530f29d1", "Big Team Fiesta",
			"Slayer with Random weapons. This is Madness!", 100, 900, 10, -1, nil, true, 0},
		"e5adf7b2": {1752170182, "f4418f06dde3ec4bb297d2ded85ab54b", "724c0b2fe5511541b3020bae4b4df7dd", "Big Team Fiesta",
			"Slayer with Random weapons. This is Madness!", 100, 900, 10, -1, nil, true, 0},
		"bcb6d393": {1763066934, "4b362453a839934f96a6b80a1f18ce8a", "7e53785ad553e244bdbdd2088038523c", "Capture the Flag",
			"Capture the enemy Flag while defending your own.", 3, 720, 10, -1, []uint32{1220648237, 4094171407}, false, 0},
		"fb1a1a72": {1784839686, "d826d29a47895b4c95bcd220187698c1", "979f422bca31c34dbec3f04825991ab0", "Capture the Flag",
			"Capture the enemy Flag while defending your own.", 3, 720, 10, -1, []uint32{1220648237, 4094171407}, false, 0},
	} {
		id, err := ReadFilmIdentity(bobineChunk00(t, film))
		if err != nil {
			t.Fatalf("%s : %v", film, err)
		}
		i := id.InfosDeMatch
		if !i.VarianteLue || i.DebutUnix == nil || *i.DebutUnix != a.debut || i.CarteAssetID == nil ||
			hex.EncodeToString(i.CarteAssetID[:]) != a.asset || i.CarteVersionID == nil ||
			hex.EncodeToString(i.CarteVersionID[:]) != a.version {
			t.Errorf("%s : en-tete %+v", film, i)
		}
		if i.Nom == nil || *i.Nom != a.nom || i.Description == nil || *i.Description != a.description {
			t.Errorf("%s : nom, description %v %v", film, i.Nom, i.Description)
		}
		egal64(t, film+" score", i.ScoreDeVictoire, a.score)
		egal64(t, film+" duree", i.DureeSecondes, a.duree)
		egal64(t, film+" reapparition", i.DelaiDeReapparitionSecondes, a.respawn)
		if a.prolongation < 0 {
			absent64(t, film+" prolongation", i.ProlongationSecondes)
		} else {
			egal64(t, film+" prolongation", i.ProlongationSecondes, a.prolongation)
		}
		absent64(t, film+" derniere chance", i.DerniereChanceSecondes)
		if d := i.Dotation; d == nil || d.Aleatoire != a.aleatoire || len(d.Armes) != len(a.armes) || d.Equipement != nil {
			t.Errorf("%s : dotation %+v", film, i.Dotation)
		} else {
			for k, w := range a.armes {
				if d.Armes[k] != w {
					t.Errorf("%s : arme %d : %d, attendu %d", film, k, d.Armes[k], w)
				}
			}
		}
	}
}
