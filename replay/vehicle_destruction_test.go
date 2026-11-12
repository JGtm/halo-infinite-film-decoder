package replay

// vehicle_destruction_test.go — LA DESTRUCTION D UN VEHICULE : ce que le dead-state ecrit, ce que le
// document en publie, et ce qu il refuse d affirmer. Cas synthetiques : le dead-state, le roster, les
// episodes et le fil des morts sont poses a la main ; aucun film n est ouvert.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/damagetag"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	xuidConducteur = "1001"
	xuidPassager   = "1002"
	xuidAutre      = "1003"
)

// causeConnue est un effet de degat que `damagetag` nomme : le test n en invente pas un.
func causeConnue(t *testing.T) uint32 {
	t.Helper()
	ids := damagetag.IDs()
	if len(ids) == 0 {
		t.Fatal("damagetag ne porte aucun effet de degat")
	}
	return ids[0]
}

// morteDeTest : un dead-state de vehicule detruit a `us`, de destructeur `killer` et de cause `tag`.
func morteDeTest(us uint64, killer int32, tag uint32) *vehicleDeath {
	return &vehicleDeath{
		TimestampUS: us, Slot: 700, Gen: 1, TypeIndex: 40,
		Dead: types.DeadState{Mort: true, EnumA: -1, EnumB: killer, SrcTag0: tag, SrcTag4c: deadStateTagAbsent},
	}
}

// vieDetruite : une vie `destroyed` a la frame `tEnd`, avec ses episodes.
func vieDetruite(tEnd int, mort *vehicleDeath, rides ...VehicleRide) VehicleTrack {
	return VehicleTrack{Slot: 700, Gen: 1, End: VehicleEndDestroyed, TEnd: &tEnd, T1Max: tEnd, Rides: rides, mort: mort}
}

func rosterDeTest() []RosterEntry {
	return []RosterEntry{
		{XUID: xuidConducteur, FilmIndex: 3},
		{XUID: xuidPassager, FilmIndex: 4},
		{XUID: xuidAutre, FilmIndex: 5},
		{FilmIndex: 9, Bot: true, Name: "Bot [bot]"},
	}
}

func documentDeTest(vies ...VehicleTrack) *ReplayDocument {
	return &ReplayDocument{
		Vehicles: vies,
		Coverage: &Coverage{Vehicles: &VehicleCoverage{Scanned: true}},
	}
}

func poserLesDestructions(t *testing.T, doc *ReplayDocument, in vehicleDestructionInputs) {
	t.Helper()
	attachVehicleDestructions(context.Background(), doc, in)
}

func TestDestructeurHumain(t *testing.T) {
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 4, deadStateTagAbsent)))
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	d := doc.Vehicles[0].Destruction
	if d == nil || d.By == nil || d.By.XUID != xuidPassager || d.By.I != 4 || d.By.Bot {
		t.Fatalf("destruction = %+v, attendu un destructeur humain d index 4 (xuid %s)", d, xuidPassager)
	}
	if c := doc.Coverage.Vehicles.Destructions; c.ByHuman != 1 || c.Destroyed != 1 || c.Published != 1 {
		t.Errorf("couverture = %+v", c)
	}
}

func TestDestructeurBotSansXuid(t *testing.T) {
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 9, deadStateTagAbsent)))
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	d := doc.Vehicles[0].Destruction
	if d == nil || d.By == nil || !d.By.Bot || d.By.XUID != "" || d.By.I != 9 {
		t.Fatalf("destruction = %+v, attendu un bot d index 9 sans xuid", d)
	}
	if c := doc.Coverage.Vehicles.Destructions; c.ByBot != 1 || c.ByHuman != 0 {
		t.Errorf("couverture = %+v", c)
	}
}

// TestDestructeurAbsentOuInconnu : un index absent, hors roster ou ambigu ne nomme PERSONNE — le bloc
// n a pas de `by`, et chaque cas se compte a part.
func TestDestructeurAbsentOuInconnu(t *testing.T) {
	ambigu := append(rosterDeTest(), RosterEntry{XUID: "1004", FilmIndex: 3})
	for _, cas := range []struct {
		nom    string
		roster []RosterEntry
		killer int32
		compte func(c *VehicleDestructionCoverage) int
	}{
		{"absent", rosterDeTest(), -1, func(c *VehicleDestructionCoverage) int { return c.KillerAbsent }},
		{"hors roster", rosterDeTest(), 21, func(c *VehicleDestructionCoverage) int { return c.KillerOutOfRoster }},
		{"ambigu", ambigu, 3, func(c *VehicleDestructionCoverage) int { return c.KillerAmbiguous }},
	} {
		doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, cas.killer, deadStateTagAbsent)))
		poserLesDestructions(t, doc, vehicleDestructionInputs{roster: cas.roster})
		if doc.Vehicles[0].Destruction != nil {
			t.Errorf("%s : un bloc est pose alors que rien n est etabli : %+v", cas.nom, doc.Vehicles[0].Destruction)
		}
		if n := cas.compte(doc.Coverage.Vehicles.Destructions); n != 1 {
			t.Errorf("%s : compteur = %d, attendu 1", cas.nom, n)
		}
	}
}

// TestDestructeurDepartageParLaPresence : deux entrees portent l index, la presence tranche.
func TestDestructeurDepartageParLaPresence(t *testing.T) {
	roster := []RosterEntry{
		{XUID: "2001", FilmIndex: 6, Presence: []PresenceInterval{{From: 0, To: 100}}},
		{XUID: "2002", FilmIndex: 6, Presence: []PresenceInterval{{From: 101, To: 900}}},
	}
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 6, deadStateTagAbsent)))
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: roster})
	if d := doc.Vehicles[0].Destruction; d == nil || d.By == nil || d.By.XUID != "2002" {
		t.Fatalf("destruction = %+v, attendu le remplacant 2002, present a la frame 282", d)
	}
}

func TestCauseResolueEtNonResolue(t *testing.T) {
	connue := causeConnue(t)
	for _, cas := range []struct {
		nom       string
		tag       uint32
		veutCause bool
		compte    func(c *VehicleDestructionCoverage) int
	}{
		{"resolue", connue, true, func(c *VehicleDestructionCoverage) int { return c.Cause }},
		{"non resolue", 0x7FFF0001, false, func(c *VehicleDestructionCoverage) int { return c.CauseUnresolved }},
		{"absente", deadStateTagAbsent, false, func(c *VehicleDestructionCoverage) int { return c.CauseAbsent }},
	} {
		if cas.nom == "non resolue" && damagetag.IsDamageEffect(cas.tag) {
			t.Fatalf("la valeur de test %#x est un effet de degat connu", cas.tag)
		}
		doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 3, cas.tag)))
		poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
		d := doc.Vehicles[0].Destruction
		if d == nil {
			t.Fatalf("%s : le destructeur etablit le bloc", cas.nom)
		}
		if (d.Cause != nil) != cas.veutCause || (cas.veutCause && *d.Cause != cas.tag) {
			t.Errorf("%s : cause = %v", cas.nom, d.Cause)
		}
		if n := cas.compte(doc.Coverage.Vehicles.Destructions); n != 1 {
			t.Errorf("%s : compteur = %d, attendu 1", cas.nom, n)
		}
	}
}

// TestUneSeuleDestructionParVehicule : le chassis et sa tourelle meurent au meme instant, avec le meme
// destructeur ; seul le chassis porte le bloc, la piece se compte.
func TestUneSeuleDestructionParVehicule(t *testing.T) {
	chassis := vieDetruite(282, morteDeTest(28_200_000, 3, deadStateTagAbsent))
	tourelle := vieDetruite(282, morteDeTest(28_200_000, 3, deadStateTagAbsent))
	tourelle.Slot, tourelle.Part = 701, "turret"
	doc := documentDeTest(chassis, tourelle)
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	if doc.Vehicles[0].Destruction == nil || doc.Vehicles[1].Destruction != nil {
		t.Fatalf("chassis = %+v, tourelle = %+v : une seule destruction attendue, sur le chassis",
			doc.Vehicles[0].Destruction, doc.Vehicles[1].Destruction)
	}
	if c := doc.Coverage.Vehicles.Destructions; c.Destroyed != 1 || c.Pieces != 1 || c.Published != 1 {
		t.Errorf("couverture = %+v, attendu destroyed=1 pieces=1 published=1", c)
	}
}

// TestSeulesLesFinsDetruitesPortentUnBloc : une fin `film_end` ou `unknown` n en porte pas.
func TestSeulesLesFinsDetruitesPortentUnBloc(t *testing.T) {
	vivante := vieDetruite(282, morteDeTest(28_200_000, 3, deadStateTagAbsent))
	vivante.End, vivante.TEnd = VehicleEndFilmEnd, nil
	doc := documentDeTest(vivante)
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	if doc.Vehicles[0].Destruction != nil || doc.Coverage.Vehicles.Destructions != nil {
		t.Errorf("une vie qui n est pas detruite porte un bloc ou entre au denominateur : %+v", doc.Vehicles[0])
	}
}

func rideDeTest(t0, t1 int, xuid string, seat int) VehicleRide {
	return VehicleRide{T0: t0, T1: t1, XUID: xuid, Seat: &seat, Src: "film"}
}

// TestOccupantConnuOuInconnu : le conducteur est prefere ; un episode sans xuid ne nomme personne ;
// un episode termine plus de vehicleDestructionAboardTolMS avant la destruction n est plus a bord (la marge
// se derive du pas du document ; ce document de test n'en porte pas : pas historique, 2 images).
func TestOccupantConnuOuInconnu(t *testing.T) {
	mort := morteDeTest(28_200_000, 3, deadStateTagAbsent)
	marge := framesOfMS(vehicleDestructionAboardTolMS, HistoricalFrameIntervalMS) // document de test sans pas
	for _, cas := range []struct {
		nom   string
		rides []VehicleRide
		veut  string
	}{
		{"conducteur prefere", []VehicleRide{rideDeTest(100, 300, xuidPassager, 1), rideDeTest(100, 300, xuidConducteur, 0)}, xuidConducteur},
		{"conducteur sans xuid : le passager nomme", []VehicleRide{rideDeTest(100, 300, "", 0), rideDeTest(100, 300, xuidPassager, 1)}, xuidPassager},
		{"episode sans xuid", []VehicleRide{rideDeTest(100, 300, "", 0)}, ""},
		{"fin d episode a la marge", []VehicleRide{rideDeTest(100, 282-marge, xuidConducteur, 0)}, xuidConducteur},
		{"fin d episode une image au-dela de la marge", []VehicleRide{rideDeTest(100, 282-marge-1, xuidConducteur, 0)}, ""},
		{"episode commence apres", []VehicleRide{rideDeTest(283, 300, xuidConducteur, 0)}, ""},
		{"aucun episode", nil, ""},
	} {
		doc := documentDeTest(vieDetruite(282, mort, cas.rides...))
		poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
		d := doc.Vehicles[0].Destruction
		if d == nil || d.Occupant != cas.veut {
			t.Errorf("%s : occupant = %+v, attendu %q", cas.nom, d, cas.veut)
		}
	}
}

// TestUneDestructionSansRienDEtabliNePorteAucunBloc : ni destructeur, ni cause, ni occupant.
func TestUneDestructionSansRienDEtabliNePorteAucunBloc(t *testing.T) {
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, -1, deadStateTagAbsent)))
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	if doc.Vehicles[0].Destruction != nil {
		t.Errorf("bloc pose sans rien d etabli : %+v", doc.Vehicles[0].Destruction)
	}
	if c := doc.Coverage.Vehicles.Destructions; c.Destroyed != 1 || c.Published != 0 {
		t.Errorf("couverture = %+v, attendu destroyed=1 published=0", c)
	}
}

// TestLAbsenceNeSePublieJamaisCommeUnZero : sur le JSON, un champ non etabli est ABSENT — ni 0, ni
// chaine vide, ni `null` — et l index 0 d un destructeur est bien publie.
func TestLAbsenceNeSePublieJamaisCommeUnZero(t *testing.T) {
	roster := []RosterEntry{{XUID: xuidConducteur, FilmIndex: 0}}
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 0, deadStateTagAbsent)))
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: roster})
	b, err := json.Marshal(doc.Vehicles[0].Destruction)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(b), `{"by":{"i":0,"xuid":"1001"}}`; got != want {
		t.Errorf("JSON = %s, attendu %s", got, want)
	}
	if enc, _ := json.Marshal(doc.Vehicles[0]); strings.Contains(string(enc), "mort") {
		t.Errorf("le dead-state de travail est publie : %s", enc)
	}
}

// TestLaMortSuitLaFinDansLaFusionDesRelais : `mergeVehicleRelay` prend la fin de `b`, cause et instant
// ensemble ; le dead-state qui la date la suit, sinon le destructeur serait celui d une autre vie.
func TestLaMortSuitLaFinDansLaFusionDesRelais(t *testing.T) {
	a := VehicleTrack{Slot: 700, Gen: 1, End: VehicleEndUnknown}
	b := vieDetruite(282, morteDeTest(28_200_000, 4, deadStateTagAbsent))
	b.Slot = 702
	m := mergeVehicleRelay(a, b)
	if m.End != VehicleEndDestroyed || m.mort == nil || m.mort.Dead.EnumB != 4 {
		t.Errorf("fusion = End %q mort %+v, attendu la fin et le dead-state de b", m.End, m.mort)
	}
}

// TestLaVieRetientLeDeadStateQuiLaDate : `vehicleTrackOf` pose le record le plus precoce, celui qui
// date `TEnd`.
func TestLaVieRetientLeDeadStateQuiLaDate(t *testing.T) {
	lives := []vehicleLife{vieDeTest(700, 1, 10_000_000, 40_000_000, 60_000_000, 0, 60_000_000)}
	deaths := []types.ObjectDeath{
		*morteDeTest(55_000_000, 4, deadStateTagAbsent),
		*morteDeTest(56_000_000, 5, deadStateTagAbsent),
	}
	assignVehicleDeaths(context.Background(), lives, deaths)
	tr, ok := vehicleTrackOf(lives[0], types.EquipmentCreation{TimestampUS: 10_000_000}, nil, nil, horlogeDeTest(3000))
	if !ok || tr.End != VehicleEndDestroyed || tr.mort == nil || tr.mort.Dead.EnumB != 4 || tr.endFrame() != 550 {
		t.Fatalf("vie = %+v, attendu detruite a 550 avec le dead-state du destructeur 4", tr)
	}
	vivante := vieDeTest(700, 1, 10_000_000, 40_000_000, 0, 0, ^uint64(0))
	if tr, _ := vehicleTrackOf(vivante, types.EquipmentCreation{TimestampUS: 10_000_000}, nil, nil, horlogeDeTest(3000)); tr.mort != nil {
		t.Errorf("une vie sans mort ecrite porte un dead-state : %+v", tr.mort)
	}
}

// TestSansCouvertureDeVehiculesRienNEstPose : un document dont les vehicules ne sont pas balayes.
func TestSansCouvertureDeVehiculesRienNEstPose(t *testing.T) {
	doc := documentDeTest(vieDetruite(282, morteDeTest(28_200_000, 3, deadStateTagAbsent)))
	doc.Coverage.Vehicles.Scanned = false
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	if doc.Vehicles[0].Destruction != nil || doc.Coverage.Vehicles.Destructions != nil {
		t.Errorf("un calque non balaye ne publie ni bloc ni couverture")
	}
}

// TestLesFaitsPortentLeDestructeurEtLaCause : la section des morts d objet des faits persistes
// reserialise le dead-state ENTIER. Le destructeur (`EnumB`) et la cause (`SrcTag0`) voyagent donc deja
// dans un fichier de faits cuit avant ce calque : le rejeu depuis les faits les republie sans redecoder.
func TestLesFaitsPortentLeDestructeurEtLaCause(t *testing.T) {
	cause := causeConnue(t)
	w := &gwriter{}
	encodeMortsDObjet(w, []types.ObjectDeath{*morteDeTest(28_200_000, 4, cause)}, nil, grammar.ObjectDeathStats{})
	if w.echec != nil {
		t.Fatal(w.echec)
	}
	morts, _, _ := decodeMortsDObjet(&greader{b: w.b})
	if len(morts) != 1 || morts[0].Dead.EnumB != 4 || morts[0].Dead.EnumA != -1 || morts[0].Dead.SrcTag0 != cause ||
		morts[0].TimestampUS != 28_200_000 {
		t.Errorf("morts relues = %+v, attendu le destructeur 4 et la cause %d", morts, cause)
	}
}

// TestSansDestructionLeDocumentNePorteAucuneCouverture : un rejeu sans vie detruite reste celui
// d avant, sans bloc de zeros.
func TestSansDestructionLeDocumentNePorteAucuneCouverture(t *testing.T) {
	doc := documentDeTest()
	poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
	if c := doc.Coverage.Vehicles.Destructions; c != nil {
		t.Errorf("couverture = %+v, attendu aucune", c)
	}
}

// TestLaMargeABordSuitLaCadenceDuDocument : la marge est une duree. Un episode clos 150 ms avant la
// destruction est a bord a toute cadence ; clos 250 ms avant, il ne l est plus.
func TestLaMargeABordSuitLaCadenceDuDocument(t *testing.T) {
	defaut := framesOfMS(vehicleDestructionAboardTolMS, HistoricalFrameIntervalMS)
	for _, c := range []struct {
		intervalMS, tEnd, t1 int
		aBord                bool
	}{
		{100, 300, 298, true}, {100, 300, 297, false},
		{33, 900, 894, true}, {33, 900, 893, false},
		// sans cadence ecrite : celle d'un document sans frameIntervalMs, soit le pas historique
		{0, 300, 300 - defaut, true}, {0, 300, 300 - defaut - 1, false},
	} {
		doc := documentDeTest(vieDetruite(c.tEnd, morteDeTest(1_000_000, -1, deadStateTagAbsent), rideDeTest(10, c.t1, xuidConducteur, 0)))
		doc.FrameIntervalMS = c.intervalMS
		poserLesDestructions(t, doc, vehicleDestructionInputs{roster: rosterDeTest()})
		d := doc.Vehicles[0].Destruction
		if got := d != nil && d.Occupant == xuidConducteur; got != c.aBord {
			t.Errorf("cadence %d ms, episode clos %d images avant : a bord = %v, attendu %v", c.intervalMS, c.tEnd-c.t1, got, c.aBord)
		}
	}
}
