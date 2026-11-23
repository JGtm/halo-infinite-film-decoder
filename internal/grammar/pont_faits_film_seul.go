package grammar

// pont_faits_film_seul.go — CE QUE LE PONT D IDENTITE LIT DU FILM SEUL.
//
// Quatre des lectures du pont ne dependent que du film charge (immuable) et de la liste des
// joueurs : les temps forts (morts et medailles), la table d index de joueur et l origine d horloge.
// Elles ne touchent ni le contexte de film, ni le profil de balayage, ni le rapport des replis. Les
// isoler en une fonction pure permet a un appelant de les lancer a part (par exemple pendant la
// marche des trames) puis de les remettre au pont par [OptionsDuPont.FaitsDuFilm].
//
// LA FONCTION NE VERSE JAMAIS UN REPLI. Elle constate deux comptes et les rend ; c est le pont qui
// les verse au rapport de son contexte, au meme endroit et dans le meme ordre que la lecture en
// serie, que les faits aient ete calcules a l avance ou par le pont lui-meme.

import (
	"github.com/JGtm/halo-infinite-film-decoder/highlightevent"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// FaitsDuFilmSeul : les lectures du pont qui ne dependent que du film et du roster. Ses champs ne
// sont pas exportes : seule [LireLesFaitsDuFilmSeul] en produit, et le pont seul les consomme.
type FaitsDuFilmSeul struct {
	morts         []types.Death
	errMorts      error
	medailles     []highlightevent.MedalAward
	medaillesLues bool

	indexLu  bool
	index    types.PlayerIndexTable
	errIndex error

	origineUS  uint64
	errOrigine error

	// Les deux comptes de replis constates, a verser par le pont (et par lui seul).
	tempsFortsAuDernierNumero bool
	chunksDeReplicationSautes int
}

// LireLesFaitsDuFilmSeul lit les temps forts, la table d index et l origine d horloge d un film
// DEJA CHARGE. `roster` construit la liste des joueurs de la table d index a partir du fil des morts
// lu ; il n est appele que si ce fil a rendu au moins une mort, et nil laisse la table non lue.
//
// SANS ETAT PARTAGE : la fonction ne lit que le film (immuable apres chargement) et n ecrit aucune
// variable de paquet ; elle n a pas de contexte de film et ne verse aucun repli. `roster` doit etre
// pure si l appelant l execute hors du fil du pont.
func LireLesFaitsDuFilmSeul(film *source.Film, roster func([]types.Death) []uint64) FaitsDuFilmSeul {
	var f FaitsDuFilmSeul
	tf := scanTempsForts(film)
	f.morts, f.errMorts, f.medailles, f.medaillesLues = tf.morts, tf.err, tf.medailles, tf.lu
	f.tempsFortsAuDernierNumero = tf.auDernierNumero
	if f.errMorts == nil && len(f.morts) > 0 && roster != nil {
		f.indexLu = true
		f.index, f.chunksDeReplicationSautes, f.errIndex = scanPlayerIndices(film, roster(f.morts))
	}
	f.origineUS, f.errOrigine = ScanClockOrigin(film)
	return f
}

// verser pose les faits dans les lectures du pont et verse les deux replis au rapport du contexte,
// dans l ordre de la lecture en serie : temps forts au dernier numero, puis chunks sautes (ce
// dernier seulement quand la table d index a ete lue).
func (f FaitsDuFilmSeul) verser(fc *FilmContext, l *LecturesDuPont) {
	l.Morts, l.ErrMorts, l.Medailles, l.MedaillesLues = f.morts, f.errMorts, f.medailles, f.medaillesLues
	fc.NoterReplis(ComptesDesReplis{TempsFortsAuDernierNumero: unSi(f.tempsFortsAuDernierNumero)})
	if f.indexLu {
		l.IndexLu, l.Index, l.ErrIndex = true, f.index, f.errIndex
		fc.NoterReplis(ComptesDesReplis{ChunksDeReplicationSautes: f.chunksDeReplicationSautes})
	}
	l.OrigineUS, l.ErrOrigine = f.origineUS, f.errOrigine
}
