package replay

// document_repliques.go — LES REPLIQUES DES SPARTANS, lues dans les messages `AIDialog` du film.
//
// CE QUE C EST. Le film ecrit une replique prononcee par une unite dans la liste d evenements de la trame (genre 75 de la
// vue A) : une priorite, l identifiant du son, un indice de replique, un index de joueur, et, dans l en-tete, l unite qui parle
// (reference 0). Ce calque pose chaque replique sur l axe de frames avec le joueur qui parle.
//
// QUI PARLE : L UNITE DE LA REFERENCE 0, ET ELLE SEULE. Elle se resout par sa piste a la frame
// ([resolveurDeTouches.victime], la voie de l unite blessee d un message de degat) : le joueur dont la piste porte le slot de
// l unite, a l instant du message. L INDEX DE JOUEUR DE LA CHARGE N EST PAS LE LOCUTEUR, ET LE FILM LE MONTRE (mesure sur un
// film de huit joueurs, 365 repliques, 199 avec index) : l unite se resout en un joueur pour les 365, a travers les huit
// joueurs, et un meme son ne vient que des joueurs d un meme groupe de voix (un son pour trois joueurs, un autre pour trois
// autres, deux sons pour un seul joueur chacun) ; l index, lui, ne prend que les valeurs 0 a 5 (190 sur 199 entre 0 et 2) et
// designe le meme joueur que l unite dans 35 cas sur 199, soit le hasard. Le document ne le publie donc pas : son sens n est pas
// etabli, et il reste dans les faits du film (`FilmInputs.Repliques`).
//
// Une replique dont l unite n a pas de piste a cette frame, ou celle d un bot, est publiee SANS joueur et comptee
// (`coverage.chatter.withoutPlayer` ; `bots` pour la piste d un bot).
//
// CE QUE LE FILM NE DIT PAS. Ni la variante audio qui a joue, ni le nom de la replique : le film n ecrit que l identifiant
// du son et l indice de replique. LE NOM COURT VIENT DE L INDICE (`name`, par [chattername.Lookup]), jamais du son : l indice
// numerote une entree des tables de vocalisation, commune a toutes les voix ; le son change avec la voix. Le nom n est publie que
// pour un build dont la disposition de ces tables est mesuree, et pour une entree dont le nom est prouve par son hachage ; toute
// autre replique est publiee SANS nom et comptee (`coverage.chatter.unnamed`).

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/chattername"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Chatter est UNE replique de Spartan, sur l axe de frames du document.
type Chatter struct {
	// T est l index de frame de la trame qui ecrit le message, sur le meme axe que Point.T.
	T int `json:"t"`
	// Player est le joueur qui parle, en decimal — meme forme que Track.XUID. Absent quand l unite qui parle n a pas de piste
	// d humain a cette frame (un bot, ou un locuteur sans piste).
	Player string `json:"player,omitempty"`
	// Sound est l identifiant du son, en hexadecimal sur huit chiffres. Absent quand le message n en porte pas.
	Sound string `json:"sound,omitempty"`
	// Priority est la classe de priorite de la replique (cinq bits).
	Priority int `json:"priority"`
	// Line est l indice de replique (douze bits). Absent quand le message n en porte pas.
	Line *int `json:"line,omitempty"`
	// Name est le nom court de la replique, en minuscules sans separateur (`seeingenemynearby`). Absent quand le build du film
	// n a pas de disposition mesuree, ou que l indice n a pas de nom prouve : voir [chattername].
	Name string `json:"name,omitempty"`
}

// ChatterCoverage dit ce que le calque a lu, publie et ecarte.
//
// L IDENTITES : read = beforeOrigin + afterEnd + published ; published = bySpeaker + withoutPlayer ; bots <= withoutPlayer ;
// published = named + unnamed.
type ChatterCoverage struct {
	// Frames, ClosedFrames, StoppedFrames : les trames delta du film et ce que la lecture de leur liste d evenements en a
	// fait, lue jusqu au terminateur ou arretee par un message non lu (ceux qui suivent l arret sont perdus : le compte est
	// alors un plancher).
	Frames        int `json:"frames"`
	ClosedFrames  int `json:"closedFrames"`
	StoppedFrames int `json:"stoppedFrames"`
	// Read : les repliques lues. BeforeOrigin et AfterEnd : celles hors de l axe de frames du match — ecartees.
	Read         int `json:"read"`
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	// Published : les repliques publiees. BySpeaker : le joueur vient de la piste de l unite qui parle. WithoutPlayer : la
	// replique n en a pas (pas de reference d unite, pas de piste a cette frame, ou piste d un bot) ; Bots : parmi elles, celles
	// dont l unite est la piste d un bot.
	Published     int `json:"published"`
	BySpeaker     int `json:"bySpeaker"`
	WithoutPlayer int `json:"withoutPlayer"`
	Bots          int `json:"bots"`
	// Named : les repliques publiees avec un nom court. Unnamed : celles sans nom (build sans disposition mesuree, film sans build
	// lu, indice absent ou sans nom prouve).
	Named   int `json:"named"`
	Unnamed int `json:"unnamed"`
}

// hexaSurHuitChiffres rend une valeur de 32 bits en hexadecimal sur huit chiffres (un identifiant de son, un masque).
func hexaSurHuitChiffres(son uint32) string { return fmt.Sprintf("%08x", son) }

// entierOptionnel rend un pointeur sur `v`, pour un champ absent quand le message ne le porte pas.
func entierOptionnel(v int, present bool) *int {
	if !present {
		return nil
	}
	return &v
}

// joueurDeLaReplique resout qui parle a la frame `t` : le xuid du joueur dont la piste porte l unite de la reference 0, et si
// cette piste est celle d un bot.
func joueurDeLaReplique(m grammar.RepliqueLue, r resolveurDeTouches, t int) (xuid string, bot bool) {
	if !m.ALocuteur {
		return "", false
	}
	xuid, piste := r.victime(m.Locuteur, t)
	return xuid, piste && xuid == ""
}

// buildChatter pose les repliques lues sur l axe de frames, avec le joueur qui parle et le nom court de la replique pour le build
// `build` du film (vide : film sans build lu, aucun nom). Les lignes sont triees par frame, dans
// l ordre du film a frame egale.
func buildChatter(in grammar.RepliquesLues, tracks []Track, axe axeDeFrames, build string) ([]Chatter, ChatterCoverage) {
	st := in.Stats
	cov := ChatterCoverage{Frames: st.Trames, ClosedFrames: st.Terminees, StoppedFrames: st.Arretees, Read: len(in.Repliques)}
	r := resolveurDeTouches{tracks: tracks, axe: axe}
	out := make([]Chatter, 0, len(in.Repliques))
	for _, m := range in.Repliques {
		t, refus := frameDeLAction(m.TS, axe)
		switch refus {
		case refusAvantOrigine, refusSansOrigine:
			cov.BeforeOrigin++
			continue
		case refusApresFin:
			cov.AfterEnd++
			continue
		}
		xuid, bot := joueurDeLaReplique(m, r, t)
		switch {
		case xuid != "":
			cov.BySpeaker++
		case bot:
			cov.WithoutPlayer++
			cov.Bots++
		default:
			cov.WithoutPlayer++
		}
		l := Chatter{T: t, Player: xuid, Priority: int(m.Priorite), Line: entierOptionnel(int(m.Ligne), m.ALigne)}
		if m.ASon {
			l.Sound = hexaSurHuitChiffres(m.Son)
		}
		if m.ALigne {
			l.Name, _ = chattername.Lookup(build, int(m.Ligne))
		}
		if l.Name != "" {
			cov.Named++
		} else {
			cov.Unnamed++
		}
		out = append(out, l)
	}
	cov.Published = len(out)
	if len(out) == 0 {
		out = nil // un document sans replique publiee garde la valeur zero, comme avant la reservation
	}
	slices.SortStableFunc(out, func(a, b Chatter) int { return cmp.Compare(a.T, b.T) })
	return out, cov
}

// poserLesRepliques publie les calques des repliques et des annonces : seulement quand la marche des trames a lu au moins une
// trame. Les deux se posent apres le roster et les pistes, qu ils lisent.
func (a *assemblage) poserLesRepliques() {
	in := a.opt.Repliques
	if in.Stats.Trames == 0 {
		return
	}
	axe := axeDeFrames{a.origin, a.step, a.doc.FrameCount}
	build := ""
	if a.opt.FilmIdentity != nil {
		build = a.opt.FilmIdentity.Build
	}
	chatter, cov := buildChatter(in, a.doc.Tracks, axe, build)
	a.doc.Chatter = chatter
	a.doc.Coverage.Chatter = &cov
	slog.InfoContext(a.ctx, "rejeu : repliques", "lues", cov.Read, "avantOrigine", cov.BeforeOrigin, "apresFin", cov.AfterEnd,
		"publiees", cov.Published, "parLocuteur", cov.BySpeaker, "sansJoueur", cov.WithoutPlayer,
		"bots", cov.Bots, "nommees", cov.Named, "sansNom", cov.Unnamed, "tramesArretees", cov.StoppedFrames, "trames", cov.Frames)
	a.poserLesAnnonces(in, axe)
}
