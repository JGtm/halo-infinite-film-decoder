package grammar

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// positions_de_vehicule.go — LES POSITIONS DE VEHICULE, LA MARCHE D ABORD.
//
// Un record delta de vehicule porte sa position dans le meme composant i0 qu un bipede, sous le meme
// decoupage. La marche des trames les lit tous dans une trame qu elle ferme : le canal des lectures
// bipedes retient le bit de leur i0, et la position se lit a ce bit par le lecteur de toujours
// ([lireLaPosition]). L ancrage bit a bit ne passe plus que derriere elle : la ou la fermeture de la
// trame ne prouve pas, pour un slot que la marche n y a pas lu ([rendParLAncrage]).

// noterLaPositionDeVehicule retient le record delta de vehicule `r` que la marche a lu quand son i0 a
// ete traverse et qu il est absolu dans la region jouee ([i0AbsoluDeLaRegion]). La bande et les
// generations sont celles du lecteur, appliquees a la lecture ([positionsDeVehicule]).
func (c *canalDesLecturesBipedes) noterLaPositionDeVehicule(p *lecture.Paquet, r *FrameRecord) {
	if !c.layOK || len(r.Trace.Comps) == 0 {
		return
	}
	i0 := r.Trace.Comps[0]
	if i0.Index != indexDeLaPosition || !i0.Ported || !i0AbsoluDeLaRegion(p.Payload, i0.StartBit, c.lay) {
		return
	}
	c.lu.vehicules.noter(p.Chunk, FilmPacket{Index: p.Index, Type: p.Type, TimestampUS: p.TS},
		positionLue{i0: uint32(i0.StartBit), slot: r.Slot, gen: uint8(r.ID >> decalageDeGeneration), //nolint:gosec // position dans un payload, generation sur 2 bits
			masque: r.Trace.Mask})
}

// positionsDeVehicule lit les positions de vehicule du film sur la bande `band` : celles des records
// que la marche des trames a lus, puis de ceux que l ancrage rend derriere elle ; et rend, comme
// [scanBipedChunks], le nombre de chunks LUS parmi `chunks`. Sans marche (pas de registre), sous un
// decoupage autre que celui du contexte ou sur des chunks choisis, l ancrage seul, sur chaque payload
// en entier.
func positionsDeVehicule(fc *FilmContext, chunks []int, band SlotBand, lay profile.I0Layout, opt ScanFilmOptions) (
	[]BipedPosition, int) {
	lu, err := fc.lecturesBipedes()
	du, errLay := fc.I0Layout()
	if err != nil || errLay != nil || du != lay || len(opt.Chunks) > 0 {
		return scanBipedChunks(fc.Film(), chunks, band, lay, opt, fc.ContexteDeLecture())
	}
	var marche, derriere positionsBipedes
	for k, p := range lu.vehicules.paquets {
		gens := opt.Generations.A(p.paquet.TimestampUS)
		for _, r := range lu.vehicules.recordsDu(k) {
			if band.Has(r.slot) && gens.Accepte(types.LifeKey{Slot: r.slot, Gen: uint32(r.gen)}) {
				marche.noter(p.chunk, p.paquet, r)
			}
		}
	}
	fin := finSelonLesTrames(lu.trames)
	read := 0
	for _, c := range chunks {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		read++
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			t, vu := lu.trames[paquetDuFlux{c, pk.Index}]
			walkDeltaBipedPayloadJusqua(pay, band, lay, opt.Generations.A(pk.TimestampUS), fin(c, pk.Index, len(pay)*8),
				func(r deltaBipedRecord) {
					if rendParLAncrage(t, vu, r.Slot, VehicleTypeIndex, r.I0) {
						derriere.noter(c, pk, positionLue{i0: uint32(r.I0), slot: r.Slot, gen: uint8(r.Gen), //nolint:gosec // position dans un payload, generation sur 2 bits
							recupere: true, masque: masqueDesIndex(r.Mask)})
					}
				})
		}
	}
	pb := fondrePositions(&marche, &derriere, fc.ChunkNumbers())
	return lireLesPositions(fc, &pb, lay, opt), read
}
