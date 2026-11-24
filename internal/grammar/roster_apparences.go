package grammar

// roster_apparences.go — L'APPARENCE DES SPARTANS : le bloc de personnalisation de chaque humain dans les paquets
// de type 8, livre en codes bruts.
//
// La table de `chunk_00` porte ce bloc ENTIEREMENT A ZERO ; les paquets de type 8 le portent rempli, avec
// le meme corps d'enregistrement ([DecodeRoster]). Sa largeur est celle du profil du build
// ([profile.PersonnalisationOctets]). Le decodeur n'interprete rien : voir [types.AppearancePair].
//
// LA REGLE DU CHANGEMENT. Un paquet de type 8 est renvoye a chaque changement de la population, et le bloc d'un
// joueur peut changer d'un paquet a l'autre (un bloc nul d'abord, puis le bloc charge ; ou un autre equipement).
// Le DERNIER BLOC NON NUL fait foi ; un bloc nul n'efface jamais un bloc lu. Un joueur est « change » quand deux
// blocs non nuls successifs different ; un bloc nul suivi du meme bloc ne change rien.

import (
	"bytes"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

const (
	// apparenceTeteOctets : la tete du bloc, un compteur de paires sur 32 bits puis un mot qui vaut zero.
	apparenceTeteOctets = 8
	// apparencePaireOctets : une paire, deux mots de 32 bits.
	apparencePaireOctets = 8
	// apparencePairesMax : le compteur au-dela duquel la tete n'est pas une tete de paires. Mesure : 22 sur tous les
	// builds du corpus ; la borne laisse de la marge sans accepter un mot quelconque.
	apparencePairesMax = 256
	// blocDeQueueBits : la largeur du bloc de 44 octets qui suit le bloc de personnalisation dans une entree.
	blocDeQueueBits = 352
)

// ScanRosterAppearances lit, dans les paquets de type 8 d'un film DEJA CHARGE, l'apparence de chaque humain. Sans
// registre, ou sans section d'identification, rien n'est lu et `Scanned` reste faux ; un build absent du profil
// lit zero apparence et le dit (`UnknownBuild`). La liste rendue est triee sur le xuid.
func ScanRosterAppearances(film *source.Film) ([]types.PlayerAppearance, types.AppearanceStats) {
	var st types.AppearanceStats
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
	st.Scanned = true
	octets, connu := profile.PersonnalisationOctets(ident.Build)
	if !connu {
		st.UnknownBuild = true
		return nil, st
	}
	return lireLesApparences(film.AllPackets(), version, octets, st)
}

// lireLesApparences est la lecture proprement dite, sur les paquets d'un film : `octets` est la largeur du bloc du
// build, `version` la version de format du registre, `st` les compteurs deja poses par l'appelant.
func lireLesApparences(paquets []types.Packet, version, octets int, st types.AppearanceStats,
) ([]types.PlayerAppearance, types.AppearanceStats) {
	suivi := &suiviDesApparences{dernier: map[uint64]*apparenceSuivie{}}
	var precedent []byte
	for _, p := range paquets {
		if p.Type != int(PacketTypeRoster) {
			continue
		}
		st.Packets++
		if precedent != nil && bytes.Equal(precedent, p.Payload) {
			continue
		}
		precedent = p.Payload
		entrees, bilan := DecodeRoster(p.Payload, version, octets*8)
		if bilan.Debordement || bilan.Entrees != bilan.Annonce {
			st.Refused++
			continue
		}
		st.Distinct++
		for _, e := range entrees {
			if e.Identite == 0 {
				st.NoXUID++
				continue
			}
			bloc, ok := octetsDuFlux(p.Payload, e.Joueur.Bit+e.Joueur.TotalBits-blocDeQueueBits-octets*8, octets)
			if !ok {
				st.Refused++
				continue
			}
			suivi.vu(e.Identite, bloc)
		}
	}
	out := suivi.apparences(&st)
	return out, st
}

// apparenceSuivie : ce que le film a dit d'un joueur jusqu'ici.
type apparenceSuivie struct {
	bloc   []byte // le dernier bloc non nul ; nil tant que tous sont nuls
	change bool
}

// suiviDesApparences range le dernier bloc non nul de chaque joueur au fil des paquets.
type suiviDesApparences struct {
	dernier map[uint64]*apparenceSuivie
}

// vu rend compte du bloc d'un joueur dans un paquet : un bloc nul ne change rien, un bloc non nul different du
// dernier non nul le remplace et marque le joueur « change ».
func (s *suiviDesApparences) vu(xuid uint64, bloc []byte) {
	a := s.dernier[xuid]
	if a == nil {
		a = &apparenceSuivie{}
		s.dernier[xuid] = a
	}
	if estNul(bloc) {
		return
	}
	if a.bloc != nil && !bytes.Equal(a.bloc, bloc) {
		a.change = true
	}
	a.bloc = bloc
}

// apparences rend l'apparence de chaque joueur lu (triee sur le xuid) et pose les compteurs de joueurs.
func (s *suiviDesApparences) apparences(st *types.AppearanceStats) []types.PlayerAppearance {
	xuids := make([]uint64, 0, len(s.dernier))
	for x := range s.dernier {
		xuids = append(xuids, x)
	}
	slices.Sort(xuids)
	var out []types.PlayerAppearance
	for _, x := range xuids {
		a := s.dernier[x]
		if a.bloc == nil {
			st.Unloaded++
			continue
		}
		if a.change {
			st.Changed++
		}
		pa := decouperLeBloc(a.bloc)
		pa.XUID = x
		if len(pa.Pairs) > 0 {
			st.Pairs++
		} else {
			st.Raw++
		}
		out = append(out, pa)
	}
	return out
}

// decouperLeBloc range les octets d'un bloc non nul : en paires quand la tete se lit proprement, en brut sinon.
// Chaque octet est range une fois ; la queue de zeros de `Rest` est omise (la largeur est `Size`).
func decouperLeBloc(bloc []byte) types.PlayerAppearance {
	pa := types.PlayerAppearance{Size: len(bloc)}
	debut := 0
	if n, ok := teteDePaires(bloc); ok {
		pa.Pairs = make([]types.AppearancePair, n)
		for i := range n {
			o := apparenceTeteOctets + i*apparencePaireOctets
			pa.Pairs[i] = types.AppearancePair{Key: mot32(bloc[o:]), Value: mot32(bloc[o+4:])}
		}
		debut = apparenceTeteOctets + n*apparencePaireOctets
	}
	if reste := bytes.TrimRight(bloc[debut:], "\x00"); len(reste) > 0 {
		pa.Rest = bytes.Clone(reste)
	}
	return pa
}

// teteDePaires dit si le bloc commence par une tete lisible : un compteur de paires entre 1 et
// [apparencePairesMax], un mot nul, et des paires qui tiennent dans le bloc.
func teteDePaires(bloc []byte) (int, bool) {
	if len(bloc) < apparenceTeteOctets {
		return 0, false
	}
	n := int(mot32(bloc))
	if n < 1 || n > apparencePairesMax || mot32(bloc[4:]) != 0 {
		return 0, false
	}
	if apparenceTeteOctets+n*apparencePaireOctets > len(bloc) {
		return 0, false
	}
	return n, true
}

// mot32 lit un mot de 32 bits en petit-boutiste.
func mot32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func estNul(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}

// octetsDuFlux lit `n` octets a partir du bit `debut` d'un paquet, dans l'ordre du flux (les bits se lisent
// poids fort d'abord). Rend faux quand la plage sort du paquet.
func octetsDuFlux(pay []byte, debut, n int) ([]byte, bool) {
	if debut < 0 || debut+n*8 > len(pay)*8 {
		return nil, false
	}
	br := LecteurSur(pay)
	br.SetBitPos(debut)
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(br.ReadBits(8))
	}
	return out, true
}
