package grammar

import (
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// journal_des_bipedes.go — LES CREATIONS DE BIPEDE PAR LA MARCHE, ET LE JOURNAL QUI LES ATTEND.
//
// Un record de creation de bipede est un record NEW comme un autre : la marche des trames le lit a
// son en-tete, et son prologue se lit a ce bit par le lecteur de toujours
// ([bipedCreationWalk.readCreation]). La passe bit a bit ne tourne plus que derriere elle : la ou la
// fermeture d une trame ne prouve pas, pour un slot que la marche n y a pas lu ([rendParLAncrage]).
//
// Les generations vivantes du film se relevent sur ces creations et sur les images-cles : elles ne
// sont connues qu a la fin de la marche. Le canal des lectures bipedes ne juge donc plus un record
// pendant la marche : il le porte a un journal, dans l ordre du flux, et le journal se rejoue a la
// cloture sous les generations vivantes du film — les memes decisions, dans le meme ordre.

// bipedeAuJournal est un record bipede que la marche a lu : une creation (qui rouvre la vie), ou un
// delta avec ce que le canal en retiendra si son corps est vivant.
type bipedeAuJournal struct {
	vie      types.LifeKey
	creation bool
	// mort : le record porte un etat de mort ecrit.
	mort   bool
	chunk  int
	paquet FilmPacket
	// position : ou se lit sa position, si aUnePosition.
	position     positionLue
	aUnePosition bool
	// rang : son rang dans l attente du canal quand il annonce un composant des huit lecteurs ; -1.
	rang int
}

// rejouerLeJournal applique au journal, dans l ordre du flux, les gardes du canal : un corps n est
// pas lu avant son record de creation ni sous une generation que le film ne connait pas
// (`gens`, date a l instant du paquet) ; un corps mort n est plus lu jusqu a sa creation suivante.
func (c *canalDesLecturesBipedes) rejouerLeJournal(gens *GenerationsVivantes) {
	var a *GenerationsVivantes
	var instant uint64
	for i := range c.journal {
		e := &c.journal[i]
		if e.creation {
			delete(c.mortA, e.vie)
			continue
		}
		if a == nil || e.paquet.TimestampUS != instant {
			a, instant = gens.A(e.paquet.TimestampUS), e.paquet.TimestampUS
		}
		if !a.Accepte(e.vie) {
			c.lu.generationsRefusees++
			continue
		}
		if _, deja := c.mortA[e.vie]; !deja && e.mort {
			c.mortA[e.vie] = e.paquet.TimestampUS
		}
		if _, mort := c.mortA[e.vie]; mort {
			c.lu.corpsMorts++
			continue
		}
		if e.aUnePosition {
			c.positionsMarche.noter(e.chunk, e.paquet, e.position)
		}
		if e.rang >= 0 {
			c.lu.records = append(c.lu.records, c.attente[e.rang])
		}
	}
	c.journal, c.attente = nil, nil
}

// creationsDeLaMarche releve les creations de bipede d un film : celles des records NEW que la marche
// lit, puis celles que la passe rend derriere elle.
type creationsDeLaMarche struct {
	fc    *FilmContext
	w     bipedCreationWalk
	st    types.BipedCreationStats
	lues  []BipedCreation
	actif bool
}

// nouvellesCreationsDeLaMarche prepare le releve sur la bande `band`. Sans chunk ou sans bande, il
// ne releve rien et ne range rien : la passe seule rend alors son erreur ([ScanBipedCreations]).
func nouvellesCreationsDeLaMarche(fc *FilmContext, band SlotBand) *creationsDeLaMarche {
	cm := &creationsDeLaMarche{fc: fc, actif: len(fc.ChunkNumbers()) > 0 && band.Count() > 0}
	cm.st.Slots = band.Count()
	cm.w = bipedCreationWalk{prof: fc.ProfilDeBalayage(), band: band, st: &cm.st, autres: map[uint32]int{}}
	return cm
}

// lire lit la creation du record NEW de bipede `r` de la trame `p`, a son en-tete, comme la passe la
// lirait a ce bit.
func (cm *creationsDeLaMarche) lire(p *lecture.Paquet, r *FrameRecord) {
	if cm == nil || !cm.actif || !cm.w.band.Has(r.Slot) {
		return
	}
	cm.st.Anchors++
	cre, ok := cm.w.readCreation(p.Payload, r.HeaderBit, len(p.Payload)*8)
	if !ok {
		return
	}
	cre.Slot, cre.Generation, cre.BitPos = r.Slot, r.ID>>decalageDeGeneration, r.HeaderBit
	cre.Chunk, cre.PacketIndex, cre.TimestampUS = p.Chunk, p.Index, p.TS
	cm.st.Accepted++
	cm.lues = append(cm.lues, cre)
}

// ranger fait passer la passe derriere la marche (`trames` : ce que la marche a rendu de chaque
// trame), range les creations dans l ordre du flux et les donne au contexte.
func (cm *creationsDeLaMarche) ranger(trames map[paquetDuFlux]trameDuCanal) {
	if cm == nil || !cm.actif || cm.fc.vies.creationsLues {
		return
	}
	fin := finSelonLesTrames(trames)
	chunks := cm.fc.ChunkNumbers()
	for _, c := range chunks {
		data, pks, ok := cm.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			t, vu := trames[paquetDuFlux{c, pk.Index}]
			cm.lues = append(cm.lues, cm.w.scanPayloadJusqua(pay, pk, c, fin(c, pk.Index, len(pay)*8),
				func(slot uint32, bit int) bool { return rendParLAncrage(t, vu, slot, BipedTypeIndex, bit) })...)
		}
	}
	rang := make(map[int]int, len(chunks))
	for i, c := range chunks {
		rang[c] = i
	}
	slices.SortStableFunc(cm.lues, func(a, b BipedCreation) int {
		return cmp.Or(cmp.Compare(rang[a.Chunk], rang[b.Chunk]), cmp.Compare(a.PacketIndex, b.PacketIndex),
			cmp.Compare(a.BitPos, b.BitPos))
	})
	cm.st.OtherWord, cm.st.OtherWordCount = motAlternatifModal(cm.w.autres)
	// Des creations deja rangees au contexte (une lecture anterieure a cette marche) restent les siennes.
	if v := &cm.fc.vies; !v.creationsLues {
		v.creations, v.stats, v.err, v.creationsLues = cm.lues, cm.st, nil, true
	}
}
