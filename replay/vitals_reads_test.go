package replay

import (
	"reflect"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// axeDeVitalites : origine 1 s, pas 100 ms, 20 frames (fin a 3 s).
var axeDeVitalites = axeDeFrames{origin: 1_000_000, step: 100_000, frames: 20}

// lecture rend une lecture d etat complet du slot 3, generation 1, a l instant `us`.
func vitLecture(us uint64, sante, bouclier float32) grammar.LectureVitale {
	return grammar.LectureVitale{TS: us, Slot: 3, Gen: 1, ASante: true, ABouclier: true, Sante: sante, Bouclier: bouclier}
}

func vitNaissance(us uint64) grammar.LectureVitale {
	return grammar.LectureVitale{TS: us, Slot: 3, Gen: 1, Creation: true}
}

// vitalsDe fait passer des lectures par la reduction a la source puis par le calque.
func vitalsDe(t *testing.T, lectures []grammar.LectureVitale, degats []grammar.DegatLu, rampe bool) ([]Vital, VitalCoverage) {
	t.Helper()
	reads, st := reduireLesVitalites(lectures)
	got, cov := buildVitals(VitalsInput{Reads: reads, Stats: st}, degats, map[uint32]bool{3: true}, axeDeVitalites, rampe)
	if cov.Read != cov.Unchanged+cov.BeforeOrigin+cov.AfterEnd+cov.NoTrack+cov.AfterDeath+cov.Ramp+cov.Published {
		t.Errorf("l identite de la couverture ne tient pas : %+v", cov)
	}
	return got, cov
}

// uneRampe rend `n` lectures de recharge : un cran de bouclier toutes les 17 ms depuis `bouclier0` crans, sante pleine.
func uneRampe(depuis uint64, bouclier0, n int) []grammar.LectureVitale {
	var out []grammar.LectureVitale
	for k := range n {
		out = append(out, vitLecture(depuis+uint64(k)*17_000, 1, float32(bouclier0+k)/64))
	}
	return out
}

// TestLesVitalesPublientChaqueChangementEtJamaisUneValeurDejaConnue : un changement de bouclier, un changement de
// sante, une lecture inchangee (jamais republiee), une lecture partielle completee de l autre composant ; la vie nait
// pleine, aucune ligne ne la marque.
// MUTATIONS : comparer un seul composant, ne pas completer une lecture partielle, publier la naissance -> ROUGE.
func TestLesVitalesPublientChaqueChangementEtJamaisUneValeurDejaConnue(t *testing.T) {
	lectures := []grammar.LectureVitale{
		vitNaissance(1_000_000),
		vitLecture(1_100_000, 1, 0.5),                                     // bouclier : changement
		vitLecture(1_200_000, 1, 0.5),                                     // inchangee
		vitLecture(1_300_000, 0.75, 0.5),                                  // sante : changement
		{TS: 1_400_000, Slot: 3, Gen: 1, ABouclier: true, Bouclier: 0.25}, // partielle : la sante tenue reste 0.75
		{TS: 1_500_000, Slot: 3, Gen: 1, ASante: true, Sante: -0.4},       // zone negative : repliee sur 0
	}
	got, cov := vitalsDe(t, lectures, nil, true)
	want := []Vital{
		{T: 1, Ms: 100, Slot: 3, Sh: 0.5, Hp: 1},
		{T: 3, Ms: 300, Slot: 3, Sh: 0.5, Hp: 0.75},
		{T: 4, Ms: 400, Slot: 3, Sh: 0.25, Hp: 0.75},
		{T: 5, Ms: 500, Slot: 3, Sh: 0.25, Hp: 0, Dead: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lignes %+v, attendu %+v", got, want)
	}
	if cov.Read != 5 || cov.Births != 1 || cov.Unchanged != 1 || cov.Published != 4 || cov.Deaths != 1 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestUneRampeDeRechargeNeGardeQueSesDeuxBouts : vingt crans, une ligne chacun sans la regle, trois avec elle ; une
// interpolation entre les lignes gardees retrouve chaque ligne ecartee a un cran pres.
// MUTATIONS : retirer le test de la droite, ecarter la derniere ligne de la vie -> ROUGE.
func TestUneRampeDeRechargeNeGardeQueSesDeuxBouts(t *testing.T) {
	lectures := append([]grammar.LectureVitale{vitLecture(1_000_000, 1, 0.2)}, uneRampe(1_100_000, 13, 20)...)
	sans, covSans := vitalsDe(t, lectures, nil, false)
	avec, covAvec := vitalsDe(t, lectures, nil, true)
	if len(sans) != 21 || covSans.Ramp != 0 {
		t.Fatalf("sans la regle : %d ligne(s), couverture %+v", len(sans), covSans)
	}
	if len(avec) != 3 || covAvec.Ramp != 18 {
		t.Fatalf("avec la regle : %d ligne(s), couverture %+v", len(avec), covAvec)
	}
	if avec[len(avec)-1] != sans[len(sans)-1] || avec[0] != sans[0] {
		t.Errorf("les bouts ne sont pas ceux de la serie entiere : %+v", avec)
	}
	// chaque ligne ecartee se retrouve, a un cran pres, par interpolation entre les deux lignes gardees qui l encadrent
	for _, ecrite := range sans {
		i := 0
		for i+1 < len(avec) && avec[i+1].Ms <= ecrite.Ms {
			i++
		}
		if avec[i].Ms == ecrite.Ms || i+1 == len(avec) {
			continue
		}
		g, d := avec[i], avec[i+1]
		f := float64(ecrite.Ms-g.Ms) / float64(d.Ms-g.Ms)
		v := float64(g.Sh) + f*float64(d.Sh-g.Sh)
		if e := v - float64(ecrite.Sh); e > crandDeBouclier+0.001 || e < -crandDeBouclier-0.001 {
			t.Errorf("ligne a %d ms : interpolee %.3f, ecrite %.3f", ecrite.Ms, v, ecrite.Sh)
		}
	}
}

// TestUnDegatPendantUneRampeGardeSonInstantEtLaValeurAvant : le coup tombe au milieu de la rampe ; la ligne a son
// instant et la derniere ligne strictement avant restent, donc l etat avant et l etat a l instant sont exacts.
// MUTATIONS : ne pas proteger la ligne avant, ne pas proteger la ligne a l instant -> ROUGE.
func TestUnDegatPendantUneRampeGardeSonInstantEtLaValeurAvant(t *testing.T) {
	// la vie est le slot 517 : l unite d un message de degat est le slot moins 512.
	var lectures []grammar.LectureVitale
	for _, l := range uneRampe(1_100_000, 13, 20) {
		l.Slot = 517
		lectures = append(lectures, l)
	}
	coup := lectures[9].TS
	degats := []grammar.DegatLu{{TS: coup, AVictime: true, Victime: 5, GenVictime: 1}}
	reads, st := reduireLesVitalites(lectures)
	got, cov := buildVitals(VitalsInput{Reads: reads, Stats: st}, degats, map[uint32]bool{517: true}, axeDeVitalites, true)
	garde := map[int]bool{}
	for _, v := range got {
		garde[v.Ms] = true
	}
	ms := func(i int) int { return axeDeVitalites.millisecondes(lectures[i].TS) }
	for _, i := range []int{0, 8, 9, 19} {
		if !garde[ms(i)] {
			t.Errorf("la ligne %d (%d ms) devrait rester : %+v", i, ms(i), got)
		}
	}
	if cov.Ramp == 0 {
		t.Errorf("les deux moities de la rampe se reduisent encore : %+v", cov)
	}
}

// TestLaMortPorteSaDerniereSanteEtFermeLaVie : un coup vide bouclier et sante d un seul trait, la sante a zero est
// ecrite apres ; les lectures suivantes de la vie sont hors vie ; une naissance de la meme vie la rouvre.
// MUTATIONS : couper la vie a la fin de piste, ne pas marquer la mort, compter apres la mort -> ROUGE.
func TestLaMortPorteSaDerniereSanteEtFermeLaVie(t *testing.T) {
	lectures := []grammar.LectureVitale{
		vitLecture(1_100_000, 0.6, 1),
		vitLecture(1_500_000, 0, 0), // la mort : le bouclier et la sante tombent ensemble
		vitLecture(1_700_000, 0, 0.3),
	}
	got, cov := vitalsDe(t, lectures, nil, true)
	want := []Vital{{T: 1, Ms: 100, Slot: 3, Sh: 1, Hp: 0.6}, {T: 5, Ms: 500, Slot: 3, Sh: 0, Hp: 0, Dead: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lignes %+v, attendu %+v", got, want)
	}
	if cov.AfterDeath != 1 || cov.Deaths != 1 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestLesLecturesHorsVieSeCompteNtEtNeSePublientPas : avant l origine, apres la fin, sans piste, apres la mort.
// MUTATIONS : retirer une des quatre gardes -> ROUGE.
func TestLesLecturesHorsVieSeCompteNtEtNeSePublientPas(t *testing.T) {
	sansPiste := vitLecture(1_200_000, 1, 0.5)
	sansPiste.Slot = 9
	lectures := []grammar.LectureVitale{
		vitLecture(900_000, 1, 0.9),   // avant l origine
		vitLecture(1_100_000, 1, 0.5), // publiee
		sansPiste,                     // sans piste
		vitLecture(9_000_000, 1, 0.1), // apres la fin de l axe
	}
	got, cov := vitalsDe(t, lectures, nil, true)
	if len(got) != 1 || got[0].Sh != 0.5 {
		t.Errorf("lignes %+v", got)
	}
	if cov.BeforeOrigin != 1 || cov.NoTrack != 1 || cov.AfterEnd != 1 || cov.Published != 1 {
		t.Errorf("couverture %+v", cov)
	}
}

// TestDeuxViesDUnSlotRecycleNeSeMelentPas : la vie de generation 2 part pleine, meme si la generation 1 est a moitie.
// MUTATION : cle de vie sans generation -> ROUGE.
func TestDeuxViesDUnSlotRecycleNeSeMelentPas(t *testing.T) {
	lectures := []grammar.LectureVitale{vitLecture(1_100_000, 1, 0.5)}
	autre := vitLecture(1_200_000, 1, 1) // generation 2, bouclier plein : c est son etat de naissance, pas un changement
	autre.Gen = 2
	lectures = append(lectures, autre)
	got, cov := vitalsDe(t, lectures, nil, true)
	if len(got) != 1 || cov.Unchanged != 1 {
		t.Errorf("lignes %+v, couverture %+v", got, cov)
	}
}

// TestLOrdreDesLecturesNeChangePasLeCalque : les memes lectures a l envers rendent les memes lignes.
func TestLOrdreDesLecturesNeChangePasLeCalque(t *testing.T) {
	lectures := append([]grammar.LectureVitale{vitLecture(1_000_000, 1, 0.2)}, uneRampe(1_100_000, 13, 12)...)
	reads, st := reduireLesVitalites(lectures)
	a, _ := buildVitals(VitalsInput{Reads: reads, Stats: st}, nil, map[uint32]bool{3: true}, axeDeVitalites, true)
	for i, j := 0, len(reads)-1; i < j; i, j = i+1, j-1 {
		reads[i], reads[j] = reads[j], reads[i]
	}
	b, _ := buildVitals(VitalsInput{Reads: reads, Stats: st}, nil, map[uint32]bool{3: true}, axeDeVitalites, true)
	if !reflect.DeepEqual(a, b) {
		t.Errorf("l ordre des entrees change le calque :\n%+v\n%+v", a, b)
	}
}

// TestLesVitalitesVoyagentDansLesFaits : l ecriture puis la relecture rendent les memes lectures et les memes comptes.
func TestLesVitalitesVoyagentDansLesFaits(t *testing.T) {
	lectures := append([]grammar.LectureVitale{vitNaissance(1_000_000), vitLecture(1_050_000, 0.5, 0.2)}, uneRampe(1_100_000, 13, 5)...)
	reads, st := reduireLesVitalites(lectures)
	w := &gwriter{}
	encodeVitalites(w, reads, st)
	r := &greader{b: w.b}
	got, gst := decodeVitalites(r)
	if r.err != nil || !reflect.DeepEqual(got, reads) || gst != st {
		t.Errorf("relecture : err=%v lectures %+v (attendu %+v), comptes %+v (attendu %+v)", r.err, got, reads, gst, st)
	}
}

// TestLaPoseDesVitalitesSuitLaMarche : une marche non jouee ne publie ni calque ni couverture.
func TestLaPoseDesVitalitesSuitLaMarche(t *testing.T) {
	reads, st := reduireLesVitalites([]grammar.LectureVitale{vitLecture(1_100_000, 1, 0.5)})
	poser := func(in VitalsInput) *assemblage {
		a := &assemblage{ctx: t.Context(), origin: axeDeVitalites.origin, step: axeDeVitalites.step}
		a.doc.FrameCount = axeDeVitalites.frames
		a.doc.Coverage = &Coverage{}
		a.doc.Tracks = []Track{{Slot: 3}}
		a.opt.Vitals = in
		a.poserLesVitalites()
		return a
	}
	if a := poser(VitalsInput{}); a.doc.Vitals != nil || a.doc.Coverage.Vitals != nil {
		t.Error("marche non jouee : ni calque ni couverture")
	}
	if a := poser(VitalsInput{Reads: reads, Stats: st}); len(a.doc.Vitals) != 1 || a.doc.Coverage.Vitals == nil {
		t.Errorf("marche jouee : %d ligne(s)", len(a.doc.Vitals))
	}
}
