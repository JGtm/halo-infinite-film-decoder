package grammar

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// noterLaVisee retient la visee du record `r` que la marche a lu quand il ne porte pas de position :
// son masque n annonce pas i0, annonce la visee, et rien d autre avant elle que les composants que la
// passe de la visee seule sait traverser ([aimOnlyCursorToI21]). La population est celle de cette
// passe : la bande bipede du contexte et les generations vivantes a l instant du paquet.
func (c *canalDesLecturesBipedes) noterLaVisee(p *lecture.Paquet, r *FrameRecord, vie types.LifeKey,
	gens *GenerationsVivantes) {
	m := r.Trace.Mask
	if m&1 != 0 || m>>indexDeLaVisee&1 == 0 || m&(1<<indexDeLaVisee-1)&^composantsAvantLaVisee != 0 {
		return
	}
	if !c.bande.Has(r.Slot) || !gens.Accepte(vie) {
		return
	}
	for _, comp := range r.Trace.Comps {
		if comp.Index == indexDeLaVisee {
			var d componentDirs
			readAimingVectorComponent(p.Payload, comp.StartBit, len(p.Payload)*8, &d)
			if d.HasYaw {
				c.lu.visees = append(c.lu.visees, BipedAim{Slot: r.Slot, Chunk: p.Chunk, PacketIndex: p.Index,
					TimestampUS: p.TS, YawRaw: d.YawRaw, PitchRaw: d.PitchRaw, bit: r.HeaderBit})
			}
			return
		}
		if !comp.Ported {
			return
		}
	}
}
