package replay

// vehicle_turrets_parent_test.go — LE PORTEUR D UNE PIECE MONTEE EST LU QUAND LE FILM LE NOMME : le handle
// d `i10` a la creation decide ; sans lecture, le repli du voisinage s applique inchange ; un handle qui ne
// designe aucun chassis connu ne pose rien et se compte. Testes sans film.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// vtPieceLue est une piece dont le record de creation nomme son parent.
func vtPieceLue(slot uint32, chassis string, parent VehicleLifeRef) VehicleTrack {
	p := vtPiece(slot, chassis)
	p.parent = &parent
	return p
}

// TestPorteurLuLEmporteSurLeVoisinage — le parent lu (slot 105) n est ni +1 ni +2 de la piece, et le
// voisin de slot 101 est de la bonne famille : le porteur est le chassis que le handle designe, le repli
// n est pas declenche.
func TestPorteurLuLEmporteSurLeVoisinage(t *testing.T) {
	tracks := []VehicleTrack{
		vtPieceLue(100, vtLAAG, VehicleLifeRef{Slot: 105, Gen: 2}),
		vtChassis(101, familleWarthog),
		vtChassis(105, familleWarthog),
	}
	fb := fallback.NouveauCompteur()
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, fb)
	if c := tracks[0].Carrier; c == nil || *c != (VehicleLifeRef{Slot: 105, Gen: 2}) {
		t.Fatalf("porteur = %+v, attendu {105 2}", c)
	}
	if tally != (turretTally{turrets: 1, onCarrier: 1, byRead: 1}) {
		t.Errorf("bilan = %+v", tally)
	}
	if len(fb.Rapport()) != 0 {
		t.Errorf("le repli ne devait pas se declencher : %+v", fb.Rapport())
	}
}

// TestPorteurLuSansTemoinDeNaissance — le chassis lu est ne 200 ms apres la piece : le temoin de naissance
// du repli ne juge pas une lecture, le porteur est pose et rien n est refuse.
func TestPorteurLuSansTemoinDeNaissance(t *testing.T) {
	piece := vtPieceLue(100, vtLAAG, VehicleLifeRef{Slot: 101, Gen: 2})
	piece.bornUS = 1_000_000
	chassis := vtChassis(101, familleWarthog)
	chassis.bornUS = 1_200_000
	tracks := []VehicleTrack{piece, chassis}
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, nil)
	if tracks[0].Carrier == nil || tally != (turretTally{turrets: 1, onCarrier: 1, byRead: 1}) {
		t.Fatalf("porteur = %+v, bilan = %+v", tracks[0].Carrier, tally)
	}
	// le meme couple SANS lecture : le temoin du repli refuse le voisin et le compte.
	nu := []VehicleTrack{vtPiece(100, vtLAAG), chassis}
	nu[0].bornUS = 1_000_000
	tally = poseTurretsOnCarriers(nu, vehicleBoardingAnchors{}, nil)
	if nu[0].Carrier != nil || tally.birthMismatch != 1 || tally.byRead != 0 {
		t.Errorf("sans lecture : porteur = %+v, bilan = %+v", nu[0].Carrier, tally)
	}
}

// TestSansLectureLAncienneRegleSApplique — aucune lecture d i10 : le voisin de slot decide, le repli se
// compte, et la piece n est pas comptee « par lecture ».
func TestSansLectureLAncienneRegleSApplique(t *testing.T) {
	tracks := []VehicleTrack{vtPiece(100, vtLAAG), vtChassis(101, familleWarthog)}
	fb := fallback.NouveauCompteur()
	tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, fb)
	if tracks[0].Carrier == nil || tally != (turretTally{turrets: 1, onCarrier: 1}) {
		t.Fatalf("porteur = %+v, bilan = %+v", tracks[0].Carrier, tally)
	}
	if fb.Rapport()[0].Nom != fallback.NomTourellePorteurVoisinDeSlot {
		t.Errorf("repli non compte : %+v", fb.Rapport())
	}
}

// TestHandleSansChassisConnuPoseRien — le handle lu designe une vie absente, une autre piece ou un chassis
// d une autre famille : aucun porteur, le handle est compte, le voisin de slot (bon et ne avec la piece) ne
// reprend pas la main et son temoin ne compte rien.
func TestHandleSansChassisConnuPoseRien(t *testing.T) {
	cas := map[string][]VehicleTrack{
		"vie absente":       {vtPieceLue(100, vtLAAG, VehicleLifeRef{Slot: 900, Gen: 2}), vtChassis(101, familleWarthog)},
		"mauvaise gen":      {vtPieceLue(100, vtLAAG, VehicleLifeRef{Slot: 101, Gen: 3}), vtChassis(101, familleWarthog)},
		"autre famille":     {vtPieceLue(100, vtLAAG, VehicleLifeRef{Slot: 101, Gen: 2}), vtChassis(101, familleMongoose)},
		"designe une piece": {vtPieceLue(100, vtLAAG, VehicleLifeRef{Slot: 102, Gen: 1}), vtChassis(101, familleWarthog), vtPiece(102, vtRoquettes)},
	}
	for nom, tracks := range cas {
		t.Run(nom, func(t *testing.T) {
			fb := fallback.NouveauCompteur()
			tally := poseTurretsOnCarriers(tracks, vehicleBoardingAnchors{}, fb)
			if tracks[0].Carrier != nil || tally.onCarrier != 0 || tally.parentUnknown < 1 || tally.birthMismatch != 0 {
				t.Errorf("porteur = %+v, bilan = %+v", tracks[0].Carrier, tally)
			}
			if len(fb.Rapport()) != 0 {
				t.Errorf("le repli ne devait pas se declencher : %+v", fb.Rapport())
			}
		})
	}
}

// TestPorteurDUnePieceLu — a la passe des positions, le record de la piece qui nomme son parent designe le
// record du chassis par son slot et sa generation, quel que soit son voisinage ; sans parent lu, le repli
// du voisin de slot ; un handle qui ne tient aucun chassis connu ne rend rien.
func TestPorteurDUnePieceLu(t *testing.T) {
	warthog := types.EquipmentCreation{Slot: 105, Gen: 2}
	warthog.MPPPresent[grammar.MPPWord32] = true
	warthog.MPPVal[grammar.MPPWord32] = 0x00002705
	voisin := warthog
	voisin.Slot = 101
	tient := map[uint32]types.EquipmentCreation{105: warthog, 101: voisin}
	creationA := func(s uint32) (types.EquipmentCreation, bool) { c, ok := tient[s]; return c, ok }
	spec := vehicleTurretByChassis[0xdd7f9102]

	piece := types.EquipmentCreation{Slot: 100, Gen: 1, HasParent: true, Parent: types.LifeKey{Slot: 105, Gen: 2}}
	if c, ok, repli := porteurDUnePiece(creationA, piece, spec); !ok || repli || c.Slot != 105 {
		t.Errorf("lu : porteur %d ok=%v repli=%v, attendu 105 par lecture", c.Slot, ok, repli)
	}
	piece.Parent.Gen = 3
	if _, ok, repli := porteurDUnePiece(creationA, piece, spec); ok || repli {
		t.Errorf("handle sans chassis connu : ok=%v repli=%v, attendu aucun porteur", ok, repli)
	}
	piece.HasParent = false
	if c, ok, repli := porteurDUnePiece(creationA, piece, spec); !ok || !repli || c.Slot != 101 {
		t.Errorf("sans lecture : porteur %d ok=%v repli=%v, attendu 101 par le repli", c.Slot, ok, repli)
	}
}
