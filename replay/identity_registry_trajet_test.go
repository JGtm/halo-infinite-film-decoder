package replay

import (
	"context"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// identity_registry_trajet_test.go — LE JOUEUR QUI PORTE UN TRAJET DE VEHICULE
// ([IdentityRegistry.XUIDDuTrajetA]). Donnees synthetiques : aucun joueur reel.

// entreeSiegeBotPuisHumain : le slot 100 porte deux corps successifs — generation 1 (index 7, que la
// table ne rattache a personne : un bot) qui vit [1 s..4 s], puis generation 2 (index 1, le joueur
// 222) nee a 12 s qui vit [14 s..18 s] ; le slot 200 porte le joueur 333 tout le film.
func entreeSiegeBotPuisHumain() IdentityInput {
	in := filmDeuxCorps()
	in.BipedCreations = []grammar.BipedCreation{
		creationGen(100, 500_000, 7, 1),
		creationGen(100, 12_000_000, 1, 2),
		creationDe(200, 1_000_000, 2),
	}
	in.PlayerIndices.ByXUID = map[uint64]int{222: 1, 333: 2}
	return in
}

// TestXUIDDuTrajet_UnBotNeRecoitPasLeNomDeLHumainDuSiege : le bot du corps de generation 1 monte a
// bord a 4,1 s, 100 ms apres sa derniere position (un occupant embarque ne replique plus) : aucune
// vie ne couvre l'instant, et le pont par slot ne connait que l'humain du corps suivant. Le trajet
// est celui du corps du bot : sans nom.
//
// ROUGE AVANT : `XUIDAt` rend 222 (le premier occupant NOMME du siege) a 4,1 s, et aussi a 3 s sous
// la vie du bot, qu'il ignore faute de nom.
//
// MUTATION : retirer la sortie `return ""` de `XUIDDuTrajetA` (le repli par `XUIDAt` quand aucune vie
// du corps n'est nommee) -> 222 a 4,1 s et a 3 s, rouge.
func TestXUIDDuTrajet_UnBotNeRecoitPasLeNomDeLHumainDuSiege(t *testing.T) {
	reg := BuildIdentityRegistry(context.Background(), entreeSiegeBotPuisHumain())
	if got := reg.XUIDAt(100, 4_100_000); got != "222" {
		t.Fatalf("XUIDAt(100, 4,1 s) = %q : le piege du siege n'est plus celui que ce test garde", got)
	}
	cas := []struct {
		nom string
		tUS uint64
		xid string
	}{
		{"juste apres la derniere position du bot", 4_100_000, ""},
		{"sous la vie du bot", 3_000_000, ""},
		{"avant la naissance du corps suivant", 11_000_000, ""},
		{"entre la creation du corps humain et sa premiere position", 13_000_000, "222"},
		{"sous la vie de l'humain", 15_000_000, "222"},
	}
	for _, c := range cas {
		if got := reg.XUIDDuTrajetA(100, c.tUS); got != c.xid {
			t.Errorf("%s : XUIDDuTrajetA(100, %d) = %q, attendu %q", c.nom, c.tUS, got, c.xid)
		}
	}
}

// TestXUIDDuTrajet_CasNominal : deux corps nommes, deux joueurs — le trajet de chacun porte son
// joueur, sous sa vie comme juste apres elle.
func TestXUIDDuTrajet_CasNominal(t *testing.T) {
	in := entreeSlotRecycle(1)
	in.PlayerIndices.ByXUID = map[uint64]int{111: 0, 222: 1, 333: 2}
	reg := BuildIdentityRegistry(context.Background(), in)
	cas := []struct {
		tUS  uint64
		want string
	}{
		{2_000_000, "111"},  // sous la vie du corps de generation 1
		{4_100_000, "111"},  // juste apres sa derniere position
		{15_000_000, "222"}, // sous la vie du corps de generation 2
	}
	for _, c := range cas {
		if got := reg.XUIDDuTrajetA(100, c.tUS); got != c.want {
			t.Errorf("XUIDDuTrajetA(100, %d) = %q, attendu %q", c.tUS, got, c.want)
		}
	}
}

// TestXUIDDuTrajet_GardeLaReponseDeXUIDAt : sans corps etabli, ou sur un siege dont les records
// portent un seul participant, le film ne dit aucune frontiere d'occupant : la reponse est celle de
// `XUIDAt`, instant par instant.
func TestXUIDDuTrajet_GardeLaReponseDeXUIDAt(t *testing.T) {
	sansCorps := filmDeuxCorps()
	sansCorps.BipedCreations = nil
	memeJoueur := entreeSlotRecycle(0) // les deux corps portent le meme joueur
	for nom, in := range map[string]IdentityInput{"sans corps etabli": sansCorps, "meme participant": memeJoueur} {
		reg := BuildIdentityRegistry(context.Background(), in)
		for _, tUS := range []uint64{500_000, 2_000_000, 4_100_000, 13_000_000, 15_000_000} {
			if got, want := reg.XUIDDuTrajetA(100, tUS), reg.XUIDAt(100, tUS); got != want {
				t.Errorf("%s : XUIDDuTrajetA(100, %d) = %q, XUIDAt = %q", nom, tUS, got, want)
			}
		}
	}
}

// TestLeTrajetLuPortLeJoueurDeSonCorps : le site de la lecture du film (`i10`) nomme le trajet par le
// corps de l'occupant. Le bot du slot 100 monte a bord du vehicule 900 a 4,1 s.
func TestLeTrajetLuPortLeJoueurDeSonCorps(t *testing.T) {
	key := types.LifeKey{Slot: 900, Gen: 1}
	in := vehicleRideInputs{
		reg:      BuildIdentityRegistry(context.Background(), entreeSiegeBotPuisHumain()),
		lives:    []vehicleLife{{key: key, loUS: 1_000_000, hiUS: 18_000_000}},
		drawable: map[types.LifeKey]bool{key: true},
		occupancy: []types.VehicleOccupancy{
			{TimestampUS: 4_100_000, Slot: 100, Gen: 1, Attached: true, ParentSlot: 900, ParentGen: 1},
			{TimestampUS: 15_100_000, Slot: 100, Gen: 2, Attached: true, ParentSlot: 900, ParentGen: 1},
		},
		clock: vehClock(),
	}
	lus, _ := buildVehicleFilmRides(in)
	rides := lus.rides[key]
	if len(rides) != 2 {
		t.Fatalf("%d episodes, attendu 2", len(rides))
	}
	if rides[0].XUID != "" {
		t.Errorf("trajet du bot : XUID = %q, attendu aucun nom", rides[0].XUID)
	}
	if rides[1].XUID != "222" {
		t.Errorf("trajet de l'humain : XUID = %q, attendu 222", rides[1].XUID)
	}
}
