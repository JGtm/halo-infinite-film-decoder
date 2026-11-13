package replay

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// visee fabrique une lecture i21 portee par un record de position, au cap et a l elevation donnes
// (quanta exacts : cap 360*(q+0.5)/4096, elevation 360*(q+0.5)/2048-180).
func visee(slot uint32, tUS uint64, yaw, pitch uint32) grammar.BipedPosition {
	p := grammar.BipedPosition{Slot: slot, TimestampUS: tUS, X: 1, Y: 1, HasWorld: true}
	p.HasYaw, p.YawRaw, p.PitchRaw = true, yaw, pitch
	return p
}

// TestLaViseeDuTirEstLaDerniereDuTireur : le tir prend la derniere lecture i21 de son slot AU PLUS
// TARD a son instant — ni la suivante, ni une interpolation, ni le champ du record.
func TestLaViseeDuTirEstLaDerniereDuTireur(t *testing.T) {
	pos := []grammar.BipedPosition{
		visee(10, 1_000_000, 1024, 1100),                               // cap 90,04 ; elevation 13,45
		{Slot: 10, TimestampUS: 1_050_000, X: 1, Y: 1, HasWorld: true}, // position sans visee
		visee(10, 1_200_000, 2048, 1024),                               // apres le tir : ne compte pas
	}
	fb := fallback.NouveauCompteur()
	v := nouveauViseur(pos, nil, fb)
	var s Shot
	v.viser(&s, fireAt(1_100_000, 3, 270), 10) // le record dit 270 : il est ignore
	if s.H != 90 || s.P != 13.4 {
		t.Fatalf("visee (%v, %v), attendu (90, 13.4) : la derniere lecture i21 avant le tir", s.H, s.P)
	}
	if *v.cov != (ShotAimCoverage{Track: 1}) || fb.Compte(fallback.NomViseeDuTirLueDansLeRecord) != 0 {
		t.Errorf("couverture %+v, repli %d", *v.cov, fb.Compte(fallback.NomViseeDuTirLueDansLeRecord))
	}
	var meme Shot
	v.viser(&meme, fireAt(1_200_000, 3, 270), 10)
	if meme.H != 180 || v.cov.SameTick != 1 {
		t.Errorf("tir du paquet de la lecture : cap %v, memeTick %d ; attendu 180 et 1", meme.H, v.cov.SameTick)
	}
}

// TestLaViseeDuRecordNEstQuUnRepliCompte : sans lecture i21 tenable (aucune, ou plus vieille que la vie),
// le champ du record sert et le repli est compte ; sans lui non plus, le tir n a pas de direction.
func TestLaViseeDuRecordNEstQuUnRepliCompte(t *testing.T) {
	pos := []grammar.BipedPosition{visee(10, 1_000_000, 1024, 1024)}
	fb := fallback.NouveauCompteur()
	v := nouveauViseur(pos, nil, fb)
	var perimee, sansRien, autreSlot Shot
	v.viser(&perimee, fireAt(1_000_000+viseeTenueMaxUS+1, 3, 270), 10)
	v.viser(&sansRien, grammar.FireEvent{TimestampUS: 1_100_000}, 11)
	v.viser(&autreSlot, fireAt(1_100_000, 3, 45), 11)
	if perimee.H != 270 || autreSlot.H != 45 || sansRien.H != 0 || sansRien.P != 0 {
		t.Errorf("caps %v / %v / %v, attendu 270 (record), 45 (record), 0 (rien)", perimee.H, autreSlot.H, sansRien.H)
	}
	if *v.cov != (ShotAimCoverage{Record: 2, None: 1}) {
		t.Errorf("couverture %+v, attendu 2 records et 1 sans direction", *v.cov)
	}
	if n := fb.Compte(fallback.NomViseeDuTirLueDansLeRecord); n != 2 {
		t.Errorf("repli compte %d fois, attendu 2", n)
	}
}

// TestLaViseeSansPositionSertLeTireur : une lecture i21 d un record SANS position (un occupant de
// vehicule) est une visee du tireur comme une autre, et la plus recente gagne.
func TestLaViseeSansPositionSertLeTireur(t *testing.T) {
	pos := []grammar.BipedPosition{visee(10, 1_000_000, 1024, 1024)}
	bord := []grammar.BipedAim{{Slot: 10, TimestampUS: 1_050_000, YawRaw: 3072, PitchRaw: 1024}}
	v := nouveauViseur(pos, bord, nil)
	var s Shot
	v.viser(&s, grammar.FireEvent{TimestampUS: 1_060_000}, 10)
	if s.H != 270 || v.cov.Track != 1 {
		t.Errorf("cap %v (couverture %+v), attendu 270 lu dans le record sans position", s.H, *v.cov)
	}
}

// TestLaCouvertureDesVisesCouvreChaqueTirPose : sur les builds figes, chaque tir pose (publie ou non)
// a une source de direction, et une seule.
func TestLaCouvertureDesVisesCouvreChaqueTirPose(t *testing.T) {
	for _, b := range goldenBuilds() {
		t.Run(b.Build+"/"+b.Short8, func(t *testing.T) {
			g, entry := chargerGoldenBuild(t, b)
			doc := assemblerGoldenBuild(t, b, g, entry)
			c := doc.Coverage.Shots
			if c.Aim == nil {
				t.Fatal("coverage.shots.aim absente")
			}
			if a := c.Aim; a.Track+a.Record+a.None != c.Attached+c.Unpublished || a.SameTick > a.Track {
				t.Errorf("visees %+v pour %d tirs poses (%d publies)", *a, c.Attached+c.Unpublished, c.Attached)
			}
		})
	}
}
