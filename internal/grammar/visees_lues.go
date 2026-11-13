package grammar

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// noterLaVisee retient la visee du record `r` que la marche a lu quand il ne porte pas de position :
// son masque n annonce pas i0, annonce la visee, et rien d autre avant elle que les composants que la
// passe de la visee seule sait traverser ([aimOnlyCursorToI21]). La population est celle de cette
// passe : la bande bipede du contexte ; les generations vivantes a l instant du paquet se jugent a la
// cloture ([viseesDesCorpsVivants]).
func (c *canalDesLecturesBipedes) noterLaVisee(p *lecture.Paquet, r *FrameRecord, vie types.LifeKey) {
	m := r.Trace.Mask
	if m&1 != 0 || m>>indexDeLaVisee&1 == 0 || m&(1<<indexDeLaVisee-1)&^composantsAvantLaVisee != 0 {
		return
	}
	if !c.bande.Has(r.Slot) {
		return
	}
	for _, comp := range r.Trace.Comps {
		if comp.Index == indexDeLaVisee {
			var d componentDirs
			readAimingVectorComponent(p.Payload, comp.StartBit, len(p.Payload)*8, &d)
			if d.HasYaw {
				c.lu.visees = append(c.lu.visees, viseeLue{BipedAim: BipedAim{Slot: r.Slot, Chunk: p.Chunk,
					PacketIndex: p.Index, TimestampUS: p.TS, YawRaw: d.YawRaw, PitchRaw: d.PitchRaw}, bit: r.HeaderBit,
					gen: uint8(vie.Gen)}) //nolint:gosec // generation sur 2 bits
			}
			return
		}
		if !comp.Ported {
			return
		}
	}
}

// viseesDesCorpsVivants rend, dans leur ordre, les visees de `v` dont le corps est vivant a l instant
// de son paquet sous `gens` : la garde des generations vivantes datees, celle de la passe.
func viseesDesCorpsVivants(v []viseeLue, gens *GenerationsVivantes) []viseeLue {
	out := v[:0]
	var a *GenerationsVivantes
	var instant uint64
	for i := range v {
		if a == nil || v[i].TimestampUS != instant {
			a, instant = gens.A(v[i].TimestampUS), v[i].TimestampUS
		}
		if a.Accepte(types.LifeKey{Slot: v[i].Slot, Gen: uint32(v[i].gen)}) {
			out = append(out, v[i])
		}
	}
	return out
}
