package grammar

// world_essai.go — LES ESSAIS DU MONDE : lire sur le monde puis le rendre tel qu il etait, sans le
// copier.
//
// Un essai (la lecture d un paquet depuis une position hypothetique, la cloture d une liste) peut
// lier et delier des slots, et le monde doit en sortir EXACTEMENT comme il y est entre. Plutot que
// de copier toute la table des slots a chaque essai ([World.Snapshot]), le monde note, tant qu un
// essai est ouvert, l ancienne liaison de chaque slot qu une ecriture touche ; [World.annulerLEssai]
// la remet, dans l ordre inverse des ecritures.
//
// L INVARIANT QUE TOUT LE PAQUET TIENT : TOUTE ECRITURE de `World.slots` passe par [World.poser] ou
// [World.retirer], et par elles seules (`world_essai_test.go` le garde sur les sources). Une ecriture
// qui les contournerait survivrait a l annulation, sans aucune erreur.
//
// Seule la table des slots est rendue, comme [World.Restore] le fait : la vue courante, le rang de
// vue des images-cles, la table anticipee, le debut non prouve et les NEW suspendus ne sont ni
// copies ni restaures. Les essais peuvent s emboiter ; chacun rend le monde a l etat de son ouverture.

// annulation : la liaison qu un slot portait avant une ecriture faite pendant un essai.
type annulation struct {
	slot     uint32
	avant    slotState
	existait bool
}

// essaiOuvert : la marque d un essai ouvert, sa position dans le journal.
type essaiOuvert struct{ marque int }

// ouvrirUnEssai ouvre un essai : les ecritures suivantes sont notees jusqu a [World.annulerLEssai].
// Tout essai ouvert est annule, y compris sur une sortie par `defer`.
func (w *World) ouvrirUnEssai() essaiOuvert {
	w.essaisOuverts++
	return essaiOuvert{marque: len(w.journal)}
}

// annulerLEssai rend la table des slots a l etat de l ouverture de `e`, et clot l essai.
func (w *World) annulerLEssai(e essaiOuvert) {
	for i := len(w.journal) - 1; i >= e.marque; i-- {
		a := w.journal[i]
		if a.existait {
			w.slots[a.slot] = a.avant
		} else {
			delete(w.slots, a.slot)
		}
	}
	w.journal = w.journal[:e.marque]
	w.essaisOuverts--
}

// poser ecrit la liaison d un slot. C est l unique ecriture de la table, avec [World.retirer].
func (w *World) poser(slot uint32, s slotState) {
	w.noter(slot)
	w.slots[slot] = s
}

// retirer supprime la liaison d un slot (sans effet sur un slot non lie).
func (w *World) retirer(slot uint32) {
	w.noter(slot)
	delete(w.slots, slot)
}

// noter journalise l etat d un slot avant son ecriture, si un essai est ouvert.
func (w *World) noter(slot uint32) {
	if w.essaisOuverts == 0 {
		return
	}
	avant, existait := w.slots[slot]
	w.journal = append(w.journal, annulation{slot: slot, avant: avant, existait: existait})
}

// restoreSlots remplace la table des slots (retour arriere apres une trame desynchronisee, pour qu un
// NEW / DEL partiel ou errone ne corrompe pas un monde PERSISTANT). Hors essai la table est
// remplacee d un bloc ; pendant un essai elle l est ecriture par ecriture, pour que le journal
// annule aussi ce remplacement.
func (w *World) restoreSlots(m map[uint32]slotState) {
	if w.essaisOuverts == 0 {
		w.slots = m
		return
	}
	for s := range w.slots {
		if _, gardee := m[s]; !gardee {
			w.retirer(s)
		}
	}
	for s, l := range m {
		w.poser(s, l)
	}
}
