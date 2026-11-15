package replay

// identity_registry_listages_test.go — LE JOUEUR QUI TIENT UN SIEGE A L INSTANT D UN CORPS.
//
//	LS-RETIRE     un siege quitte par son humain, repris par un bot declare : le corps n est pas nomme sous
//	              l humain que la table garde pour cet index, il l est sous le bot (`bid`) ;
//	LS-REPRISE    un siege repris par un autre humain : le corps est nomme sous celui qui le tient ;
//	LS-TABLE      sans plage valide (lecture non faite, siege absent), la table d index nomme comme avant ;
//	LS-CONTROLE   un corps sans record de creation est nomme par la tete d `unit-control` de son slot, depuis la
//	              fin de la vie precedente ; une lecture d un autre slot ou d avant la fin de la vie precedente ne
//	              le nomme pas ;
//	LS-SIEGES     la lecture des sieges refuse une lecture sans siege ou sans paquet.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/identity"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	xuidParti  = 2533491259474651 // fictif : il tenait le siege 1 jusqu a 86 s
	xuidRepris = 2534026679052315 // fictif : il le reprend a 90 s
)

func entreeDeSiege(listings ...types.RosterListing) LecturesDePresence {
	return LecturesDePresence{Listings: listings, ListingStats: types.RosterListingStats{Scanned: true}}
}

func entreeDeLaVie(creations []grammar.BipedCreation, p LecturesDePresence, bots []BotIdentity) IdentityInput {
	return IdentityInput{BipedCreations: creations, Presences: p, Bots: bots,
		PlayerIndices: types.PlayerIndexTable{ByXUID: map[uint64]int{xuidParti: 1, xuidRepris: 2}}}
}

func creationAuSiege1(tUS uint64) []grammar.BipedCreation {
	return []grammar.BipedCreation{{Slot: 565, Generation: 1, ParticipantIndex: 1, HasIndex: true, TimestampUS: tUS}}
}

func TestSiegeQuitteParSonHumainNeNommePasLeCorps(t *testing.T) {
	bot := BotIdentity{FilmIndex: 1, Name: "343 Test [bot]", BotID: 7, BotIDLu: true,
		Declarations: [][2]uint64{{1_101_000_000, 0}}}
	p := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 1000, ToUS: 86_000_000})
	vies := func() []lifeSpan { return []lifeSpan{{slot: 565, from: 1_105_000_000, to: 1_158_000_000}} }

	lives := vies()
	rep := nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(1_105_000_000), p, []BotIdentity{bot}), nil)
	if lives[0].xuid != 0 || lives[0].bid != "bid(7.0)" || rep.ParDeclaration != 1 {
		t.Fatalf("vie %+v, rapport %+v : le bot declare au siege nomme le corps", lives[0], rep)
	}
	// Sans bot declare : le xuid de la table n est pas pose, la vie reste sans nom et la lecture se compte.
	lives = vies()
	rep = nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(1_105_000_000), p, nil), nil)
	if lives[0].xuid != 0 || rep.HumainRetire != 1 || rep.causes[0] != identity.MethodIndexOutOfTable {
		t.Fatalf("vie %+v, rapport %+v : l humain parti ne nomme pas le corps", lives[0], rep)
	}
	// Sans listage : la table nomme comme avant (le defaut que la lecture corrige).
	lives = vies()
	nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(1_105_000_000), LecturesDePresence{}, nil), nil)
	if lives[0].xuid != xuidParti {
		t.Fatalf("vie %+v : sans listage la table d index nomme", lives[0])
	}
}

func TestSiegeRepris(t *testing.T) {
	p := entreeDeSiege(
		types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 1000, ToUS: 86_000_000},
		types.RosterListing{XUID: xuidRepris, Slot: 1, FromUS: 90_000_000})
	lives := []lifeSpan{{slot: 565, from: 100_000_000, to: 120_000_000}}
	rep := nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(100_000_000), p, nil), nil)
	if lives[0].xuid != xuidRepris || rep.ParListage != 1 {
		t.Fatalf("vie %+v, rapport %+v : le siege est a l humain qui le tient a la creation", lives[0], rep)
	}
}

func TestSiegesListesRefusentUneLectureSansSiege(t *testing.T) {
	cas := map[string]LecturesDePresence{
		"non lue":    {Listings: []types.RosterListing{{XUID: 1, Slot: 0}}},
		"sans plage": {ListingStats: types.RosterListingStats{Scanned: true}},
		"sans siege": {Listings: []types.RosterListing{{XUID: 1, Slot: -1}}, ListingStats: types.RosterListingStats{Scanned: true}},
	}
	for nom, p := range cas {
		if nouveauxSiegesListes(p).valide {
			t.Errorf("%s : la lecture ne doit pas etre valide", nom)
		}
	}
	if !nouveauxSiegesListes(entreeDeSiege(types.RosterListing{XUID: 1, Slot: 3})).valide {
		t.Error("une lecture avec siege est valide")
	}
}

func TestCorpsSansRecordNommeParLeControle(t *testing.T) {
	p := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 1000})
	p.ControlStats = types.UnitControlStats{Scanned: true}
	p.Controls = []types.UnitControlRead{
		{Slot: 565, TimestampUS: 10_000_000, Controller: 1},  // vie 1
		{Slot: 565, TimestampUS: 40_000_000, Controller: -1}, // fin de la vie 1
		{Slot: 565, TimestampUS: 60_000_000, Controller: 2},  // vie 2 : siege 2, aucun listage
		{Slot: 566, TimestampUS: 10_000_000, Controller: 1},  // autre slot
	}
	lives := []lifeSpan{{slot: 565, from: 11_000_000, to: 40_000_000}, {slot: 565, from: 61_000_000, to: 80_000_000}}
	rep := nommerAvecLesTables(lives, entreeDeLaVie(nil, p, nil), nil)
	if lives[0].xuid != xuidParti || lives[0].nomPar != NomParCreation || rep.ParControle != 1 {
		t.Fatalf("vie 1 : %+v, rapport %+v : la tete d unit-control nomme le corps", lives[0], rep)
	}
	if lives[1].xuid != 0 || rep.causes[1] != identity.MethodNoCreationRecord {
		t.Fatalf("vie 2 : %+v, causes %v : le siege 2 n a aucun humain liste, la cause d origine reste", lives[1], rep.causes)
	}
}

// LS-RANG : la table d un film a vacant intercale est employee quand les plages de listage donnent a chaque
// occupe, des sa premiere plage, le siege de son rang absolu ; un seul siege qui differe, ou une lecture sans
// siege, la refuse. Les sieges du debut que la pose des places lit suivent cette decision.
func TestRangAbsoluConfirmeParLesListages(t *testing.T) {
	table := grammar.FilmPlayerTable{Refusal: grammar.FilmTableRead, InterleavedVacant: true,
		Seats: []grammar.FilmPlayerSeat{{FilmIndex: 0, XUID: xuidParti}, {FilmIndex: 3, XUID: xuidRepris}}}
	bon := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 0, FromUS: 10},
		types.RosterListing{XUID: xuidRepris, Slot: 3, FromUS: 10})
	autre := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 0, FromUS: 10},
		types.RosterListing{XUID: xuidRepris, Slot: 1, FromUS: 10})
	if !rangAbsoluConfirmeParLesListages(table, bon) || !tableDuFilmEmployee(IdentityInput{FilmTable: table, Presences: bon}) {
		t.Error("les listages donnent le rang absolu : la table est employee")
	}
	if rangAbsoluConfirmeParLesListages(table, autre) || rangAbsoluConfirmeParLesListages(table, LecturesDePresence{}) {
		t.Error("un siege different, ou aucune lecture : la table n est pas confirmee")
	}
	sansVacant := table
	sansVacant.InterleavedVacant = false
	if rangAbsoluConfirmeParLesListages(sansVacant, bon) {
		t.Error("sans vacant intercale la question ne se pose pas")
	}
	if got := siegesDuDebut(table, true); !got[0] || !got[3] || len(got) != 2 {
		t.Errorf("sieges du debut %v : la table employee donne ses sieges", got)
	}
	if siegesDuDebut(table, false) != nil {
		t.Error("table a vacant intercale non employee : aucun siege du debut")
	}
}

// LS-IMAGE-CLE : un corps cree avant le premier paquet de type 8 de son joueur (image-cle du debut du film, arrivant dont
// le corps precede le listage de quelques centaines de millisecondes) garde son nom : la plage vaut depuis l image-cle.
func TestCorpsCreeAvantLeListageGardeSonNom(t *testing.T) {
	p := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 300_000})
	lives := []lifeSpan{{slot: 565, from: 1_000_000, to: 20_000_000}}
	rep := nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(0), p, nil), nil)
	if lives[0].xuid != xuidParti || rep.HumainRetire != 0 {
		t.Fatalf("vie %+v, rapport %+v : le listage a 0,3 s nomme le corps cree a 0 s", lives[0], rep)
	}
	loin := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 5_000_000})
	lives = []lifeSpan{{slot: 565, from: 1_000_000, to: 20_000_000}}
	rep = nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(0), loin, nil), nil)
	if lives[0].xuid != 0 || rep.HumainRetire != 1 {
		t.Fatalf("vie %+v, rapport %+v : un listage 5 s apres la creation ne la nomme pas", lives[0], rep)
	}
}

// LS-HORS-TABLE : un siege tenu par A dans la table et repris par B, que les plages listent mais que la table ne
// nomme pas, ne donne jamais le corps de B a A.
func TestSiegeReprisParUnHumainHorsTableNeNommePasLAncien(t *testing.T) {
	const xuidAbsent = 2534144863254370 // fictif (registre), absent de la table
	p := entreeDeSiege(
		types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 1000, ToUS: 86_000_000},
		types.RosterListing{XUID: xuidAbsent, Slot: 1, FromUS: 90_000_000})
	lives := []lifeSpan{{slot: 565, from: 100_000_000, to: 120_000_000}}
	rep := nommerAvecLesTables(lives, entreeDeLaVie(creationAuSiege1(100_000_000), p, nil), nil)
	if lives[0].xuid != 0 || rep.HumainRetire != 1 || rep.causes[0] != identity.MethodIndexOutOfTable {
		t.Fatalf("vie %+v, rapport %+v : le corps n est pas celui de la table", lives[0], rep)
	}
}

// LS-COMPTE : une lecture d unit-control qui ne nomme pas la vie ne compte aucune decision, et ses controles ne
// sont lus que sur un film dont la lecture a eu lieu.
func TestLaTentativeParLeControleNeCompteRienSiElleEchoue(t *testing.T) {
	p := entreeDeSiege(types.RosterListing{XUID: xuidParti, Slot: 1, FromUS: 1000, ToUS: 5_000_000})
	p.ControlStats = types.UnitControlStats{Scanned: true}
	p.Controls = []types.UnitControlRead{{Slot: 565, TimestampUS: 10_000_000, Controller: 1}}
	lives := []lifeSpan{{slot: 565, from: 11_000_000, to: 30_000_000}}
	rep := nommerAvecLesTables(lives, entreeDeLaVie(nil, p, nil), nil)
	if rep.HumainRetire != 0 || rep.ParControle != 0 || len(rep.indexLu) != 0 || rep.causes[0] != identity.MethodNoCreationRecord {
		t.Fatalf("rapport %+v : la tentative refusee ne laisse ni compte ni index lu", rep)
	}
	p.ControlStats = types.UnitControlStats{}
	p.Listings[0].ToUS = 0
	lives = vies565()
	rep = nommerAvecLesTables(lives, entreeDeLaVie(nil, p, nil), nil)
	if lives[0].xuid != 0 || rep.ParControle != 0 {
		t.Fatalf("vie %+v, rapport %+v : sans lecture faite, les controles ne nomment pas", lives[0], rep)
	}
}

func vies565() []lifeSpan { return []lifeSpan{{slot: 565, from: 11_000_000, to: 30_000_000}} }
