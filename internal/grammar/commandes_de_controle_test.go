package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// commandes_de_controle_test.go — LA COMMANDE DE DEPLACEMENT TENUE, pliee des verdicts de vue C paquet par
// paquet comme le tir continu.

// entreeDeCommande rend l entree d une place qui porte le bloc de commande, avec son couple analogique.
func entreeDeCommande(index int, analogique [2]int) EntreeDeControle {
	return EntreeDeControle{Index: index, Bloc: true, Action: BlocDAction{Present: true,
		Arme: [2]int{ArmeAbsente, ArmeAbsente}},
		Champs: ChampsDeControle{Cdc04: ChampDeControleAbsent, Court: ChampDeControleAbsent,
			Troisieme: ChampDeControleAbsent, Champ10: ChampDeControleAbsent, Drapeaux: ChampDeControleAbsent,
			Analogique: analogique}}
}

var (
	zeroAnalogique = [2]int{scalaireAnalogiqueNul, scalaireAnalogiqueNul}
	avantAnalog    = [2]int{scalaireAnalogiqueNul, 50}
)

// plierLesCommandes joue un scenario dans le collecteur et rend les commandes tenues et les compteurs.
func plierLesCommandes(ps []paquetTC) ([]types.ControlHold, types.ControlHoldStats, types.ContinuousFireStats) {
	var st types.ContinuousFireStats
	c := nouveauCollecteurTirContinu(&st)
	for _, p := range ps {
		if !p.trou {
			c.recevoir(LectureVueC{Atteinte: true, Fermee: true, Entrees: p.entrees})
		}
		c.Trame(&lecture.Paquet{TS: p.ts, Debut: lecture.DebutEnTete})
	}
	c.Clore(BilanDeMarche{})
	return c.commandes, c.commandeStats, st
}

// TestLaCommandeDeDeplacementEstNonNulleHorsDuZeroExact : le deplacement est tenu des qu un des deux scalaires
// quitte `0x1f`, et se relache quand les deux y reviennent. MUTATIONS : le zero lu a 0 au lieu de `0x1f`, un
// seul des deux scalaires lu -> ROUGE.
func TestLaCommandeDeDeplacementEstNonNulleHorsDuZeroExact(t *testing.T) {
	hs, st, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeCommande(1, zeroAnalogique)}},
		{ts: 20, entrees: []EntreeDeControle{entreeDeCommande(1, avantAnalog)}},
		{ts: 30, entrees: []EntreeDeControle{entreeDeCommande(1, [2]int{0, scalaireAnalogiqueNul})}},
		{ts: 40, entrees: []EntreeDeControle{entreeDeCommande(1, zeroAnalogique)}},
	})
	if len(hs) != 1 || hs[0].FilmIndex != 1 || hs[0].StartUS != 20 || hs[0].EndUS != 40 || hs[0].Entries != 2 ||
		hs[0].StartBound != types.ContinuousFireBoundPressed || hs[0].EndBound != types.ContinuousFireBoundReleased {
		t.Fatalf("commandes %+v, attendu une commande du joueur 1 sur [20, 40), 2 entrees, posee puis relachee", hs)
	}
	if st.Entries != 4 || st.Move.Holds != 1 {
		t.Errorf("compteurs %+v", st)
	}
	if (ChampsDeControle{Analogique: [2]int{ChampDeControleAbsent, ChampDeControleAbsent}}).deplacementNonNul() {
		t.Error("un couple absent du flux ne porte pas de commande")
	}
}

// TestLesTrousDeLaCommandeSontPortesSansToucherAuTirContinu : une commande tenue avant et apres un trou le
// porte comme passage interieur ; l entree sans bloc ne dit rien ; le temps de tir tu ne bouge pas.
func TestLesTrousDeLaCommandeSontPortesSansToucherAuTirContinu(t *testing.T) {
	sansBloc := EntreeDeControle{Index: 2}
	hs, st, tir := plierLesCommandes([]paquetTC{
		{ts: 1_000, entrees: []EntreeDeControle{entreeDeCommande(2, avantAnalog)}},
		{ts: 2_000, trou: true},
		{ts: 3_000, entrees: []EntreeDeControle{sansBloc}},
		{ts: 4_000, entrees: []EntreeDeControle{entreeDeCommande(2, avantAnalog)}},
		{ts: 5_000, entrees: []EntreeDeControle{entreeDeCommande(2, zeroAnalogique)}},
	})
	if len(hs) != 1 || len(hs[0].Holes) != 1 || hs[0].Holes[0].StartUS != 2_000 || hs[0].Holes[0].EndUS != 4_000 {
		t.Fatalf("commandes %+v, attendu un passage interieur [2000, 4000)", hs)
	}
	if st.Move.WithHole != 1 || st.Move.InnerHoles != 1 {
		t.Errorf("compteurs %+v", st)
	}
	if tir.HeldHoleMS != 0 || tir.Bursts != 0 {
		t.Errorf("le tir continu doit rester intact : %+v", tir)
	}
}

// TestUneCommandePoseeApresUnTrouCommenceSurLEntree : l etat du joueur est inconnu apres un trou ; une commande
// posee sur la premiere entree lue qui suit commence avec la borne « trou ». MUTATION : lever l inconnu du
// joueur avant de plier la commande -> ROUGE.
func TestUneCommandePoseeApresUnTrouCommenceSurLEntree(t *testing.T) {
	hs, _, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeCommande(3, zeroAnalogique)}},
		{ts: 20, trou: true},
		{ts: 30, entrees: []EntreeDeControle{entreeDeCommande(3, avantAnalog)}},
		{ts: 40, entrees: []EntreeDeControle{entreeDeCommande(3, zeroAnalogique)}},
	})
	if len(hs) != 1 || hs[0].StartUS != 30 || hs[0].StartBound != types.ContinuousFireBoundHole {
		t.Fatalf("commandes %+v, attendu une commande ouverte a 30 (trou)", hs)
	}
}

// TestUnLacherDansLeTrouFinitLaCommandeAuTrou : l entree lue apres le trou ne tient plus la commande : le
// lacher est DANS le trou, la commande finit au premier paquet non lu.
func TestUnLacherDansLeTrouFinitLaCommandeAuTrou(t *testing.T) {
	hs, _, _ := plierLesCommandes([]paquetTC{
		{ts: 10, entrees: []EntreeDeControle{entreeDeCommande(3, avantAnalog)}},
		{ts: 20, trou: true},
		{ts: 30, entrees: []EntreeDeControle{entreeDeCommande(3, zeroAnalogique)}},
	})
	if len(hs) != 1 || hs[0].EndUS != 20 || hs[0].EndBound != types.ContinuousFireBoundHole {
		t.Fatalf("commandes %+v, attendu une commande finie a 20 (trou)", hs)
	}
}
