package replay

// projectiles_plein_debit_test.go — LE VOL GARDE SES ECHANTILLONS NATIFS ET SA HAUTEUR, SUR DEMANDE.
//
// La grille du rejeu retient UN point par frame : c est `p`, la vue que tous les lecteurs existants
// dessinent. `z` en porte la hauteur, au meme indice ; `fullRate` porte la MEME trajectoire au debit du
// film, chaque echantillon lu avant le premier pas impossible, a son instant. Ces tests tiennent trois
// proprietes : option eteinte rien ne change, option allumee `p` ne change pas, et la coupure au pas
// impossible borne les DEUX series.

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	origineVol = 1_000_000
	pasVolUS   = 33_000
)

// vol60Hz fabrique une piste de projectile a son debit reel : un echantillon tous les 16 700 us, a 2 m
// par record (120 m/s, sous la coupure de 200 m/s) sur X, hauteur 10 + 0,5 par record.
func vol60Hz(n int, repos bool) types.ProjectileTrack {
	pts := make([]types.ProjectileSample, n)
	for i := range pts {
		pts[i] = types.ProjectileSample{
			TimestampUS: origineVol + uint64(i)*16_700, X: float32(i) * 2, Y: 1.25, Z: 10 + float32(i)*0.5,
		}
	}
	pts[n-1].AtRest = repos
	return types.ProjectileTrack{Slot: 4096, Gen: 1, Pts: pts}
}

// volsAvecEtSansPleineCadence construit les projectiles des pistes, puis pose la pleine cadence sur une
// seconde construction : les deux resultats d une meme entree, comme le fait l assemblage.
func volsAvecEtSansPleineCadence(pistes []types.ProjectileTrack) (eteint, allume []Projectile, echantillons int) {
	eteint, _, _ = buildProjectiles(pistes, origineVol, pasVolUS, nil, nil)
	allume, par, _ := buildProjectiles(pistes, origineVol, pasVolUS, nil, nil)
	echantillons = poserLaPleineCadenceDesProjectiles(allume, pistes, par, origineVol, pasVolUS)
	return eteint, allume, echantillons
}

func sommeCumulee32(col []int32) []int32 {
	out := make([]int32, len(col))
	var acc int32
	for i, v := range col {
		acc += v
		out[i] = acc
	}
	return out
}

func sommeCumulee64(col []int64) []int64 {
	out := make([]int64, len(col))
	var acc int64
	for i, v := range col {
		acc += v
		out[i] = acc
	}
	return out
}

// TestVolEteintNePublieNiHauteurNiPleineCadence : sans la demande de l hote, un projectile n a ni `z`
// ni `fullRate`, et la serialisation n en contient pas la balise ; la couverture non plus.
func TestVolEteintNePublieNiHauteurNiPleineCadence(t *testing.T) {
	out, _, _ := buildProjectiles([]types.ProjectileTrack{vol60Hz(15, true)}, origineVol, pasVolUS, nil, nil)
	if len(out) != 1 || out[0].Z != nil || out[0].FullRate != nil {
		t.Fatalf("projectile sans option : %d trajectoire(s) %+v", len(out), out)
	}
	blob, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, balise := range []string{`"z"`, `"fullRate"`} {
		if strings.Contains(string(blob), balise) {
			t.Errorf("la serialisation sans option contient %s : %s", balise, blob)
		}
	}
	blob, _ = json.Marshal(ProjectileCoverage{Tracks: 1, Published: 1})
	if strings.Contains(string(blob), "fullRateSamples") {
		t.Errorf("la couverture sans option publie son compteur : %s", blob)
	}
}

// TestVolGardeSesEchantillonsAuDebitDuFilm : quinze echantillons a ~60 Hz (250 ms) : la grille de 33 ms
// y retient 8 points, la serie native en garde QUINZE, chacun a son instant, a son centieme.
func TestVolGardeSesEchantillonsAuDebitDuFilm(t *testing.T) {
	const n = 15
	pistes := []types.ProjectileTrack{vol60Hz(n, true)}
	eteint, allume, echantillons := volsAvecEtSansPleineCadence(pistes)
	if len(allume) != 1 || echantillons != n {
		t.Fatalf("%d trajectoire(s), %d echantillons de pleine cadence ; attendu 1 et %d", len(allume), echantillons, n)
	}
	p, fr := allume[0], allume[0].FullRate
	if len(p.P) != 8 {
		t.Fatalf("la vue grille retient un point par frame (8), obtenu %d", len(p.P))
	}
	if fr == nil || len(fr.T) != n || len(fr.X) != n || len(fr.Y) != n || len(fr.Z) != n {
		t.Fatalf("la serie native porte les %d echantillons lus, en quatre colonnes de meme longueur : %+v", n, fr)
	}
	ms, x, y, z := sommeCumulee64(fr.T), sommeCumulee32(fr.X), sommeCumulee32(fr.Y), sommeCumulee32(fr.Z)
	for i, s := range pistes[0].Pts {
		wantMS := int64((s.TimestampUS - origineVol) / fullRateTimeUnitUS)
		if ms[i] != wantMS || x[i] != centiemes(s.X) || y[i] != centiemes(s.Y) || z[i] != centiemes(s.Z) {
			t.Fatalf("echantillon %d : (t=%d, %d, %d, %d), attendu (t=%d, %d, %d, %d)", i, ms[i], x[i], y[i], z[i],
				wantMS, centiemes(s.X), centiemes(s.Y), centiemes(s.Z))
		}
	}
	// Les points existants sont ceux de l option eteinte, au bit pres.
	if len(p.P) != len(eteint[0].P) {
		t.Fatalf("%d points allume, %d eteint", len(p.P), len(eteint[0].P))
	}
	for i := range p.P {
		if p.P[i] != eteint[0].P[i] {
			t.Fatalf("point %d : %v allume, %v eteint", i, p.P[i], eteint[0].P[i])
		}
	}
	if p.T0 != eteint[0].T0 || p.Rest != eteint[0].Rest || p.DetoT != eteint[0].DetoT || p.DetoX != eteint[0].DetoX {
		t.Errorf("les champs existants ont bouge : %+v contre %+v", p, eteint[0])
	}
	verifierLesPointsDuVol(t, p, ms, x, y, z, pasVolUS)
}

// verifierLesPointsDuVol : chaque point de la grille est le PREMIER echantillon de sa frame (meme x, y en
// centiemes), et `z` donne, au meme indice, la hauteur de cet echantillon, au centieme.
func verifierLesPointsDuVol(t *testing.T, p Projectile, ms []int64, x, y, z []int32, pasUS int64) {
	t.Helper()
	if len(p.Z) != len(p.P) {
		t.Fatalf("`z` a %d valeurs pour %d points : il doit etre parallele a `p`", len(p.Z), len(p.P))
	}
	j := 0
	for i, pt := range p.P {
		frame := p.T0 + int(pt[0])
		for j < len(ms) && int(ms[j]*fullRateTimeUnitUS/pasUS) < frame {
			j++
		}
		if j >= len(ms) || int(ms[j]*fullRateTimeUnitUS/pasUS) != frame {
			t.Fatalf("point %d (frame %d) sans echantillon dans sa frame", i, frame)
		}
		if int32(math.Round(float64(pt[1])*coordScale)) != x[j] || int32(math.Round(float64(pt[2])*coordScale)) != y[j] ||
			int32(math.Round(float64(p.Z[i])*coordScale)) != z[j] {
			t.Fatalf("point %d : (%v, %v, z=%v) contre l echantillon (%d, %d, %d) centiemes", i, pt[1], pt[2], p.Z[i], x[j], y[j], z[j])
		}
	}
}

// TestVolCoupeCoupeAussiSesEchantillonsNatifs : cinq echantillons lisibles a 1 m par record, puis un saut de
// 40 m en un record (2400 m/s) : le vol s arrete au cinquieme, et la serie native et la hauteur s arretent AVEC
// LUI.
func TestVolCoupeCoupeAussiSesEchantillonsNatifs(t *testing.T) {
	pistes := []types.ProjectileTrack{vol60Hz(10, true)}
	for i := range pistes[0].Pts {
		x := float32(i)
		if i >= 5 {
			x += 40
		}
		pistes[0].Pts[i].X = x
	}
	eteint, allume, echantillons := volsAvecEtSansPleineCadence(pistes)
	if len(allume) != 1 || echantillons != 5 || len(allume[0].FullRate.T) != 5 {
		t.Fatalf("la coupure borne la serie native : 5 echantillons attendus, obtenu %d", echantillons)
	}
	if len(allume[0].Z) != len(allume[0].P) || len(allume[0].P) != len(eteint[0].P) {
		t.Errorf("la coupure borne aussi la hauteur : %d valeurs pour %d points (eteint : %d)", len(allume[0].Z),
			len(allume[0].P), len(eteint[0].P))
	}
	if allume[0].Rest {
		t.Error("un vol coupe ne CERTIFIE aucune fin : Rest doit valoir false")
	}
}

// TestVolPleineCadenceIgnoreUnePisteTropCourte : une piste ignoree (moins de trois echantillons) ne laisse
// ni hauteur ni serie, et le vol publie garde la sienne.
func TestVolPleineCadenceIgnoreUnePisteTropCourte(t *testing.T) {
	pistes := []types.ProjectileTrack{vol60Hz(2, true), vol60Hz(15, true)}
	_, allume, echantillons := volsAvecEtSansPleineCadence(pistes)
	if len(allume) != 1 || echantillons != 15 || len(allume[0].FullRate.T) != 15 {
		t.Fatalf("%d projectile(s) publie(s), %d echantillons ; attendu 1 et 15 (la piste de 2 echantillons n est pas un vol)",
			len(allume), echantillons)
	}
}

// verifierLesVolsDuDocument : sur un document assemble option allumee, chaque projectile porte sa hauteur et son
// vol, coherents avec ses points ; rend le nombre d echantillons de pleine cadence, et retire la hauteur et le vol pour
// que le document se compare a celui de l option eteinte.
func verifierLesVolsDuDocument(t *testing.T, doc *ReplayDocument) int {
	t.Helper()
	somme := 0
	for i := range doc.Projectiles {
		p := &doc.Projectiles[i]
		if p.FullRate == nil {
			t.Fatalf("projectile %d sans pleine cadence", i)
		}
		ms, x, y, z := sommeCumulee64(p.FullRate.T), sommeCumulee32(p.FullRate.X), sommeCumulee32(p.FullRate.Y), sommeCumulee32(p.FullRate.Z)
		verifierLesPointsDuVol(t, *p, ms, x, y, z, int64(doc.FrameIntervalMS)*1000)
		somme += len(ms)
		p.Z, p.FullRate = nil, nil
	}
	if pc := doc.Coverage.Projectiles; pc != nil {
		if pc.FullRateSamples != somme {
			t.Errorf("couverture des projectiles : %d echantillons, colonnes : %d", pc.FullRateSamples, somme)
		}
		pc.FullRateSamples = 0
	}
	return somme
}
