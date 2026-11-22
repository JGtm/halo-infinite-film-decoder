package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// vue_a_tirs_test.go — LES TIRS DE LA VUE A, A TOUT RANG (la tete, mais aussi ce qui suit un autre message).

// tirCourtDeVueA ecrit un message de tir de forme courte : la continuation, le genre 36, la reference 0 (unite
// `unite`, generation 2, largeur 13) et les deux gardes suivantes fermees, puis la charge de `FUN_14080c1f8`.
func tirCourtDeVueA(w *bitWriter, unite, numero, tireur, armeHaute, armeBasse uint64) {
	w.bit(1)
	w.bits(TypeTirArme, 7)
	w.bit(1) // reference 0 presente
	w.bit(0) // sans sonde : treize bits
	w.bits(unite, 13)
	w.bits(2, 2)           // generation
	w.bits(0, 2)           // references 1 et 2 absentes
	w.bit(1)               // court
	w.bit(0)               // bloc
	w.bits(numero&0x7f, 7) // R(7) puis le bit de poids fort
	w.bit(numero >> 7)
	w.bit(0) // tireur present (polarite inversee)
	w.bits(tireur, 5)
	w.bit(1) // champ e absent
	w.bit(1) // moitie haute de l'arme presente
	w.bits(armeHaute, 32)
	w.bits(armeBasse, 32)
	w.bits(0, 2)
	w.bits(0x3ff, largeurTirCourt)
}

// TestLaVueARendLesTirsAApresLaTete : un tir ecrit APRES un autre message de la trame est un evenement de tir, avec
// les champs que sa charge ecrit ; le tir de tete aussi. Le meme message lu en tete par la grammaire de tete
// ([lireEnteteTir36]) rend les memes valeurs.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne recueillir que le tir de rang 0 ; lire le tireur sans la polarite
// inversee de sa garde ; omettre le bit de poids fort du numero.
func TestLaVueARendLesTirsAApresLaTete(t *testing.T) {
	var w bitWriter
	w.bit(1)       // bit de configuration
	w.bit(1)       // continuation
	w.bits(21, 7)  // unit_zoom
	w.bit(1)       // garde de la reference 0 : domaine 4, R(9) puis R(2)
	w.bits(0, 9+2) //
	w.bits(0, 2)   // gardes des references 1 et 2
	w.bits(0, 2)   // FUN_14080cb98
	tirCourtDeVueA(&w, 0x0F1, 0xB0, 19, 0x6ACDC44D, 0x42C9679F)
	tirCourtDeVueA(&w, 0x0F2, 0x03, 7, 0x121B4009, 0x42C9679F)
	w.bit(0) // terminateur
	w.bits(0, 16)
	a := lireSous(w.buf, grammaireRecente())
	if !a.Porte || len(a.Tirs) != 2 {
		t.Fatalf("vue lue jusqu au terminateur : %v, %d tirs ; attendu vrai et 2", a.Porte, len(a.Tirs))
	}
	for i, want := range []lecture.MessageDeTir{
		{Rang: 1, Unite: 0x0F1, Gen: 2, Numero: 0xB0, Tireur: 19, Drapeaux: lecture.TirUnite | lecture.TirCourt, Arme: 0x6ACDC44D42C9679F},
		{Rang: 2, Unite: 0x0F2, Gen: 2, Numero: 0x03, Tireur: 7, Drapeaux: lecture.TirUnite | lecture.TirCourt, Arme: 0x121B400942C9679F},
	} {
		if a.Tirs[i] != want {
			t.Errorf("tir %d : %+v, attendu %+v", i, a.Tirs[i], want)
		}
	}
}

// TestLeTirDeTeteDeLaVueAEstCeluiDeLaGrammaireDeTete : sur des enregistrements ecrits avec chaque combinaison de
// champs optionnels, le message de tete que la vue A recueille porte les valeurs que [decodeFireEvent] lit.
func TestLeTirDeTeteDeLaVueAEstCeluiDeLaGrammaireDeTete(t *testing.T) {
	const arme = uint64(0x6ACDC44D42C9679F)
	for _, o := range []modalHeaderOpts{
		{ref0: true, ref0Wide: true, ref0Index: 0x0F1, dExtra: true, tireur: 19, fWeapon: true, armeHaute: arme >> 32, armeBasse: arme & 0xFFFFFFFF, numero: 0x61},
		{ref0: true, ref0Wide: true, fWeapon: true, armeHaute: arme >> 32, armeBasse: arme & 0xFFFFFFFF},
		{ref0: true, ref1: true, ref2: true, dExtra: true, eExtra: true, fWeapon: true},
	} {
		pay := tirSynthetique(t, o)
		e, ok := decodeFireEvent(pay)
		a := lireSous(pay, grammaireRecente())
		if !ok || len(a.Tirs) == 0 {
			t.Fatalf("%+v : tete refusee (%v) ou aucun tir recueilli (%d)", o, ok, len(a.Tirs))
		}
		m := a.Tirs[0]
		if m.Rang != 0 || m.Arme != e.WeaponID || m.Numero != e.FireNumber || (m.Tireur >= 0) != e.HasShooter ||
			(m.Tireur >= 0 && int(m.Tireur) != e.FilmIndex) || (m.Drapeaux&lecture.TirUnite != 0) != e.Unit.Present ||
			(e.Unit.Present && parentHandleBase+uint32(m.Unite) != e.Unit.Slot) {
			t.Errorf("%+v : la vue A lit %+v, la grammaire de tete %+v", o, m, e)
		}
	}
}

// TestLeCanalDesTirsCompteLeTirQuiSuitUnAutreMessage : une trame dont la tete est un zoom puis un tir donne UN
// evenement de tir, date du paquet, sans visee de record ; une trame dont la tete est un tir suivi d un second
// en donne deux, dans l ordre.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : ne lire que le message de tete.
func TestLeCanalDesTirsCompteLeTirQuiSuitUnAutreMessage(t *testing.T) {
	var w bitWriter
	w.bit(1)
	w.bit(1)
	w.bits(21, 7)
	w.bit(1)
	w.bits(0, 9+2)
	w.bits(0, 2)
	w.bits(0, 2)
	tirCourtDeVueA(&w, 0x0F1, 0x05, 4, 0x6ACDC44D, 0x42C9679F)
	tirCourtDeVueA(&w, 0x0F1, 0x06, 4, 0x6ACDC44D, 0x42C9679F)
	w.bit(0)
	w.bits(0, 16)
	var p lecture.Paquet
	p.Chunk, p.Index, p.TS, p.Payload = 3, 17, 123456, w.buf
	rangerLaTete(&p, ProfilDeBalayageParDefaut(), grammaireRecente())
	c := &canalDesTirs{}
	c.Tete(&p)
	if len(c.out) != 2 {
		t.Fatalf("%d evenements de tir, attendu 2 (la tete est un zoom, deux tirs la suivent)", len(c.out))
	}
	for i, e := range c.out {
		if e.Chunk != 3 || e.PacketIndex != 17 || e.TimestampUS != 123456 || !e.HasShooter || e.FilmIndex != 4 || e.FireNumber != uint8(5+i) || e.HasAim || !e.Unit.Present || e.Unit.Slot != parentHandleBase+0x0F1 {
			t.Errorf("evenement %d : %+v", i, e)
		}
	}
}
