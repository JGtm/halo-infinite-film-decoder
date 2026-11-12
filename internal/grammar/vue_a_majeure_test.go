package grammar

// vue_a_majeure_test.go — LA CLASSE D UN FILM VIENT DE SA SEULE TABLE DES GENRES : LA VERSION MAJEURE
// N Y ENTRE PAS, ET LA SUITE DES GENRES DIT OU SA NUMEROTATION DEVIENT PRESUMEE.
//
// `FUN_1428e219c` ne lit un film que sous `*film == 0x29` : cette garde dit ce que l executable joue,
// pas ce que l ecrivain d un film a ecrit. Un film dont la table des genres est la native est de
// classe EGALE sous toute majeure (les films HI_1_12_0, majeure 0x28, en font partie) ; la fin de sa
// vue A est le debut de sa vue B sans preuve de fermeture.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// majeureHI1120 est la version majeure des films HI_1_12_0 (0x28), lue en tete de leur `chunk_00` ;
// majeureJouee est la seule que l executable lu joue (0x29).
const (
	majeureHI1120 = 0x28
	majeureJouee  = 0x29
)

// tableNative rend la table des genres que l executable porte.
func tableNative() []uint32 {
	natives := make([]uint32, GenresVueA)
	for g := range natives {
		natives[g] = versionNative(g)
	}
	return natives
}

// profilDeTable resout le profil d un film de version majeure `majeure` (lue quand `lue`) qui declare
// la table `table`, sans carte.
func profilDeTable(table []uint32, majeure int, lue bool) profile.Profile {
	return profile.Resoudre(profile.ClesDuFilm{RegistrePresent: true, Majeure: majeure, MajeureLue: lue,
		Identite: profile.FilmIdentity{TypeVersions: table}, IdentiteLue: true}, nil)
}

// profilDeMajeure resout le profil d un film de table native, de version majeure `majeure` (lue
// quand `lue`), sans carte.
func profilDeMajeure(majeure int, lue bool) profile.Profile {
	return profilDeTable(tableNative(), majeure, lue)
}

// TestLaClasseDUnFilmNeDependQueDeSaTable : une table native rend la classe EGALE, cardinal 123, sous
// toute version majeure — 0x28 des films HI_1_12_0 et 0x29 comprises, majeure non lue comprise ; un
// prefixe strict de la table native (122 genres) reste PREFIXE, une version differente ou une
// identite non lue restent ILLISIBLES, sous 0x28 comme sous 0x29. MUTATION — une garde de version
// majeure reintroduite dans [tableDesGenresDuFilm] : ROUGE.
func TestLaClasseDUnFilmNeDependQueDeSaTable(t *testing.T) {
	autre := tableNative()
	autre[35]++
	for _, c := range []struct {
		nom     string
		table   []uint32
		majeure int
		lue     bool
		classe  classeDeLaVueA
		genres  int
	}{
		{"table native, majeure 0x29", tableNative(), majeureJouee, true, vueAEgale, GenresVueA},
		{"table native, majeure 0x28", tableNative(), majeureHI1120, true, vueAEgale, GenresVueA},
		{"table native, majeure 0x27", tableNative(), 0x27, true, vueAEgale, GenresVueA},
		{"table native, majeure non lue", tableNative(), 0, false, vueAEgale, GenresVueA},
		{"table de 122 genres, majeure 0x29", tableNative()[:122], majeureJouee, true, vueAPrefixe, 122},
		{"table de 122 genres, majeure 0x28", tableNative()[:122], majeureHI1120, true, vueAPrefixe, 122},
		{"version differente, majeure 0x28", autre, majeureHI1120, true, vueAIllisible, 0},
		{"table vide, majeure 0x29", nil, majeureJouee, true, vueAIllisible, 0},
	} {
		if classe, genres := tableDesGenresDuFilm(profilDeTable(c.table, c.majeure, c.lue)); classe != c.classe ||
			genres != c.genres {
			t.Errorf("%s : (%d, %d), attendu (%d, %d)", c.nom, classe, genres, c.classe, c.genres)
		}
	}
	sansIdentite := profile.Resoudre(profile.ClesDuFilm{RegistrePresent: true, Majeure: majeureHI1120, MajeureLue: true}, nil)
	if classe, genres := tableDesGenresDuFilm(sansIdentite); classe != vueAIllisible || genres != 0 {
		t.Errorf("identite non lue : (%d, %d), attendu illisible", classe, genres)
	}
	for film, attendu := range map[string]classeDeLaVueA{"bcb6d393": vueAEgale, "fb1a1a72": vueAEgale} {
		if classe, _ := tableDesGenresDuFilm(ResolveProfile(bobineFilm(t, film), nil)); classe != attendu {
			t.Errorf("bobine %s : classe %d, attendu %d", film, classe, attendu)
		}
	}
}

// TestLaFinDeLaVueADunFilmDeMajeure0x28EstLeDebutDeSaVueB : le paquet de
// [TestLaFinDeLaVueAPrimeQuandLaMarcheButeEnsuite] — la marche depuis E bute, une signature suit.
// Sous la table native, majeures 0x28 et 0x29 comprises, la marche part de E, comme la regle du
// lecteur du jeu ; sous un prefixe de la table (122 genres), E n est pas prouve et le paquet suit le
// localisateur a l identique. MUTATION — la classe d un film de table native et de majeure 0x28 ramenee
// a PREFIXE : ROUGE.
func TestLaFinDeLaVueADunFilmDeMajeure0x28EstLeDebutDeSaVueB(t *testing.T) {
	pay, e := paquetVueA(1, zoomCourt, func(w *bitWriter) {
		w.deltaMasque13(124, 5)
		w.bit(0)
		w.signature123()
		w.finDeVueB()
		w.vueCUneEntree()
	})
	w := mondeDeCarte(compHighFrequency)
	cfg := cadreDeCarte()
	if l := lectureDEssai(pay, w, cfg, e); l.Fermee {
		t.Fatalf("la marche depuis E ferme le paquet : le vecteur doit la faire buter")
	}
	s, commentS := localiserLaListe(pay, w, cfg)
	if s == e {
		t.Fatalf("la signature donne E = %d : le vecteur doit les separer", e)
	}
	for _, majeure := range []int{majeureJouee, majeureHI1120} {
		g := grammaireDeLaVueASousFilm(profilDeMajeure(majeure, true))
		if d, comment := debutDuPaquet(t, pay, w, g); d != e || comment != lecture.DebutParVueA {
			t.Errorf("table native, majeure %#x : debut (%d, %d) ; attendu %d par la vue A", majeure, d, comment, e)
		}
	}
	ancien := grammaireDeLaVueASousFilm(profilDeTable(tableNative()[:122], majeureHI1120, true))
	if d, comment := debutDuPaquet(t, pay, w, ancien); d == e || d != s || comment != commentS {
		t.Errorf("table de 122 genres : debut (%d, %d) ; attendu le localisateur (%d, %d), et pas E = %d",
			d, comment, s, commentS, e)
	}
}

// TestLaSuiteDesGenresDitOuSaNumerotationEstPresumee : la vue A d un zoom (genre 21), puis d un genre
// au-dela du 107 que la lecture ne porte pas (elle s arrete apres lui). Sous un film PREFIXE, le
// second genre est presume (rang 1) ; sous un film EGALE, aucun (rang = 2). Le rang est range dans
// `lecture.Paquet.VueA`. Le dernier genre a version distinctive est bien le 107 de la table native.
// MUTATIONS — [premierGenrePresume] rend toujours len(genres) ; le rang non range : ROUGES.
func TestLaSuiteDesGenresDitOuSaNumerotationEstPresumee(t *testing.T) {
	dernier := -1
	for g := range GenresVueA {
		if versionNative(g) != 1 {
			dernier = g
		}
	}
	if dernier != dernierGenreAVersionDistinctive {
		t.Fatalf("dernier genre a version distinctive %d, constante %d", dernier, dernierGenreAVersionDistinctive)
	}
	pay, _ := paquetVueA(1, func(w *bitWriter) {
		zoomCourt(w)
		w.ecrireEnTeteDeMessage(dernierGenreAVersionDistinctive + 3)
	}, vueBFermee)
	for _, c := range []struct {
		nom     string
		g       grammaireDeLaVueA
		attendu uint16
	}{
		{"film PREFIXE", grammaireDeTest(vueAPrefixe, GenresVueA), 1},
		{"film EGALE", grammaireRecente(), 2},
	} {
		p := lecture.Paquet{Payload: pay}
		rangerLaTete(&p, ProfilDeBalayageParDefaut(), c.g)
		if len(p.VueA.Genres) != 2 || p.VueA.Genres[1] != dernierGenreAVersionDistinctive+3 {
			t.Fatalf("%s : genres %v, attendu [21 %d]", c.nom, p.VueA.Genres, dernierGenreAVersionDistinctive+3)
		}
		if p.VueA.PremierPresume != c.attendu {
			t.Errorf("%s : premier genre presume au rang %d, attendu %d", c.nom, p.VueA.PremierPresume, c.attendu)
		}
	}
}
