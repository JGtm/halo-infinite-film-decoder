package grammar

// roster_listages.go — QUI EST LISTE DANS LA POPULATION DE LA SESSION, ET QUAND : la lecture de
// production des paquets de type 8 ([DecodeRoster]).
//
// Le jeu rafraichit la population de la session a chaque changement (un joueur arrive, un joueur
// part, un siege change de titulaire) en emettant un paquet de type 8 qui la liste en entier
// ([PacketTypeRoster]). Le premier paquet qui nomme un joueur date son arrivee, le premier paquet qui
// ne le nomme plus date son depart ; un joueur liste des le premier paquet est present au coup
// d envoi. Les paquets de type 8 sont rares (une dizaine par film) : la lecture parcourt les
// paquets deja decoupes de la source, elle ne lit aucun bit en dehors de la grammaire du paquet.
//
// LES BOTS NE SONT PAS ICI : ils sont declares par les paquets BOT_METADATA ([PaquetsBotMetadata]).

import (
	"bytes"
	"cmp"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ScanRosterListings lit les paquets de type 8 d un film DEJA CHARGE et rend, par humain, les plages
// pendant lesquelles il est liste. Sans registre, build inconnu ou table des joueurs refusee, la
// largeur du bloc de personnalisation n est pas connue : rien n est lu et `Scanned` reste faux.
//
// Un paquet dont la lecture deborde, ou ne rend pas toutes les entrees qu il annonce, est ecarte et
// compte ([types.RosterListingStats.Refused]) : il ne ferme ni n ouvre aucune plage.
func ScanRosterListings(film *source.Film) ([]types.RosterListing, types.RosterListingStats) {
	var st types.RosterListingStats
	chunk0, ok := FilmRegistryChunk(film)
	if !ok {
		return nil, st
	}
	version, ok := FilmFormatVersionFromHeader(chunk0)
	if !ok {
		return nil, st
	}
	ident, err := ReadFilmIdentity(chunk0)
	if err != nil {
		return nil, st
	}
	_, rep, err := ReadPlayerTable(chunk0, ident)
	if err != nil {
		return nil, st
	}
	st.Scanned = true
	suivi := &suiviDesListages{ouvertes: map[cleDeListage]int{}}
	var precedent []byte
	for _, p := range film.AllPackets() {
		if p.Type != int(PacketTypeRoster) {
			continue
		}
		st.Packets++
		if precedent != nil && bytes.Equal(precedent, p.Payload) {
			continue
		}
		precedent = p.Payload
		entrees, bilan := DecodeRoster(p.Payload, version, rep.PersoBytes*8)
		if bilan.Debordement || bilan.Entrees != bilan.Annonce {
			st.Refused++
			continue
		}
		st.Distinct++
		st.Entries += len(entrees)
		suivi.listage(p.TS, entrees)
	}
	return suivi.plages(), st
}

// cleDeListage est un humain sous un siege.
type cleDeListage struct {
	xuid uint64
	slot int
}

// suiviDesListages range les plages au fil des paquets : `ouvertes` pointe, par humain et siege, la
// plage encore ouverte dans `lus`.
type suiviDesListages struct {
	lus      []types.RosterListing
	ouvertes map[cleDeListage]int
}

// listage rend compte d un paquet : ouvre la plage de chaque humain nouvellement liste sous un siege,
// et ferme celle de chaque humain que ce paquet ne liste plus sous ce siege.
func (s *suiviDesListages) listage(ts uint64, entrees []RosterEntry) {
	presents := make(map[cleDeListage]struct{}, len(entrees))
	for _, e := range entrees {
		presents[cleDeListage{e.Identite, e.Option}] = struct{}{}
	}
	for k, i := range s.ouvertes {
		if _, ok := presents[k]; !ok {
			s.lus[i].ToUS = ts
			delete(s.ouvertes, k)
		}
	}
	for _, e := range entrees {
		k := cleDeListage{e.Identite, e.Option}
		if _, ok := s.ouvertes[k]; ok {
			continue
		}
		s.ouvertes[k] = len(s.lus)
		s.lus = append(s.lus, types.RosterListing{XUID: e.Identite, Slot: e.Option, FromUS: ts})
	}
}

// plages rend les plages lues, triees sur (debut, xuid, siege) : un ordre total.
func (s *suiviDesListages) plages() []types.RosterListing {
	out := slices.Clone(s.lus)
	slices.SortStableFunc(out, func(a, b types.RosterListing) int {
		return cmp.Or(cmp.Compare(a.FromUS, b.FromUS), cmp.Compare(a.XUID, b.XUID), cmp.Compare(a.Slot, b.Slot))
	})
	return out
}
