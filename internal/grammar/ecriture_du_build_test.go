package grammar

// ecriture_du_build_test.go — CE QUE L ECRIVAIN D UN BUILD ECRIT ([profile.EcritureDuBuild]) : les trois
// dispositions qui ne valent que sur les builds ou elles sont prouvees, et leur effet sur la lecture.

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// registreJusquA rend un registre dont les archetypes 0 a n-1 n ont aucun composant.
func registreJusquA(n int) *Registry {
	archs := make([]Archetype, n)
	for i := range archs {
		archs[i] = Archetype{Index: i}
	}
	return &Registry{Archetypes: archs}
}

// leRecordNeuf lit un record NEW d archetype `ti` ecrit par `ecrire` sous les bascules `parPorte` et `complete`,
// suivi du marqueur 0x2a, et rend la trace et le lecteur.
func leRecordNeuf(ti uint64, parPorte, complete bool, ecrire func(w *bitw)) (EntityTrace, *Lecteur) {
	w := &bitw{}
	w.put(ti, 6)
	ecrire(w)
	w.put(0x2a, 8)
	br := lecteurDInstrument(append(w.buf, make([]byte, 16)...))
	br.p.Grammaire.CreationParPorte, br.p.Grammaire.ProprieteGereeComplete = parPorte, complete
	return TraverseEntity(br, registreJusquA(50), 0), br
}

// TestLaPorteDeCreationA0SauteLeMasque : un record NEW dont la porte de composants vaut 0 ne lit pas de
// masque sous [GrammaireBalayage.CreationParPorte] ; sans elle, le masque (R(1) et R(3)) se lit toujours.
// MUTATIONS — `t.Gate || !parPorte` remplace par `true` : ROUGE ; par `t.Gate` seul : ROUGE.
func TestLaPorteDeCreationA0SauteLeMasque(t *testing.T) {
	// ti=10 : version a 0, reference absente, porte a 0.
	etat := func(w *bitw) { w.put(0, 1); w.put(0, 1); w.put(0, 1) }
	tr, br := leRecordNeuf(10, true, false, etat)
	if tr.EndBit != 9 || tr.Gate || tr.Mask != 0 || br.ReadBits(8) != 0x2a {
		t.Errorf("par porte : fin %d (porte %v, masque %#x), attendu la fin au bit 9 sans masque lu", tr.EndBit, tr.Gate, tr.Mask)
	}
	// Sans la lecture par porte, la porte a 0 est suivie de son masque : R(1) = 0, R(3) = 0.
	avecMasque := func(w *bitw) { etat(w); w.put(0, 4) }
	tr, br = leRecordNeuf(10, false, false, avecMasque)
	if tr.EndBit != 13 || br.ReadBits(8) != 0x2a {
		t.Errorf("sans lecture par porte : fin %d, attendu 13 (porte et masque lus)", tr.EndBit)
	}
	// Une porte a 1 est suivie de son masque dans les deux lectures.
	avecPorte := func(w *bitw) { w.put(0, 1); w.put(0, 1); w.put(1, 1); w.put(0, 4) }
	if tr, _ = leRecordNeuf(10, true, false, avecPorte); tr.EndBit != 13 || !tr.Gate {
		t.Errorf("porte a 1 : fin %d (porte %v), attendu 13", tr.EndBit, tr.Gate)
	}
}

// TestUnArchetypeAMasqueDeCreationVideNeLitPasDePorte : sous la lecture par porte, ti=47 (version, R(5))
// ecrit son etat puis rien : ni porte ni masque. Sans elle la porte se lit. MUTATION —
// [masqueDeCreationVide] rendant faux pour 47 : ROUGE.
func TestUnArchetypeAMasqueDeCreationVideNeLitPasDePorte(t *testing.T) {
	etat := func(w *bitw) { w.put(0, 1); w.put(0x15, 5) }
	tr, br := leRecordNeuf(47, true, false, etat)
	if tr.EndBit != 12 || tr.Gate || br.ReadBits(8) != 0x2a {
		t.Errorf("ti=47 par porte : fin %d (porte %v), attendu 12 sans porte lue", tr.EndBit, tr.Gate)
	}
	avecPorte := func(w *bitw) { etat(w); w.put(0, 1); w.put(0, 4) }
	if tr, br = leRecordNeuf(47, false, false, avecPorte); tr.EndBit != 17 || br.ReadBits(8) != 0x2a {
		t.Errorf("ti=47 sans lecture par porte : fin %d, attendu 17 (porte et masque lus)", tr.EndBit)
	}
	// ti=11 (version seule) garde sa porte sous la lecture par porte : son masque de creation n est pas vide.
	if tr, _ = leRecordNeuf(11, true, false, func(w *bitw) { w.put(0, 1); w.put(0, 1) }); tr.EndBit != 8 {
		t.Errorf("ti=11 par porte : fin %d, attendu 8 (porte lue)", tr.EndBit)
	}
}

// TestLEtatDeLaProprieteGereeALaCreationSuitLeBuild : ti=13 lit, dans un record NEW, la charge de ses
// etiquettes quand le profil l etablit, leurs etiquettes seules sinon. MUTATION — la bascule ignoree dans
// [consumeDefaultStateTI13Creation] : ROUGE.
func TestLEtatDeLaProprieteGereeALaCreationSuitLeBuild(t *testing.T) {
	etat := func(w *bitw) {
		w.put(0, 1)           // version absente
		w.put(0x20010f22, 32) // "propertyName"
		w.put(0, 1)           // un variant, mode A
		w.put(ManagedPropertyTagQuant, 4)
	}
	// Complete : la charge R(24) de l etiquette se lit, puis aucune porte (masque de creation vide).
	complet := func(w *bitw) { etat(w); w.put(0x5a5a5a, 24) }
	if tr, br := leRecordNeuf(archetypeProprieteGeree, true, true, complet); tr.EndBit != 6+1+32+1+4+24 || br.ReadBits(8) != 0x2a {
		t.Errorf("etat complet : fin %d, attendu %d", tr.EndBit, 6+1+32+1+4+24)
	}
	// Etiquettes seules : la porte et le masque suivent l etiquette.
	seul := func(w *bitw) { etat(w); w.put(0, 1); w.put(0, 4) }
	if tr, br := leRecordNeuf(archetypeProprieteGeree, false, false, seul); tr.EndBit != 6+1+32+1+4+1+4 || br.ReadBits(8) != 0x2a {
		t.Errorf("etiquettes seules : fin %d, attendu %d", tr.EndBit, 6+1+32+1+4+1+4)
	}
}

// ecrireChargeMesuree ecrit, apres l en-tete du message de genre `genre`, sa charge de disposition mesuree.
func (w *bitWriter) ecrireChargeMesuree(genre uint64) {
	switch genre {
	case 12: // deux joueurs, un tag, une position (porte posee, niveau 0xc), la direction
		w.bit(0)
		w.bits(7, 5)
		w.bit(0)
		w.bits(1, 5)
		w.bit(1)
		w.bits(0, 32)
		w.ecrirePositionDeNiveau(true, 0, profile.LargeursAxeParDefautDuBuild(0xc))
		w.bits(0, 19)
	case 14:
		w.bits(0, 32)
	case 106: // une position, une direction R(10), un index R(9)
		w.ecrirePositionDeNiveau(true, 0, profile.LargeursAxeParDefautDuBuild(0xc))
		w.bits(0, 10+9)
	}
}

// TestLesChargesMesureesNeSeLisentQueSousUnBuildQuiLesEtablit : la vue A d un film dont le build etablit les
// largeurs mesurees se lit jusqu a son terminateur au travers d un message de genre 12, 14 ou 106 ; sinon elle
// s arrete apres le genre de ce message, comme sur un genre dont la charge n est pas portee. MUTATION — la
// condition de [chargeDuGenre] retiree : ROUGE.
func TestLesChargesMesureesNeSeLisentQueSousUnBuildQuiLesEtablit(t *testing.T) {
	for _, genre := range []uint64{12, 14, 106} {
		var w bitWriter
		w.bit(1) // configuration
		w.ecrireEnTeteDeMessage(genre)
		w.ecrireChargeMesuree(genre)
		w.bit(0) // terminateur
		fin := w.n
		w.bits(0x3f, 6)
		g := grammaireRecente()
		g.chargesMesurees = true
		if a := lireSous(w.buf, g); !a.Porte || a.Fin != fin || !slices.Equal(a.Genres, []int{int(genre)}) {
			t.Errorf("genre %d sous un build qui l etablit : %+v, attendu portee jusqu au bit %d", genre, a, fin)
		}
		g.chargesMesurees = false
		if a := lireSous(w.buf, g); a.Porte || a.Fin != 9 || !slices.Equal(a.Genres, []int{int(genre)}) {
			t.Errorf("genre %d sous un build qui ne l etablit pas : %+v, attendu arretee au bit 9", genre, a)
		}
	}
}

// TestLeProfilDuBuildPoseLesTroisEcritures : la grammaire que le profil d un film rend porte les trois
// dispositions sur HI_1_11_0 a HI_1_13_0 et sur aucun autre build, un film sans section d identification compris.
// MUTATION — une ligne de [profile.EcritureDuBuildPour] retiree ou ajoutee : ROUGE.
func TestLeProfilDuBuildPoseLesTroisEcritures(t *testing.T) {
	for _, c := range []struct {
		build   string
		attendu bool
	}{
		{"HI_1_13_0", true}, {"HI_1_12_0", true}, {"HI_1_11_0", true},
		{"HI_1_10_0", false}, {"HI_1_9_0", false}, {"HI_1_8_0", false}, {"HI_1_4_1", false}, {"HI_1_14_0", false}, {"", false},
	} {
		p := profilDIdentite(profile.FilmIdentity{Build: c.build}, nil)
		g, _ := grammaireSousFilm(ProfilDeBalayageParDefaut().Grammaire, p)
		if g.CreationParPorte != c.attendu || g.ProprieteGereeComplete != c.attendu ||
			grammaireDeLaVueASousFilm(p).chargesMesurees != c.attendu {
			t.Errorf("build %q : creation par porte %v, propriete %v, charges mesurees %v, attendu %v", c.build,
				g.CreationParPorte, g.ProprieteGereeComplete, grammaireDeLaVueASousFilm(p).chargesMesurees, c.attendu)
		}
	}
	sans := profile.Resoudre(profile.ClesDuFilm{}, nil)
	if g, _ := grammaireSousFilm(ProfilDeBalayageParDefaut().Grammaire, sans); g.CreationParPorte || g.ProprieteGereeComplete {
		t.Errorf("film sans section d identification : %+v, attendu aucune ecriture", g)
	}
}

// TestLeContexteDeFilmRendLesEcrituresDeSonBuild : le profil de balayage d un contexte de film porte les
// dispositions de son build, sur une bobine de chaque build du depot, et un profil pose par-dessus (celui de
// `killsource`) ne les efface pas.
func TestLeContexteDeFilmRendLesEcrituresDeSonBuild(t *testing.T) {
	for _, c := range []struct {
		film    string
		attendu bool
	}{
		{"fb1a1a72", true}, {"bcb6d393", true}, {"e5adf7b2", true},
		{"111fa685", false}, {"11de8353", false}, {"60ae07c4", false}, {"a521164d", false},
	} {
		fc := NewFilmContext(bobineFilm(t, c.film))
		if g := fc.ProfilDeBalayage().Grammaire; g.CreationParPorte != c.attendu || g.ProprieteGereeComplete != c.attendu {
			t.Errorf("bobine %s : %+v, attendu %v", c.film, g, c.attendu)
		}
		fc.PoserProfilDeBalayage(ProfilDeBalayageParDefaut())
		if g := fc.ProfilDeBalayage().Grammaire; g.CreationParPorte != c.attendu {
			t.Errorf("bobine %s sous un profil pose : creation par porte %v, attendu %v", c.film, g.CreationParPorte, c.attendu)
		}
	}
}

// TestLaListeDesMasquesDeCreationVides epingle la liste entiere de [masqueDeCreationVide] : une porte lue pour un
// archetype qui n en ecrit pas, ou sautee pour un archetype qui en ecrit une, decale tout le reste du paquet sans
// que rien ne le signale. Les archetypes 8 et 31 gardent leur porte.
func TestLaListeDesMasquesDeCreationVides(t *testing.T) {
	vides := map[uint32]bool{7: true, 13: true, 14: true, 20: true, 23: true, 28: true, 29: true, 46: true, 47: true, 49: true}
	for ti := uint32(0); ti < 64; ti++ {
		if got := masqueDeCreationVide(ti); got != vides[ti] {
			t.Errorf("masqueDeCreationVide(%d) = %v, attendu %v", ti, got, vides[ti])
		}
	}
}
