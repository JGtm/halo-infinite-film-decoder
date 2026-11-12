package grammar

// film_infos_de_match_test.go — les vecteurs des infos de match : un message Bond de la variante ecrit a
// l octet pres, des identifiants de carte et une heure de debut, puis le corps lu par
// [lireLesInfosDeMatch]. Chaque champ a son vecteur, chaque condition de lecture a un vecteur qui la
// renverse.

import (
	"math"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// octet ecrit un champ d octet non signe (type 3).
func (b *bondEcrit) octet(id int, v byte) {
	b.enTete(id, bondOctet)
	*b = append(*b, v)
}

// chaine ecrit un champ de chaine (type 9).
func (b *bondEcrit) chaine(id int, s string) {
	b.enTete(id, bondChaine)
	b.entier(uint64(len(s)))
	*b = append(*b, s...)
}

// listeOctets ecrit une liste d octets (type 11, element de type 3).
func (b *bondEcrit) listeOctets(id int, d []byte) {
	b.enTete(id, bondListe)
	b.enTeteDeListe(bondOctet, len(d))
	*b = append(*b, d...)
}

// enTeteDeListe ecrit l octet de tete d une liste de `n` elements du type `et` (`FUN_140ac7430`).
func (b *bondEcrit) enTeteDeListe(et, n int) {
	if n < 6 {
		*b = append(*b, byte(et|(n+1)<<5))
		return
	}
	*b = append(*b, byte(et))
	b.entier(uint64(n))
}

// listeEntiers32 ecrit une liste d entiers signes de 32 bits (type 16).
func (b *bondEcrit) listeEntiers32(id int, vs ...int32) {
	b.enTete(id, bondListe)
	b.enTeteDeListe(bondEntier32, len(vs))
	for _, v := range vs {
		b.entier(uint64(uint32(v<<1) ^ uint32(v>>31))) //nolint:gosec // zigzag
	}
}

// listeDeStructures ecrit une liste de structures de n elements (n quelconque, tete `FUN_140ac7430`).
func (b *bondEcrit) listeDeStructures(id int, elements ...func(*bondEcrit)) {
	b.enTete(id, bondListe)
	b.enTeteDeListe(bondStructure, len(elements))
	for _, e := range elements {
		b.structure(e)
	}
}

// uneVarianteEcrite decrit les champs d une variante de vecteur ; zero = champ non ecrit.
type uneVarianteEcrite struct {
	nom, description string // ecrits en bloc, avec decalage, quand non vides
	score, duree     *int32
	reapparition     *int32
	prolongation     *int32
	chanceActive     bool
	chance           *int32
	chargements      []chargementEcrit
	options          [][2]any // {hachage int32, 8 octets []byte}
	bruit            bool     // champs inconnus de tous les types, a des id non suivis
}

// chargementEcrit : un chargement de depart.
type chargementEcrit struct {
	armes      []int32
	aleatoire  bool
	equipement *int32
}

func (c chargementEcrit) ecrire(b *bondEcrit) {
	b.listeDeStructures(1, func() []func(*bondEcrit) {
		var es []func(*bondEcrit)
		for _, a := range c.armes {
			es = append(es, func(w *bondEcrit) { w.champStructure(0, func(x *bondEcrit) { x.entier32(0, a) }) })
		}
		if c.aleatoire {
			es = append(es, func(w *bondEcrit) { w.booleen(2, true) }) // l element d arme porte le drapeau
		}
		return es
	}()...)
	if c.equipement != nil {
		b.champStructure(2, func(e *bondEcrit) {
			e.champStructure(0, func(x *bondEcrit) { x.entier32(0, *c.equipement) })
		})
	}
}

// bondDeLaVarianteDesInfos ecrit une variante comme `FUN_140b85504` : la structure de tete dont le champ 1
// est une liste de structures de variante, avec les champs que la lecture suit.
func bondDeLaVarianteDesInfos(vs ...uneVarianteEcrite) bondEcrit {
	var b bondEcrit
	b.structure(func(t *bondEcrit) {
		var es []func(*bondEcrit)
		for _, v := range vs {
			es = append(es, v.ecrire)
		}
		t.listeDeStructures(1, es...)
	})
	return b
}

// blocDeChaines rend le bloc d octets et les decalages des chaines non vides, dans l ordre.
func blocDeChaines(chaines ...string) ([]byte, []int32) {
	var bloc []byte
	var dec []int32
	for _, s := range chaines {
		dec = append(dec, int32(len(bloc))) //nolint:gosec // petit
		bloc = append(bloc, s...)
		bloc = append(bloc, 0)
	}
	return bloc, dec
}

func (v uneVarianteEcrite) ecrire(s *bondEcrit) {
	if v.bruit {
		s.champStructure(0, func(e *bondEcrit) { e.entier32(0, 2); e.chaine(6, "Multiplayer/Test") })
	}
	s.champStructure(1, func(o *bondEcrit) {
		if v.bruit {
			ecrireDuBruit(o)
		}
		if v.reapparition != nil {
			o.champStructure(2, func(r *bondEcrit) { r.entier32(3, *v.reapparition) })
		}
		if len(v.chargements) > 0 {
			var es []func(*bondEcrit)
			for _, c := range v.chargements {
				es = append(es, c.ecrire)
			}
			o.champStructure(6, func(r *bondEcrit) { r.listeDeStructures(1, es...) })
		}
		if v.score != nil || v.duree != nil || v.prolongation != nil || v.chanceActive || v.chance != nil {
			o.champStructure(7, func(r *bondEcrit) {
				if v.score != nil {
					r.entier32(0, *v.score)
				}
				if v.duree != nil {
					r.entier32(3, *v.duree)
				}
				if v.prolongation != nil {
					r.entier32(9, *v.prolongation)
				}
				if v.chanceActive {
					r.booleen(10, true)
				}
				if v.chance != nil {
					r.entier32(11, *v.chance)
				}
			})
		}
	})
	if v.nom != "" || v.description != "" || len(v.options) > 0 {
		s.champStructure(3, func(x *bondEcrit) { v.ecrireLesChaines(x) })
	}
}

// ecrireLesChaines ecrit le champ 3 de la variante : le bloc des chaines (3.0.0.0), la table des decalages
// (3.0.1.0) et les options (3.9.0).
func (v uneVarianteEcrite) ecrireLesChaines(x *bondEcrit) {
	bloc, dec := blocDeChaines("Autre", v.nom, v.description)
	// Le bloc range une chaine d une autre langue d abord (rang 0 : le nom a le decalage dec[1]).
	x.champStructure(0, func(c *bondEcrit) {
		c.champStructure(0, func(d *bondEcrit) { d.listeOctets(0, bloc) })
		c.champStructure(1, func(d *bondEcrit) {
			d.listeDeStructures(0,
				func(e *bondEcrit) { e.octet(0, 1); e.listeEntiers32(1, dec[1], dec[0], dec[0]) },
				func(e *bondEcrit) { e.octet(0, 1); e.listeEntiers32(1, dec[2], dec[0]) },
				func(e *bondEcrit) { e.octet(0, 1); e.listeEntiers32(1, dec[0]) })
		})
	})
	if len(v.options) > 0 {
		var es []func(*bondEcrit)
		for _, o := range v.options {
			es = append(es, func(e *bondEcrit) {
				if h, ok := o[0].(int32); ok { // sans hachage : une valeur orpheline
					e.champStructure(0, func(x *bondEcrit) { x.entier32(0, h) })
				}
				e.champStructure(1, func(h *bondEcrit) { h.listeOctets(0, o[1].([]byte)) })
			})
		}
		x.champStructure(9, func(d *bondEcrit) { d.listeDeStructures(0, es...) })
	}
}

// chunkDesInfos ecrit un `chunk_00` de vecteur : deux identifiants de 16 octets, puis le corps. Rend le
// tampon et l identite que [ReadFilmIdentity] poserait.
func chunkDesInfos(asset, version [16]byte, debut uint32, variante bondEcrit, presente bool) ([]byte, profile.FilmIdentity) {
	var w bitWriter
	for _, o := range asset {
		w.bits(uint64(o), 8)
	}
	for _, o := range version {
		w.bits(uint64(o), 8)
	}
	corps := corpsEcrit{bondDeTete: bondDeTeteDeTest(), chaine: []byte("Slayer")}
	if presente {
		corps.variante = &varianteEcrite{}
		corps.casser = func(bondEcrit) bondEcrit { return variante }
	}
	w.ecrireCorps(corps)
	w.bits(0xffff, 16)
	id := profile.FilmIdentity{BodyBit: 256, MatchStartUnix: debut,
		Variante: profile.VarianteDePartie{Lue: true, Presente: presente}}
	return w.buf, id
}

func p32(v int32) *int32 { return &v }

func unGUID(n byte) [16]byte {
	var g [16]byte
	for i := range g {
		g[i] = n + byte(i)
	}
	return g
}

// infosDe lit les infos d une variante de vecteur.
func infosDe(t *testing.T, v bondEcrit) profile.InfosDeMatch {
	t.Helper()
	buf, id := chunkDesInfos(unGUID(1), unGUID(0x40), 1700000000, v, true)
	return lireLesInfosDeMatch(buf, id)
}

func egal64(t *testing.T, nom string, got *int64, want int64) {
	t.Helper()
	if got == nil || *got != want {
		t.Errorf("%s : %v, attendu %d", nom, got, want)
	}
}

func absent64(t *testing.T, nom string, got *int64) {
	t.Helper()
	if got != nil {
		t.Errorf("%s : %d, attendu absent", nom, *got)
	}
}

// TestLesInfosDeMatchSeLisentChampParChamp : chaque champ de la variante est lu a son chemin, a l octet
// pres ; l en-tete donne l heure de debut et les deux identifiants de carte.
// MUTATIONS — un chemin decale d un id, le nom lu au decalage de la description, le rang 0 de la liste de
// langues remplace par le rang 1, la derniere chance rendue sans son drapeau, les armes lues dans le
// second chargement : ROUGES.
func TestLesInfosDeMatchSeLisentChampParChamp(t *testing.T) {
	equip := p32(-77)
	v := uneVarianteEcrite{
		nom: "Slayer d essai", description: "Elimine tout le monde.",
		score: p32(50), duree: p32(720), reapparition: p32(8), prolongation: p32(120),
		chanceActive: true, chance: p32(30),
		chargements: []chargementEcrit{
			{armes: []int32{-5, 1234567}, equipement: equip},
			{armes: []int32{99}, aleatoire: true},
		},
		options: [][2]any{
			{int32(-3), []byte{0, 0, 0, 0x3f, 3, 0, 0, 0}},
			{int32(9), []byte{1, 0, 0, 0, 2, 0, 0, 0}},
		},
		bruit: true,
	}
	buf, id := chunkDesInfos(unGUID(1), unGUID(0x40), 1700000000, bondDeLaVarianteDesInfos(v), true)
	got := lireLesInfosDeMatch(buf, id)
	if !got.VarianteLue {
		t.Fatalf("variante non lue : %+v", got)
	}
	if got.DebutUnix == nil || *got.DebutUnix != 1700000000 {
		t.Errorf("debut : %v", got.DebutUnix)
	}
	if got.CarteAssetID == nil || *got.CarteAssetID != profile.GUID(unGUID(1)) ||
		got.CarteVersionID == nil || *got.CarteVersionID != profile.GUID(unGUID(0x40)) {
		t.Errorf("identifiants de carte : %v %v", got.CarteAssetID, got.CarteVersionID)
	}
	if got.Nom == nil || *got.Nom != "Slayer d essai" || got.Description == nil || *got.Description != "Elimine tout le monde." {
		t.Errorf("nom, description : %v %v", got.Nom, got.Description)
	}
	egal64(t, "score", got.ScoreDeVictoire, 50)
	egal64(t, "duree", got.DureeSecondes, 720)
	egal64(t, "reapparition", got.DelaiDeReapparitionSecondes, 8)
	egal64(t, "prolongation", got.ProlongationSecondes, 120)
	egal64(t, "derniere chance", got.DerniereChanceSecondes, 30)
	d := got.Dotation
	if d == nil || len(d.Armes) != 2 || d.Armes[0] != uint32(0xFFFFFFFB) || d.Armes[1] != 1234567 || d.Aleatoire ||
		d.Equipement == nil || *d.Equipement != uint32(0xFFFFFFB3) {
		t.Errorf("dotation : %+v", d)
	}
	want := []profile.OptionDeVariante{
		{Hachage: 0xFFFFFFFD, Genre: profile.OptionNombre, Valeur: 0.5},
		{Hachage: 9, Genre: profile.OptionBooleenne, Valeur: 1},
	}
	if len(got.Options) != len(want) || got.Options[0] != want[0] || got.Options[1] != want[1] {
		t.Errorf("options : %+v", got.Options)
	}
}

// TestUnChampNonEcritResteAbsent : un champ que la variante n ecrit pas est absent, jamais zero ; la
// derniere chance sans son drapeau, et une variante sans chargement, sont absentes. Un zero ECRIT est lu.
// MUTATIONS — le drapeau de derniere chance ignore, un pointeur pose a zero par defaut : ROUGES.
func TestUnChampNonEcritResteAbsent(t *testing.T) {
	got := infosDe(t, bondDeLaVarianteDesInfos(uneVarianteEcrite{chance: p32(30), score: p32(0)}))
	if !got.VarianteLue {
		t.Fatalf("variante non lue")
	}
	egal64(t, "score ecrit a zero", got.ScoreDeVictoire, 0)
	absent64(t, "duree", got.DureeSecondes)
	absent64(t, "reapparition", got.DelaiDeReapparitionSecondes)
	absent64(t, "prolongation", got.ProlongationSecondes)
	absent64(t, "derniere chance sans drapeau", got.DerniereChanceSecondes)
	if got.Nom != nil || got.Description != nil || got.Dotation != nil || len(got.Options) != 0 {
		t.Errorf("champs non ecrits rendus : %+v", got)
	}
}

// TestLeCheminSuitUneSeuleVariante : deux variantes dans le message, ou un champ de tete qui n est pas une
// liste de variantes, ne rendent rien de la variante ; l en-tete reste lu.
// MUTATIONS — le compte de variantes non verifie : ROUGE.
func TestLeCheminSuitUneSeuleVariante(t *testing.T) {
	deux := bondDeLaVarianteDesInfos(uneVarianteEcrite{score: p32(50)}, uneVarianteEcrite{score: p32(60)})
	got := infosDe(t, deux)
	if got.VarianteLue || got.ScoreDeVictoire != nil {
		t.Errorf("deux variantes : %+v", got)
	}
	if got.DebutUnix == nil || got.CarteAssetID == nil {
		t.Errorf("l en-tete doit rester lu : %+v", got)
	}
	var horsListe bondEcrit // une variante dans un champ de tete qui n est pas l une des listes de variantes
	horsListe.structure(func(t *bondEcrit) {
		t.listeDeStructures(5, uneVarianteEcrite{score: p32(50)}.ecrire)
	})
	if got := infosDe(t, horsListe); got.VarianteLue || got.ScoreDeVictoire != nil {
		t.Errorf("champ de tete 5 : %+v", got)
	}
	var zero bondEcrit
	zero.structure(func(*bondEcrit) {})
	if got := infosDe(t, zero); got.VarianteLue {
		t.Errorf("aucune variante dans le message : %+v", got)
	}
}

// TestUneVarianteAbsenteOuNonLueNeRendQueLEnTete : la variante absente du corps, ou non lue par la marche du
// corps, ne rend que l heure et les identifiants de carte.
// MUTATIONS — la garde sur l identite de la variante retiree : ROUGE.
func TestUneVarianteAbsenteOuNonLueNeRendQueLEnTete(t *testing.T) {
	v := bondDeLaVarianteDesInfos(uneVarianteEcrite{score: p32(50)})
	buf, id := chunkDesInfos(unGUID(1), unGUID(2), 5, v, false)
	if got := lireLesInfosDeMatch(buf, id); got.VarianteLue || got.DebutUnix == nil || *got.DebutUnix != 5 {
		t.Errorf("variante absente : %+v", got)
	}
	buf, id = chunkDesInfos(unGUID(1), unGUID(2), 5, v, true)
	id.Variante.Lue = false
	if got := lireLesInfosDeMatch(buf, id); got.VarianteLue || got.ScoreDeVictoire != nil {
		t.Errorf("variante non lue par la marche : %+v", got)
	}
	id.Variante = profile.VarianteDePartie{Lue: true, Presente: false}
	if got := lireLesInfosDeMatch(buf, id); got.VarianteLue || got.ScoreDeVictoire != nil {
		t.Errorf("variante non presente : %+v", got)
	}
}

// TestLEnTeteSeLitOuResteAbsent : l heure nulle, des identifiants nuls et des identifiants hors du tampon
// sont absents.
// MUTATIONS — le zero rendu comme une valeur, la borne du tampon retiree : ROUGES.
func TestLEnTeteSeLitOuResteAbsent(t *testing.T) {
	buf, id := chunkDesInfos([16]byte{}, unGUID(3), 0, bondDeLaVarianteDesInfos(uneVarianteEcrite{}), true)
	got := lireLesInfosDeMatch(buf, id)
	if got.DebutUnix != nil || got.CarteAssetID != nil || got.CarteVersionID == nil {
		t.Errorf("zeros : %+v", got)
	}
	id.BodyBit = 100 // les deux blocs de 16 octets ne tiennent plus avant le corps
	if got := lireLesInfosDeMatch(buf, id); got.CarteAssetID != nil || got.CarteVersionID != nil {
		t.Errorf("hors du tampon : %+v", got)
	}
	if got := lireUnGUID(buf[:16], 8); got != nil {
		t.Errorf("guid coupe : %v", got)
	}
}

// TestLesChainesSeLisentParLeurDecalage : un decalage hors du bloc, une chaine sans octet nul, vide ou
// qui n est pas de l UTF-8 sont absents.
// MUTATIONS — la borne du decalage, le test du nul, la validite UTF-8 : ROUGES.
func TestLesChainesSeLisentParLeurDecalage(t *testing.T) {
	bloc := []byte("abc\x00\x00\xff\xfe\x00fin")
	neuf := func(v int64) *int64 { return &v }
	for _, c := range []struct {
		nom string
		dec *int64
		ok  bool
		txt string
	}{
		{"chaine", neuf(0), true, "abc"},
		{"vide", neuf(4), false, ""},
		{"non UTF-8", neuf(5), false, ""},
		{"sans nul final", neuf(8), false, ""},
		{"hors du bloc", neuf(int64(len(bloc))), false, ""},
		{"loin hors du bloc", neuf(int64(len(bloc)) + 5), false, ""},
		{"negatif", neuf(-1), false, ""},
		{"absent", nil, false, ""},
	} {
		got := texteADecalage(bloc, c.dec)
		if (got != nil) != c.ok || (got != nil && *got != c.txt) {
			t.Errorf("%s : %v", c.nom, got)
		}
	}
}

// TestLesOptionsSeLisentParLeurGenre : un genre inconnu, une valeur de longueur inattendue et une valeur
// sans hachage ne sont pas rendus ; le suivant reste lu.
// MUTATIONS — le genre inconnu rendu comme nombre, la longueur non verifiee : ROUGES.
func TestLesOptionsSeLisentParLeurGenre(t *testing.T) {
	v := uneVarianteEcrite{nom: "n", options: [][2]any{
		{int32(1), []byte{0, 0, 0, 0, 9, 0, 0, 0}},
		{int32(2), []byte{0, 0, 0, 0x40, 3, 0, 0}},
		{int32(3), []byte{0, 0, 0xC0, 0xBF, 3, 0, 0, 0}},
		{int32(4), []byte{7, 0, 0, 0, 2, 0, 0, 0}},
		{nil, []byte{1, 0, 0, 0, 2, 0, 0, 0}}, // valeur sans hachage : ne reprend pas celui d avant
	}}
	got := infosDe(t, bondDeLaVarianteDesInfos(v))
	if len(got.Options) != 2 || got.Options[0].Hachage != 3 || got.Options[0].Valeur != float64(math.Float32frombits(0xBFC00000)) ||
		got.Options[1].Hachage != 4 || got.Options[1].Genre != profile.OptionBooleenne || got.Options[1].Valeur != 1 {
		t.Errorf("options : %+v", got.Options)
	}
}
