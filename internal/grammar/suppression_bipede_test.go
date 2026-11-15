package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// enregistrement fabrique un record de la vue B pour le canal des suppressions.
func enregistrement(genre lecture.Genre, ti int16, slot, gen uint32) lecture.Record {
	return lecture.Record{Genre: genre, TI: ti, Desync: lecture.SansDesynchronisation,
		Vie: types.LifeKey{Slot: slot, Gen: gen}}
}

func paquetDeRecords(ts uint64, debut lecture.DebutDeVueB, recs ...lecture.Record) *lecture.Paquet {
	return &lecture.Paquet{Chunk: 1, Index: 7, TS: ts, Debut: debut, Records: recs}
}

func TestSuppressionsDeBipedes(t *testing.T) {
	const autre = 40 // un archetype qui n est pas le bipede
	cas := []struct {
		nom      string
		paquets  []*lecture.Paquet
		slots    []uint32
		attendus types.BipedDeletionStats
	}{
		{"DEL d un slot bipede de la meme generation", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreDelta, BipedTypeIndex, 5, 1)),
			paquetDeRecords(20, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 5, 1)),
		}, []uint32{5}, types.BipedDeletionStats{Deletions: 1}},
		{"DEL d un slot cree par un NEW de bipede", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreNeuf, BipedTypeIndex, 6, 2)),
			paquetDeRecords(20, lecture.DebutParSignature, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 6, 2)),
		}, []uint32{6}, types.BipedDeletionStats{Deletions: 1}},
		{"DEL d un autre archetype : rien", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreNeuf, autre, 8, 0)),
			paquetDeRecords(20, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 8, 0)),
		}, nil, types.BipedDeletionStats{}},
		{"DEL d un slot jamais vu : rien", []*lecture.Paquet{
			paquetDeRecords(20, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 9, 0)),
		}, nil, types.BipedDeletionStats{}},
		{"DEL d une generation ancienne : rien", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreNeuf, BipedTypeIndex, 5, 2)),
			paquetDeRecords(20, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 5, 1)),
		}, nil, types.BipedDeletionStats{}},
		{"un autre archetype reprend le slot : le DEL qui suit n est pas un bipede", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreDelta, BipedTypeIndex, 5, 1)),
			paquetDeRecords(20, lecture.DebutEnTete, enregistrement(lecture.GenreNeuf, autre, 5, 1)),
			paquetDeRecords(30, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 5, 1)),
		}, nil, types.BipedDeletionStats{}},
		{"DEL d une marche non prouvee : compte, non publie", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreDelta, BipedTypeIndex, 5, 1)),
			paquetDeRecords(20, lecture.DebutParFermetureAuBit, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 5, 1)),
		}, nil, types.BipedDeletionStats{DeletionsUnproven: 1}},
		{"une suppression ne se publie qu une fois", []*lecture.Paquet{
			paquetDeRecords(10, lecture.DebutEnTete, enregistrement(lecture.GenreDelta, BipedTypeIndex, 5, 1)),
			paquetDeRecords(20, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 5, 1)),
			paquetDeRecords(30, lecture.DebutEnTete, enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 5, 1)),
		}, []uint32{5}, types.BipedDeletionStats{Deletions: 1}},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			var s suppressionsDeBipedes
			for _, p := range c.paquets {
				s.trame(p)
			}
			lues := s.publier()
			if len(lues) != len(c.slots) {
				t.Fatalf("%d suppression(s) publiee(s), attendu %d : %+v", len(lues), len(c.slots), lues)
			}
			for i, slot := range c.slots {
				if lues[i].Slot != slot || lues[i].Chunk != 1 || lues[i].PacketIndex != 7 {
					t.Errorf("suppression %d = %+v, slot attendu %d", i, lues[i], slot)
				}
			}
			if s.stats != c.attendus {
				t.Errorf("comptes %+v, attendu %+v", s.stats, c.attendus)
			}
		})
	}
}

func TestSuppressionsDeBipedes_OrdreEtHorodatage(t *testing.T) {
	var s suppressionsDeBipedes
	s.trame(paquetDeRecords(10, lecture.DebutEnTete,
		enregistrement(lecture.GenreDelta, BipedTypeIndex, 9, 0), enregistrement(lecture.GenreDelta, BipedTypeIndex, 3, 0)))
	s.trame(paquetDeRecords(20, lecture.DebutEnTete,
		enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 9, 0),
		enregistrement(lecture.GenreSuppression, lecture.TINonResolu, 3, 0)))
	lues := s.publier()
	if len(lues) != 2 || lues[0].Slot != 3 || lues[1].Slot != 9 || lues[0].TimestampUS != 20 {
		t.Fatalf("ordre (instant, slot) attendu : %+v", lues)
	}
}
