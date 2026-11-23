package replay

// grenades_detonation_test.go — L'EXPLOSION D'UN LANCER SANS VOL PUBLIÉ (Grenade.Deto), sur des
// entrées SYNTHÉTIQUES.
//
// Le lien est le handle : le slot du projectile du lancer vaut la référence du message plus la base
// 0x400, à génération égale. Chaque cas ci-dessous écarte un message qu'une lecture plus lâche (le
// seul temps, la seule proximité, n'importe quel genre) aurait pris pour le bon.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	gdTestSlot = uint32(1179)
	gdTestGen  = uint32(1)
)

// gdTestThrow est un lancer de plasma à t = 10 s dont le projectile a le handle (1179, 1).
func gdTestThrow() grammar.GrenadeThrow {
	return grammar.GrenadeThrow{
		TimestampUS: 10_000_000, FilmIndex: 3, TypeID: grammar.GrenadePlasma,
		Life: types.LifeKey{Slot: gdTestSlot, Gen: gdTestGen}, HasLife: true,
	}
}

// gdTestDeto fabrique un message de détonation du handle du lancer, `ecartUS` après lui.
func gdTestDeto(genre uint8, ecartUS uint64, x, y, z float32) types.ProjectileDetonation {
	return types.ProjectileDetonation{
		TimestampUS: 10_000_000 + ecartUS, Genre: genre,
		Projectile:    types.LifeKey{Slot: gdTestSlot - detonationHandleBase, Gen: gdTestGen},
		HasProjectile: true, X: x, Y: y, Z: z, HasPosition: true,
	}
}

func TestDetonationDuLancerParLeHandle(t *testing.T) {
	d, ok := detonationOfThrow(gdTestThrow(), []types.ProjectileDetonation{gdTestDeto(5, 1_200_000, 9.69, 5.09, 5.02)})
	if !ok || d.X != 9.69 || d.TimestampUS != 11_200_000 {
		t.Fatalf("la détonation du handle doit être rendue : %+v (%v)", d, ok)
	}
}

// TestDetonationDuLancerEcarteCeQuiNeNommePasLeHandle : chaque message écarté est un piège distinct.
func TestDetonationDuLancerEcarteCeQuiNeNommePasLeHandle(t *testing.T) {
	cas := []struct {
		nom string
		d   func() types.ProjectileDetonation
	}{
		{"autre génération", func() types.ProjectileDetonation {
			d := gdTestDeto(5, 1_000_000, 1, 2, 3)
			d.Projectile.Gen = gdTestGen + 1
			return d
		}},
		{"autre slot", func() types.ProjectileDetonation {
			d := gdTestDeto(5, 1_000_000, 1, 2, 3)
			d.Projectile.Slot++
			return d
		}},
		{"référence prise pour un slot brut", func() types.ProjectileDetonation {
			d := gdTestDeto(5, 1_000_000, 1, 2, 3)
			d.Projectile.Slot = gdTestSlot
			return d
		}},
		{"effet d'impact et non détonation", func() types.ProjectileDetonation { return gdTestDeto(6, 1_000_000, 1, 2, 3) }},
		{"avant le lancer", func() types.ProjectileDetonation {
			d := gdTestDeto(5, 0, 1, 2, 3)
			d.TimestampUS = 9_000_000
			return d
		}},
		{"au-delà de la fenêtre", func() types.ProjectileDetonation {
			return gdTestDeto(5, grenadeDetonationWindowUS+1, 1, 2, 3)
		}},
		{"sans position lue", func() types.ProjectileDetonation {
			d := gdTestDeto(5, 1_000_000, 1, 2, 3)
			d.HasPosition = false
			return d
		}},
		{"sans référence de projectile", func() types.ProjectileDetonation {
			d := gdTestDeto(5, 1_000_000, 1, 2, 3)
			d.HasProjectile = false
			return d
		}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if d, ok := detonationOfThrow(gdTestThrow(), []types.ProjectileDetonation{c.d()}); ok {
				t.Fatalf("message pris à tort : %+v", d)
			}
		})
	}
}

// TestDetonationDuLancerPrendLExplosionDeLaGrenadeAPointes : la grenade à pointes colle (effet
// d'impact) puis explose 1,2 s plus tard ; l'explosion est le second message, pas le premier.
func TestDetonationDuLancerPrendLExplosionDeLaGrenadeAPointes(t *testing.T) {
	detos := []types.ProjectileDetonation{
		gdTestDeto(6, 300_000, 1, 1, 1),
		gdTestDeto(5, 1_500_000, 1.1, 1.2, 1.3),
	}
	d, ok := detonationOfThrow(gdTestThrow(), detos)
	if !ok || d.Genre != genreDetonation || d.TimestampUS != 11_500_000 {
		t.Fatalf("l'explosion (genre 5) est attendue, pas l'impact : %+v (%v)", d, ok)
	}
}

// TestDetonationDuLancerPrendLaPremiere : deux détonations du même handle dans la fenêtre, la
// première gagne, quel que soit l'ordre de la liste.
func TestDetonationDuLancerPrendLaPremiere(t *testing.T) {
	detos := []types.ProjectileDetonation{gdTestDeto(5, 5_000_000, 9, 9, 9), gdTestDeto(5, 2_000_000, 1, 1, 1)}
	if d, ok := detonationOfThrow(gdTestThrow(), detos); !ok || d.X != 1 {
		t.Fatalf("la première détonation est attendue : %+v", d)
	}
}

// TestDetonationDuLancerSansHandleNeRendRien : un lancer dont le handle n'a pas été lu ne se lie à
// rien — pas de repli par le temps ou la proximité.
func TestDetonationDuLancerSansHandleNeRendRien(t *testing.T) {
	th := gdTestThrow()
	th.HasLife = false
	if d, ok := detonationOfThrow(th, []types.ProjectileDetonation{gdTestDeto(5, 1_000_000, 1, 2, 3)}); ok {
		t.Fatalf("lancer sans handle lié : %+v", d)
	}
}

// TestGrenadeDetonationSurLaGrille : l'instant est l'index de frame sur l'axe de Point.T, la position
// est arrondie au centième ; une détonation antérieure à l'origine n'a pas d'instant sur l'axe.
func TestGrenadeDetonationSurLaGrille(t *testing.T) {
	g := grenadeDetonation(gdTestThrow(), []types.ProjectileDetonation{gdTestDeto(5, 1_234_000, 1.234, 2.345, 3.456)}, 1_000_000, 100_000)
	if g == nil || g.T != 102 || g.X != 1.23 || g.Y != 2.35 || g.Z != 3.46 {
		t.Fatalf("frame (11,234-1)/0,1 = 102 et centièmes attendus : %+v", g)
	}
	if g := grenadeDetonation(gdTestThrow(), []types.ProjectileDetonation{gdTestDeto(5, 1_000_000, 1, 2, 3)}, 12_000_000, 100_000); g != nil {
		t.Errorf("une détonation avant l'origine ne se publie pas : %+v", g)
	}
	if g := grenadeDetonation(gdTestThrow(), nil, 0, 100_000); g != nil {
		t.Errorf("aucune détonation lue, aucune publiée : %+v", g)
	}
}

// gdTestPos fabrique la piste du lanceur, pour que le lancer se situe par son biped.
func gdTestLancer(t *testing.T, vols volsDeGrenades) []Grenade {
	t.Helper()
	pos := []grammar.BipedPosition{{Slot: 1024, TimestampUS: 10_000_000, X: 4, Y: 5, HasWorld: true}}
	gren, _ := buildGrenades(indexBySlot(pos), []grammar.GrenadeThrow{gdTestThrow()}, 1_000_000, 100_000,
		occupantsPlats(map[uint32]int{1024: 3}), vols)
	return gren
}

// TestBuildGrenadesPoseLExplosionDuLancerSansVol : sans projectile publié, le lancer porte l'explosion
// que le film écrit pour son handle ; avec un projectile publié, il ne la répète pas.
func TestBuildGrenadesPoseLExplosionDuLancerSansVol(t *testing.T) {
	detos := []types.ProjectileDetonation{gdTestDeto(5, 1_200_000, 9.69, 5.09, 5.02)}
	gren := gdTestLancer(t, volsDeGrenades{Detonations: detos})
	if len(gren) != 1 || gren[0].Proj != nil {
		t.Fatalf("un lancer sans vol publié est attendu : %+v", gren)
	}
	if d := gren[0].Deto; d == nil || d.T != 102 || d.X != 9.69 || d.Z != 5.02 {
		t.Errorf("l'explosion du handle est attendue sur le lancer : %+v", d)
	}
	// Le même lancer, relié à un projectile publié par sa naissance : l'explosion est celle du projectile.
	proj := []types.ProjectileTrack{{
		Slot: gdTestSlot, Gen: gdTestGen,
		Pts: []types.ProjectileSample{{TimestampUS: 10_000_000, X: 4, Y: 5}, {TimestampUS: 10_100_000, X: 5, Y: 5}},
	}}
	gren = gdTestLancer(t, volsDeGrenades{Tracks: proj, PubByRaw: map[int]int{0: 0}, Detonations: detos})
	if len(gren) != 1 || gren[0].Proj == nil {
		t.Fatalf("un lancer relié à un projectile publié est attendu : %+v", gren)
	}
	if gren[0].Deto != nil {
		t.Errorf("l'explosion est publiée sur le projectile : Deto doit rester nil, obtenu %+v", gren[0].Deto)
	}
}
