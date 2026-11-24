package replay

// flag_return_gauge_equipe_test.go — LA JAUGE DE RETOUR D UNE EQUIPE QUI A PLUSIEURS DRAPEAUX.
//
// Le film porte une jauge par equipe, pas par drapeau : la serie d un slot sert tour a tour les drapeaux de l equipe qui sont au sol.
// Sa part dans les lachers d UN drapeau ne passe pas le seuil, sa part dans l UNION des lachers de l equipe le passe.

import (
	"context"
	"testing"
)

// frgDeuxDrapeauxDUneEquipe : deux drapeaux de l equipe 0, au sol a tour de role ([20, 40] puis [60, 80]).
func frgDeuxDrapeauxDUneEquipe() []FlagCarry {
	return []FlagCarry{frgCarry(0, 20, 40), frgCarry(0, 60, 80)}
}

// TestLaJaugeDUneEquipeServiraTourATourSesDrapeaux : un slot lu dans les lachers de deux drapeaux de la meme equipe (50 % dans chacun,
// 100 % dans leur union) est apparie a l equipe, et chaque point va au drapeau qui est au sol a son instant.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : calculer la part drapeau par drapeau ; poser toute la serie sur chaque drapeau.
func TestLaJaugeDUneEquipeServiraTourATourSesDrapeaux(t *testing.T) {
	carries := frgDeuxDrapeauxDUneEquipe()
	reads := append(frgSerie(1639, 20, 40, 0), frgSerie(1639, 60, 80, 0)...)
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	premier, second := carries[0].Spans[1].ReturnProgress, carries[1].Spans[1].ReturnProgress
	if len(premier) == 0 || len(second) == 0 {
		t.Fatalf("un drapeau n a pas sa serie : %d / %d points (couverture %+v)", len(premier), len(second), cov)
	}
	if premier[len(premier)-1].T > 40 || second[0].T < 60 {
		t.Errorf("la serie d un drapeau deborde sur le lacher de l autre : %v / %v", premier, second)
	}
	if cov.GaugePaired != 2 || cov.GaugeSpans != 2 || cov.GaugeReadsAmbiguous != 0 || cov.GaugeSlotsOffDrops != 0 {
		t.Errorf("couverture = apparies %d, lachers %d, ambigus %d, slots hors lachers %d", cov.GaugePaired, cov.GaugeSpans,
			cov.GaugeReadsAmbiguous, cov.GaugeSlotsOffDrops)
	}
	// Le document publie : les deux drapeaux portent leur serie, et la couverture dit ce qui a ete lu.
	doc := &ReplayDocument{Coverage: &Coverage{}}
	attachFlagLayer(context.Background(), doc, carries, &cov)
	pub := documentJSON(t, *doc)
	for i, c := range pub["flagCarries"].([]any) {
		lacher := c.(map[string]any)["spans"].([]any)[1].(map[string]any)
		if serie, _ := lacher["returnProgress"].([]any); len(serie) == 0 {
			t.Errorf("le drapeau %d publie un lacher sans jauge", i)
		}
	}
	couv := pub["coverage"].(map[string]any)["flagCarries"].(map[string]any)
	if couv["gaugePaired"] != float64(2) || couv["gaugeSlotsTeamTaken"] != float64(0) || couv["gaugeReadsUnpublished"] != float64(0) {
		t.Errorf("couverture publiee = %v", couv)
	}
}

// TestDeuxDrapeauxDeLEquipeAuSolEnMemeTempsNeRecoiventPasLeMemePoint : quand les deux drapeaux sont au sol a la meme frame, le film ne dit
// pas auquel va la jauge : le point n est a aucun, et il se compte.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : poser le point ambigu sur le premier drapeau ; ne pas le compter.
func TestDeuxDrapeauxDeLEquipeAuSolEnMemeTempsNeRecoiventPasLeMemePoint(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40), frgCarry(0, 30, 50)} // au sol ensemble sur [30, 40]
	reads := frgSerie(1639, 20, 50, 0)                               // 16 lectures, une toutes les deux frames
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	if cov.GaugeReadsAmbiguous != 6 { // les frames 30, 32, 34, 36, 38 et 40
		t.Errorf("lectures ambigues = %d, attendu 6", cov.GaugeReadsAmbiguous)
	}
	for _, p := range carries[0].Spans[1].ReturnProgress {
		if p.T >= 30 {
			t.Errorf("le premier drapeau recoit un point ambigu a la frame %d", p.T)
		}
	}
	for _, p := range carries[1].Spans[1].ReturnProgress {
		if p.T <= 40 {
			t.Errorf("le second drapeau recoit un point ambigu a la frame %d", p.T)
		}
	}
}

// TestUneSerieLueEtNonPublieeSeCompteSousSonMotif : un slot trop court, un slot qui ne suit aucun lacher et un second slot de la meme
// equipe se comptent chacun sous leur nom ; aucune serie lue ne disparait sans compteur.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas compter un motif ; confondre deux motifs.
func TestUneSerieLueEtNonPublieeSeCompteSousSonMotif(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40)}
	reads := frgSerie(1639, 20, 40, 0)                                          // la jauge : 11 lectures
	reads = append(reads, frgSerie(1640, 20, 38, 0)...)                         // meme equipe, part de 100 % mais moins bonne en nombre : prise
	reads = append(reads, frgLecture(1700, 22, 0.1), frgLecture(1700, 24, 0.2)) // trop court
	for i := range 10 {
		reads = append(reads, frgLecture(1800, 60+i, 0.5)) // aucun lacher
	}
	var cov FlagCarriesCoverage
	attachFlagReturnGauges(carries, frgInput(reads), frgClock(), &cov)
	if cov.GaugeSlotsFewReads != 1 || cov.GaugeSlotsOffDrops != 1 || cov.GaugeSlotsTeamTaken != 1 {
		t.Errorf("motifs = trop court %d, hors lachers %d, equipe prise %d, attendu 1/1/1 (%+v)",
			cov.GaugeSlotsFewReads, cov.GaugeSlotsOffDrops, cov.GaugeSlotsTeamTaken, cov)
	}
	if cov.GaugeReadsUnpublished != 10+2+10 {
		t.Errorf("lectures non publiees = %d, attendu 22", cov.GaugeReadsUnpublished)
	}
}

// TestToutEchantillonDeLaJaugeEstPoseOuCompte : pour une lecture, CHAQUE echantillon de chaque slot lu est soit pose sur un drapeau, soit
// compte sous son motif (slot trop court, hors lachers, equipe prise ; echantillon ambigu ou hors lachers d'un slot apparie). Deux equipes,
// un drapeau chacune pour l'une et deux drapeaux au sol ensemble un instant pour l'autre.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : un motif non compte, un echantillon compte deux fois, un echantillon perdu sans compteur.
func TestToutEchantillonDeLaJaugeEstPoseOuCompte(t *testing.T) {
	carries := []FlagCarry{frgCarry(0, 20, 40), frgCarry(0, 30, 50), frgCarry(1, 60, 80)}
	reads := frgSerie(1639, 18, 52, 0)                                          // equipe 0 : echantillons hors lachers aux deux bouts, ambigus sur [30, 40]
	reads = append(reads, frgSerie(1640, 19, 33, 0)...)                         // meme equipe, 0,875 : moins bonne part, prise
	reads = append(reads, frgSerie(1641, 60, 80, 0)...)                         // equipe 1
	reads = append(reads, frgLecture(1700, 22, 0.1), frgLecture(1700, 24, 0.2)) // trop court
	for i := range 10 {
		reads = append(reads, frgLecture(1800, 85+i, 0.5)) // aucun lacher
	}
	slots := flagGaugeSlotsOf(frgInput(reads).Gauge, frgClock())
	total := 0
	for _, s := range slots {
		total += len(s.samples)
	}
	bound, rej := bindFlagGauges(slots, carries, flagGroupsByTeam(carries))
	var cov FlagCarriesCoverage
	placees := 0
	for _, serie := range flagGaugeSeriesOf(bound, carries, &cov) {
		placees += len(serie)
	}
	if got := placees + rej.reads + cov.GaugeReadsOffDrops + cov.GaugeReadsAmbiguous; got != total {
		t.Errorf("echantillons lus %d, poses %d + non publies %d + hors lachers %d + ambigus %d = %d",
			total, placees, rej.reads, cov.GaugeReadsOffDrops, cov.GaugeReadsAmbiguous, got)
	}
	if rej.fewSlots != 1 || rej.takenSlots != 1 || rej.offSlots != 1 || cov.GaugeReadsAmbiguous == 0 || cov.GaugeReadsOffDrops == 0 || placees == 0 {
		t.Errorf("le cas ne couvre pas tous les motifs : %+v %+v poses=%d", rej, cov, placees)
	}
}
