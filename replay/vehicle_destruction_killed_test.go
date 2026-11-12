package replay

// vehicle_destruction_killed_test.go — QUI EST MORT AVEC UN VEHICULE DETRUIT : chaque condition de la
// regle de vehicle_destruction_killed.go a un test qui rougit quand on l inverse. Cas synthetiques :
// dead-states de bipede, lectures de parente et roster sont poses a la main ; aucun film n est ouvert.
// Les identites sont les xuids fictifs de `xuidConducteur`, `xuidPassager` et `xuidAutre`.

import (
	"slices"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	paquetDestruction = uint64(28_200_000)
	slotChassis       = uint32(700)
	slotPiece         = uint32(701)
	slotBipede        = uint32(900)
	genBipede         = uint32(2)
	indexPassager     = int32(4) // `rosterDeTest` : xuidPassager
)

// mortDeBipede : le dead-state du bipede (slotBipede, genBipede) a `us`, de victime `victime`.
func mortDeBipede(us uint64, victime int32, effet uint32) types.ObjectDeath {
	return types.ObjectDeath{
		TimestampUS: us, Slot: slotBipede, Gen: genBipede, TypeIndex: 35,
		Dead: types.DeadState{Mort: true, EnumA: victime, EnumB: 3, SrcTag0: effet, SrcTag4c: deadStateTagAbsent},
	}
}

// attache : la lecture « attache au parent (slot, gen) » du bipede (slotBipede, genBipede) a `us`.
func attache(us uint64, parentSlot, parentGen uint32) types.VehicleOccupancy {
	return types.VehicleOccupancy{TimestampUS: us, Slot: slotBipede, Gen: genBipede, Attached: true, ParentSlot: parentSlot, ParentGen: parentGen}
}

// libre : la lecture « libre » du bipede (slotBipede, genBipede) a `us`.
func libre(us uint64) types.VehicleOccupancy {
	return types.VehicleOccupancy{TimestampUS: us, Slot: slotBipede, Gen: genBipede}
}

// tuesAvec cuit le document d un chassis detruit a [paquetDestruction] (slot 700, generation 1) et rend
// son bloc de destruction et sa couverture. `pieces` sont des vies posees avec lui.
func tuesAvec(t *testing.T, occupancy []types.VehicleOccupancy, morts []types.ObjectDeath, pieces ...VehicleTrack,
) (*VehicleDestruction, *VehicleDestructionCoverage) {
	t.Helper()
	vies := append([]VehicleTrack{vieDetruite(282, morteDeTest(paquetDestruction, 3, deadStateTagAbsent))}, pieces...)
	doc := documentDeTest(vies...)
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest(), reads: newOccupantReads(occupancy, morts)})
	return doc.Vehicles[0].Destruction, doc.Coverage.Vehicles.Destructions
}

// pieceMontee : une tourelle du chassis (slot 700, generation 1) dont le dead-state est ecrit a `us`.
func pieceMontee(us uint64, porteur VehicleLifeRef) VehicleTrack {
	m := morteDeTest(us, 3, deadStateTagAbsent)
	m.Slot = slotPiece
	p := vieDetruite(282, m)
	p.Slot, p.Part, p.Carrier = slotPiece, "turret", &porteur
	return p
}

func tuesPublies(d *VehicleDestruction) []string {
	if d == nil {
		return nil
	}
	return d.Killed
}

// TestTueQuandRattacheAvantEtMortDansLePaquet : condition 1 et condition 2 par `Pb == P`. Inverser la
// condition 1 (libre avant, ou rattache ailleurs) ne tue personne.
func TestTueQuandRattacheAvantEtMortDansLePaquet(t *testing.T) {
	mort := []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent)}
	for _, cas := range []struct {
		nom  string
		lues []types.VehicleOccupancy
		tue  bool
	}{
		{"rattache au chassis avant", []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1)}, true},
		{"libre avant", []types.VehicleOccupancy{attache(20_000_000, slotChassis, 1), libre(27_000_000)}, false},
		{"jamais lu", nil, false},
		{"rattache a un autre vehicule", []types.VehicleOccupancy{attache(27_000_000, 777, 1)}, false},
		{"meme slot, autre generation du chassis", []types.VehicleOccupancy{attache(27_000_000, slotChassis, 2)}, false},
		{"rattache dans le paquet meme (pas avant)", []types.VehicleOccupancy{attache(paquetDestruction, slotChassis, 1)}, false},
		{"lecture d une autre vie du slot du bipede", []types.VehicleOccupancy{
			{TimestampUS: 27_000_000, Slot: slotBipede, Gen: genBipede + 1, Attached: true, ParentSlot: slotChassis, ParentGen: 1}}, false},
	} {
		d, c := tuesAvec(t, cas.lues, mort)
		if got := slices.Equal(tuesPublies(d), []string{xuidPassager}); got != cas.tue {
			t.Errorf("%s : killed = %v, tue attendu %v", cas.nom, tuesPublies(d), cas.tue)
		}
		if cas.tue && (c.Killed != 1 || c.KilledOccupants != 1) {
			t.Errorf("%s : couverture = %+v", cas.nom, c)
		}
	}
}

// TestLaDerniereLectureAvantLaMortFaitFoi : un bipede rattache puis libre puis rattache de nouveau au
// chassis est tue ; rattache au chassis puis libre ne l est pas.
func TestLaDerniereLectureAvantLaMortFaitFoi(t *testing.T) {
	mort := []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent)}
	d, _ := tuesAvec(t, []types.VehicleOccupancy{
		attache(10_000_000, slotChassis, 1), libre(15_000_000), attache(20_000_000, slotChassis, 1)}, mort)
	if !slices.Equal(tuesPublies(d), []string{xuidPassager}) {
		t.Errorf("rattache de nouveau : killed = %v", tuesPublies(d))
	}
	d, _ = tuesAvec(t, []types.VehicleOccupancy{attache(10_000_000, slotChassis, 1), libre(15_000_000)}, mort)
	if len(tuesPublies(d)) != 0 {
		t.Errorf("sorti avant la destruction : killed = %v", tuesPublies(d))
	}
}

// TestUneMortUlterieureExigeLaSortieEcrite : condition 2. Une mort dans un paquet POSTERIEUR a P ne tue
// avec le vehicule que si son paquet porte la lecture « libre » du bipede.
func TestUneMortUlterieureExigeLaSortieEcrite(t *testing.T) {
	apres := paquetDestruction + 34_000
	mort := []types.ObjectDeath{mortDeBipede(apres, indexPassager, deadStateTagAbsent)}
	rattache := attache(27_000_000, slotChassis, 1)
	for _, cas := range []struct {
		nom  string
		lues []types.VehicleOccupancy
		tue  bool
	}{
		{"paquet ulterieur avec sortie ecrite", []types.VehicleOccupancy{rattache, libre(apres)}, true},
		{"paquet ulterieur sans sortie", []types.VehicleOccupancy{rattache}, false},
		{"sortie ecrite dans un autre paquet que celui de la mort", []types.VehicleOccupancy{rattache, libre(apres - 10_000)}, false},
		{"lecture rattachee (pas libre) dans le paquet de la mort", []types.VehicleOccupancy{rattache, attache(apres, slotChassis, 1)}, false},
	} {
		d, _ := tuesAvec(t, cas.lues, mort)
		if got := len(tuesPublies(d)) == 1; got != cas.tue {
			t.Errorf("%s : killed = %v, tue attendu %v", cas.nom, tuesPublies(d), cas.tue)
		}
	}
}

// TestUneMortAvantLaDestructionNeTuePas : un dead-state d un paquet anterieur a P n est pas une mort avec
// le vehicule, meme rattache.
func TestUneMortAvantLaDestructionNeTuePas(t *testing.T) {
	mort := []types.ObjectDeath{mortDeBipede(paquetDestruction-33_000, indexPassager, deadStateTagAbsent)}
	d, _ := tuesAvec(t, []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1), libre(paquetDestruction - 33_000)}, mort)
	if len(tuesPublies(d)) != 0 {
		t.Errorf("mort anterieure : killed = %v", tuesPublies(d))
	}
}

// TestLIdentiteEstCelleDuFilm : la victime est `roster[].filmIndex == EnumA`, sans aucun episode a bord.
// Le premier cas n a AUCUN trajet : le mort se publie quand meme. Le second porte un trajet qui nomme
// un AUTRE joueur sur le slot du bipede : seul l index du film compte.
func TestLIdentiteEstCelleDuFilm(t *testing.T) {
	lues := []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1)}
	mort := []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent)}
	d, _ := tuesAvec(t, lues, mort)
	if !slices.Equal(tuesPublies(d), []string{xuidPassager}) {
		t.Fatalf("sans trajet : killed = %v, attendu [%s]", tuesPublies(d), xuidPassager)
	}

	vie := vieDetruite(282, morteDeTest(paquetDestruction, 3, deadStateTagAbsent), rideDeTest(100, 300, xuidConducteur, 0))
	doc := documentDeTest(vie)
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest(), reads: newOccupantReads(lues, mort)})
	if got := doc.Vehicles[0].Destruction; got == nil || !slices.Equal(got.Killed, []string{xuidPassager}) || got.Occupant != xuidConducteur {
		t.Errorf("trajet sur un autre joueur : destruction = %+v, attendu killed [%s] et occupant %s", got, xuidPassager, xuidConducteur)
	}
}

// TestUneVictimeSansJoueurNommableSeCompte : un bot, un index absent ou hors roster n est pas
// publie (l index ambigu suit la resolution du destructeur) ; chaque cas se compte parmi les tues sans identite, et la destruction ne porte pas de tue.
func TestUneVictimeSansJoueurNommableSeCompte(t *testing.T) {
	lues := []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1)}
	for _, cas := range []struct {
		nom     string
		victime int32
	}{{"bot", 9}, {"index absent", -1}, {"hors roster", 21}} {
		d, c := tuesAvec(t, lues, []types.ObjectDeath{mortDeBipede(paquetDestruction, cas.victime, deadStateTagAbsent)})
		if len(tuesPublies(d)) != 0 || c.Killed != 0 || c.KilledOccupants != 0 || c.KilledNoIdentity != 1 {
			t.Errorf("%s : killed = %v, couverture = %+v, attendu un seul tue sans identite", cas.nom, tuesPublies(d), c)
		}
	}
}

// TestLeNettoyageDeFinDeMancheNEstPasUnTue : l effet 4208020 ecarte la mort, qui se compte a part ; un
// autre effet la garde.
func TestLeNettoyageDeFinDeMancheNEstPasUnTue(t *testing.T) {
	lues := []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1)}
	d, c := tuesAvec(t, lues, []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, effetNettoyageDeFinDeManche)})
	if len(tuesPublies(d)) != 0 || c.KilledCleanup != 1 || c.Killed != 0 || c.KilledNoIdentity != 0 {
		t.Errorf("nettoyage : killed = %v, couverture = %+v, attendu 1 mort de nettoyage et aucun tue", tuesPublies(d), c)
	}
	d, c = tuesAvec(t, lues, []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, effetNettoyageDeFinDeManche+1)})
	if !slices.Equal(tuesPublies(d), []string{xuidPassager}) || c.KilledCleanup != 0 {
		t.Errorf("autre effet : killed = %v, couverture = %+v", tuesPublies(d), c)
	}
}

// TestUnePieceMonteeDuMemePaquetEstDansG : un bipede rattache a la tourelle du chassis, morte dans le meme
// paquet, est tue avec le chassis. Une tourelle morte dans un autre paquet, ou portee par un autre
// chassis, n en fait pas partie.
func TestUnePieceMonteeDuMemePaquetEstDansG(t *testing.T) {
	lues := []types.VehicleOccupancy{attache(27_000_000, slotPiece, 1)}
	mort := []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent)}
	for _, cas := range []struct {
		nom    string
		piece  VehicleTrack
		tue    bool
		compte int
	}{
		{"tourelle du chassis, meme paquet", pieceMontee(paquetDestruction, VehicleLifeRef{Slot: slotChassis, Gen: 1}), true, 1},
		{"tourelle du chassis, autre paquet", pieceMontee(paquetDestruction+1_000, VehicleLifeRef{Slot: slotChassis, Gen: 1}), false, 0},
		{"tourelle d un autre chassis", pieceMontee(paquetDestruction, VehicleLifeRef{Slot: 750, Gen: 1}), false, 0},
	} {
		d, c := tuesAvec(t, lues, mort, cas.piece)
		if got := slices.Equal(tuesPublies(d), []string{xuidPassager}); got != cas.tue {
			t.Errorf("%s : killed = %v, tue attendu %v", cas.nom, tuesPublies(d), cas.tue)
		}
		if c.Killed != cas.compte {
			t.Errorf("%s : couverture = %+v", cas.nom, c)
		}
	}
	d, _ := tuesAvec(t, lues, mort)
	if len(tuesPublies(d)) != 0 {
		t.Errorf("sans la tourelle dans G, le rattache a la tourelle n est pas tue : %v", tuesPublies(d))
	}
}

// TestUneVieDeBipedeNeCompteQuUneFois : le film re-replique un dead-state ; la meme vie ne se publie ni ne
// se compte deux fois, et deux vies d un meme joueur ne donnent qu un xuid.
func TestUneVieDeBipedeNeCompteQuUneFois(t *testing.T) {
	lues := []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1)}
	morts := []types.ObjectDeath{
		mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent),
		mortDeBipede(paquetDestruction+34_000, indexPassager, deadStateTagAbsent),
	}
	d, c := tuesAvec(t, lues, morts)
	if !slices.Equal(tuesPublies(d), []string{xuidPassager}) || c.KilledOccupants != 1 {
		t.Errorf("killed = %v, couverture = %+v, attendu un seul tue", tuesPublies(d), c)
	}
}

// TestLesTuesSontTriesEtSansTrajet : deux bipedes morts, victimes d index differents, tries par xuid,
// aucun trajet dans le document.
func TestLesTuesSontTriesEtSansTrajet(t *testing.T) {
	autre := types.VehicleOccupancy{TimestampUS: 27_000_000, Slot: 901, Gen: 1, Attached: true, ParentSlot: slotChassis, ParentGen: 1}
	mortAutre := mortDeBipede(paquetDestruction, 5, deadStateTagAbsent)
	mortAutre.Slot, mortAutre.Gen = 901, 1
	d, c := tuesAvec(t, []types.VehicleOccupancy{attache(27_000_000, slotChassis, 1), autre},
		[]types.ObjectDeath{mortAutre, mortDeBipede(paquetDestruction, indexPassager, deadStateTagAbsent)})
	if !slices.Equal(tuesPublies(d), []string{xuidPassager, xuidAutre}) || c.KilledOccupants != 2 || c.Killed != 1 {
		t.Errorf("killed = %v, couverture = %+v", tuesPublies(d), c)
	}
}

// TestLesBipedesMortsDesFaitsFontLAllerRetour : les dead-states de bipede voyagent dans la section des
// morts d objet des faits persistes.
func TestLesBipedesMortsDesFaitsFontLAllerRetour(t *testing.T) {
	w := &gwriter{}
	morts := []types.ObjectDeath{mortDeBipede(paquetDestruction, indexPassager, effetNettoyageDeFinDeManche)}
	encodeMortsDObjet(w, nil, morts, grammar.ObjectDeathStats{})
	if w.echec != nil {
		t.Fatal(w.echec)
	}
	_, relus, _ := decodeMortsDObjet(&greader{b: w.b})
	if len(relus) != 1 || relus[0] != morts[0] {
		t.Errorf("morts de bipede relues = %+v, attendu %+v", relus, morts)
	}
}
