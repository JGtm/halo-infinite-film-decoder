package replay

// vehicle_destruction_killed_vie_test.go — LA BORNE DE VIE DE LA REGLE DES OCCUPANTS TUES, LA FUSION DES RELAIS,
// LA DEDUPLICATION, ET LE CHEMIN DE LA MARCHE A LA PUBLICATION. Cas synthetiques, aucun film ouvert.

import (
	"context"
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// TestUnDeadStateRepliqueApresLaDestructionNeTuePas (a) : un bipede rattache au chassis meurt a P0 < P, puis son
// dead-state est re-replique dans P. Il est mort AVANT le vehicule : sa vie finit a P0, la replication n a plus de
// lecture « attache » dans sa fenetre de vie.
func TestUnDeadStateRepliqueApresLaDestructionNeTuePas(t *testing.T) {
	avant := paquetDestruction - 1_000_000
	morts := []types.ObjectDeath{
		mortDeBipede(avant, indexPassager, deadStateTagAbsent),
		mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent),
	}
	d, c := tuesAvec(t, []types.VehicleOccupancy{attache(20_000_000, slotChassis, 1)}, morts)
	if len(tuesPublies(d)) != 0 || c.Killed != 0 || c.KilledNoIdentity != 0 {
		t.Errorf("killed = %v, couverture = %+v : un mort avant la destruction ne se publie pas, replique ou non", tuesPublies(d), c)
	}
}

// TestUneVieSuivanteDeLaMemeCleNHeritePasDuRattachement (b) : la vie A de cle (S, g), rattachee au chassis sans
// lecture libre, meurt avant P ; la vie B de meme cle, sans aucune lecture de parente, meurt dans P. B n a jamais
// ete a bord.
func TestUneVieSuivanteDeLaMemeCleNHeritePasDuRattachement(t *testing.T) {
	morts := []types.ObjectDeath{
		mortDeBipede(paquetDestruction-5_000_000, 5, deadStateTagAbsent),   // la vie A
		mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent), // la vie B
	}
	d, c := tuesAvec(t, []types.VehicleOccupancy{attache(20_000_000, slotChassis, 1)}, morts)
	if len(tuesPublies(d)) != 0 || c.Killed != 0 {
		t.Errorf("killed = %v, couverture = %+v : la vie B n a aucune lecture, elle n etait pas a bord", tuesPublies(d), c)
	}
}

// TestLaBorneDeVieGardeLesCasLegitimes : la borne ne retire que le rattachement d une vie anterieure. Une vie
// rattachee apres la mort de la precedente de sa cle est tuee avec sa sortie ecrite (clause 2) ; deux dead-states de
// la meme cle dans le MEME paquet ne se neutralisent pas.
func TestLaBorneDeVieGardeLesCasLegitimes(t *testing.T) {
	apres := paquetDestruction + 34_000
	lues := []types.VehicleOccupancy{
		attache(5_000_000, slotChassis, 1), // vie anterieure de la cle
		attache(27_000_000, slotChassis, 1),
		libre(apres),
	}
	morts := []types.ObjectDeath{
		mortDeBipede(10_000_000, 5, deadStateTagAbsent), // fin de la vie anterieure
		mortDeBipede(apres, indexPassager, deadStateTagAbsent),
	}
	if d, _ := tuesAvec(t, lues, morts); !slices.Equal(tuesPublies(d), []string{xuidPassager}) {
		t.Errorf("clause 2 apres une vie anterieure : killed = %v, attendu [%s]", tuesPublies(d), xuidPassager)
	}

	memePaquet := []types.ObjectDeath{
		mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent),
		mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent),
	}
	d, c := tuesAvec(t, []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1)}, memePaquet)
	if !slices.Equal(tuesPublies(d), []string{xuidPassager}) || c.KilledOccupants != 1 {
		t.Errorf("deux dead-states dans le meme paquet : killed = %v, couverture = %+v, attendu un seul tue", tuesPublies(d), c)
	}
}

// TestUneVieEstCompteeUneFoisEtUnJoueurPublieUneFois : un bot ne se compte qu une fois par vie, meme quand deux de
// ses dead-states satisfont la regle (rattache de nouveau apres le premier) ; un humain ne se publie qu une fois,
// meme sur deux vies.
func TestUneVieEstCompteeUneFoisEtUnJoueurPublieUneFois(t *testing.T) {
	apres := paquetDestruction + 34_000
	rattachements := []types.VehicleOccupancy{
		attache(27_000_000, slotChassis, 1),
		attache(paquetDestruction+10_000, slotChassis, 1), // rattache de nouveau juste apres sa mort
		libre(apres),
	}
	for _, cas := range []struct {
		nom       string
		victime   int32
		identites int
		sans      int
	}{{"humain", indexPassager, 1, 0}, {"bot", 9, 0, 1}} {
		d, c := tuesAvec(t, rattachements, []types.ObjectDeath{
			mortDeBipede(paquetDestruction, cas.victime, deadStateTagAbsent),
			mortDeBipede(apres, cas.victime, deadStateTagAbsent),
		})
		if len(tuesPublies(d)) != cas.identites || c.KilledOccupants != cas.identites || c.KilledNoIdentity != cas.sans {
			t.Errorf("%s, deux dead-states d une vie : killed = %v, couverture = %+v, attendu %d publie(s) et %d sans identite",
				cas.nom, tuesPublies(d), c, cas.identites, cas.sans)
		}
	}

	// Deux vies d un meme joueur (deux slots), l une et l autre rattachees : un seul xuid.
	autre := types.VehicleOccupancy{TimestampUS: 27_000_000, Slot: 901, Gen: 1, Attached: true, ParentSlot: slotChassis, ParentGen: 1}
	mortAutre := mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent)
	mortAutre.Slot, mortAutre.Gen = 901, 1
	d, c := tuesAvec(t, []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1), autre},
		[]types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent), mortAutre})
	if !slices.Equal(tuesPublies(d), []string{xuidPassager}) || c.KilledOccupants != 1 {
		t.Errorf("deux vies du meme joueur : killed = %v, couverture = %+v, attendu un seul xuid", tuesPublies(d), c)
	}
}

// TestLaFusionDesRelaisPorteLaVieDeLaMortEtLeSlotPublie : apres `mergeVehicleRelays`, la piste porte le slot et la
// generation de la PREMIERE vie, et `mort` ceux du relais qui est mort. G est fait de la vie de la mort (un bipede
// rattache au relais est tue) et ses pieces se trouvent par la cle PUBLIEE du chassis (`Carrier`).
func TestLaFusionDesRelaisPorteLaVieDeLaMortEtLeSlotPublie(t *testing.T) {
	mort := morteDeTest(paquetDestruction, 3, deadStateTagAbsent)
	mort.Slot, mort.Gen = 702, 3 // le relais
	chassis := vieDetruite(282, mort)
	chassis.Slot, chassis.Gen = 700, 1 // la premiere vie : la cle publiee
	piece := pieceMontee(paquetDestruction, VehicleLifeRef{Slot: 700, Gen: 1})
	piece.mort.Slot, piece.mort.Gen = slotPiece, 2

	surLePorteur := types.VehicleOccupancy{TimestampUS: 27_000_000, Slot: 901, Gen: 1, Attached: true, ParentSlot: 702, ParentGen: 3}
	surLaPiece := attache(27_000_000, slotPiece, 2)
	mortSurLePorteur := mortDeBipede(paquetDestruction, 5, deadStateTagAbsent)
	mortSurLePorteur.Slot, mortSurLePorteur.Gen = 901, 1
	morts := []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent), mortSurLePorteur}

	doc := documentDeTest(chassis, piece)
	poserLesDestructions(t, doc, vehicleDestructionInputs{
		roster: rosterDeTest(), reads: newOccupantReads([]types.VehicleOccupancy{surLaPiece, surLePorteur}, morts)})
	d := doc.Vehicles[0].Destruction
	if d == nil || !slices.Equal(d.Killed, []string{xuidPassager, xuidAutre}) {
		t.Errorf("destruction = %+v, attendu le tue de la piece (%s) et celui du relais (%s)", d, xuidPassager, xuidAutre)
	}
}

// TestDeLaMarcheALaPublication : le chemin de la production, de la marche des trames au document. Les dead-states
// de la marche sont rangees par archetype (`mortsDeVehicule`), prises par `balayerEtatsDeMouvement`, passees au
// calque des vehicules et assemblees : le bipede rattache au vehicule et mort avec lui est publie.
func TestDeLaMarcheALaPublication(t *testing.T) {
	const slotRattache = uint32(950)
	vehicule := types.LifeKey{Slot: 700, Gen: 1}
	mortVehicule := *morteDeTest(12_000_000, 3, deadStateTagAbsent)
	mortBipede := mortDeBipede(12_000_000, indexPassager, deadStateTagAbsent)
	mortBipede.Slot = slotRattache
	mortAutreArchetype := mortBipede
	mortAutreArchetype.TypeIndex = 38
	lecture := attache(11_000_000, 700, 1)
	lecture.Slot = slotRattache

	marche := grammar.MarcheDesTrames{
		ObjectDeaths: []types.ObjectDeath{mortVehicule, mortBipede, mortAutreArchetype},
		Occupancy:    []types.VehicleOccupancy{lecture},
	}
	times := []uint64{2_000_000, 22_000_000}
	s := &filmScan{
		ctx: context.Background(), matchID: "m", marche: marche,
		in: FilmInputs{Vehicles: VehicleScan{
			Scanned: true, Keyframes: vehKeyframes(times, vehicule, times),
			Creations: []types.EquipmentCreation{vehCreation(vehicule, 1_500_000, 40, 40, vehChassisKnown)},
			Positions: []grammar.BipedPosition{vehPos(700, 3_000_000, 40, 40)},
		}},
	}
	s.balayerEtatsDeMouvement()
	if got := s.in.Vehicles.BipedDeaths; len(got) != 1 || got[0].Slot != slotRattache || got[0].TypeIndex != uint32(grammar.BipedTypeIndex) {
		t.Fatalf("morts de bipede prises de la marche = %+v, attendu la seule mort du bipede", got)
	}
	if got := s.in.Vehicles.Deaths; len(got) != 1 || got[0].TypeIndex != uint32(grammar.VehicleTypeIndex) {
		t.Fatalf("morts de vehicule prises de la marche = %+v, attendu la seule mort du vehicule", got)
	}

	opt := Options{FrameIntervalMS: 100, Vehicles: s.in.Vehicles,
		PlayerIndices: types.PlayerIndexTable{ByXUID: map[uint64]int{1002: int(indexPassager)}, Readings: 1}}
	doc := BuildFromPositions(context.Background(), "m", "halo_infinite", porteFoule(300, 0), nil, opt)
	if len(doc.Vehicles) != 1 || doc.Vehicles[0].End != VehicleEndDestroyed {
		t.Fatalf("vehicules publies = %+v, attendu un chassis detruit", doc.Vehicles)
	}
	if d := doc.Vehicles[0].Destruction; d == nil || !slices.Equal(d.Killed, []string{xuidPassager}) {
		t.Errorf("destruction = %+v, attendu killed = [%s]", d, xuidPassager)
	}
}

// TestMortsDeVehiculeRangeParArchetype : la marche rend les morts de tous les archetypes ; le calque prend celles
// des vehicules (`ti=40`) et des bipedes (`ti=35`), et l occupation telle quelle.
func TestMortsDeVehiculeRangeParArchetype(t *testing.T) {
	m := grammar.MarcheDesTrames{
		ObjectDeaths: []types.ObjectDeath{
			{Slot: 1, TypeIndex: uint32(grammar.VehicleTypeIndex)},
			{Slot: 2, TypeIndex: uint32(grammar.BipedTypeIndex)},
			{Slot: 3, TypeIndex: 38},
			{Slot: 4, TypeIndex: uint32(grammar.BipedTypeIndex)},
		},
		Occupancy: []types.VehicleOccupancy{{Slot: 2, Attached: true}},
	}
	got := mortsDeVehicule(context.Background(), "m", m)
	if len(got.Vehicles) != 1 || got.Vehicles[0].Slot != 1 {
		t.Errorf("morts de vehicule = %+v, attendu le slot 1 seul", got.Vehicles)
	}
	if len(got.Bipeds) != 2 || got.Bipeds[0].Slot != 2 || got.Bipeds[1].Slot != 4 {
		t.Errorf("morts de bipede = %+v, attendu les slots 2 et 4", got.Bipeds)
	}
	if len(got.Occupancy) != 1 {
		t.Errorf("occupation = %+v, attendu une lecture", got.Occupancy)
	}
}
