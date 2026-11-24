package replay

// lecture_monde.go — LES LECTURES DU MONDE, SEPAREES DE LEUR PUBLICATION.
//
// Les balayages des poses, des evenements de piece engendree, des socles et des vehicules faisaient,
// chacun, la lecture du film PUIS le journal et le versement de leurs replis dans la meme fonction.
// Ils se coupent ici en deux : une fonction `decodeFilm<Calque>Lecture`, qui ne fait que lire et rend
// une valeur (resultat, erreur, et ce qu il faudra verser ou journaliser), et une fonction
// `journaliser<Calque>`, qui prend cette valeur et ecrit le journal et verse les replis, dans l ordre
// ou la fonction d origine le faisait. Les fonctions `decodeFilm<Calque>` d origine en sont la
// composition, pour leurs autres appelants (le sync, les instruments).
//
// CE QUE LA LECTURE TOUCHE ENCORE DU CONTEXTE DU FILM : ses memos (bande bipede, registre,
// profil, pistes et creations du monde) et, pour les socles et les vehicules, les largeurs du bloc
// MPP, posees sur le contexte (`PoserMPP`) le temps de la lecture puis restituees. Elle ne verse
// aucun repli, n ecrit ni journal ni compteur d expvar : les decisions de repli voyagent dans la valeur
// rendue ([versementMPP]).

import (
	"context"
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// versementMPP porte les replis que le choix des largeurs MPP a decides : le compteur d un format de
// version inconnue (avec la version lue), le repli sur les largeurs calibrees sur le film, le repli sur
// les largeurs par defaut de l invariant. [versementMPP.verser] les verse, dans cet ordre.
type versementMPP struct {
	formatInconnu bool
	version       int
	calibrees     bool
	parDefaut     bool
}

// verser compte les replis decides sur le contexte et dans les compteurs d exploitation.
func (v versementMPP) verser(fc *grammar.FilmContext) {
	if v.formatInconnu {
		publierFormatSansProfil(v.version)
	}
	if v.calibrees {
		fc.NoterReplis(grammar.ComptesDesReplis{LargeursMPPCalibrees: 1})
	}
	if v.parDefaut {
		fc.NoterReplis(grammar.ComptesDesReplis{LargeursMPPParDefaut: 1})
	}
}

// ---------------------------------------------------------------------------
// POSES D EQUIPEMENT ET EVENEMENTS 103
// ---------------------------------------------------------------------------

// lecturePoses est ce que le balayage des poses a lu : les poses, leurs statistiques et l erreur.
type lecturePoses struct {
	poses []types.EquipmentPlacement
	stats grammar.EquipmentPlacementStats
	err   error
}

// decodeFilmPlacementsLecture lit les poses du film. Aucun journal, aucun compteur.
func decodeFilmPlacementsLecture(fc *grammar.FilmContext, worldRange *profile.Vec3Range) lecturePoses {
	pl, st, err := grammar.ScanEquipmentPlacements(fc, worldRange)
	return lecturePoses{poses: pl, stats: st, err: err}
}

// journaliserPoses verse le compteur du format sans profil, ecrit le journal des poses et rend ce que
// le balayage publie (rien d autre que les statistiques sur une lecture en erreur).
func journaliserPoses(ctx context.Context, matchID string, lu lecturePoses,
) ([]types.EquipmentPlacement, grammar.EquipmentPlacementStats) {
	st := lu.stats
	if st.FormatSansProfil {
		// SITE 1 DU REPLI `repli_largeurs_mpp_calibrees_sur_le_film` : le compteur, pas le
		// journal — l avertissement est emis UNE FOIS PAR FILM par `avertirFormatSansProfil`.
		publierFormatSansProfil(st.FormatVersion)
	}
	switch {
	case lu.err != nil:
		slog.WarnContext(ctx, "poses d'equipement illisibles — rejeu sans equipement pose",
			"err", lu.err, "match_id", matchID)
		return nil, st
	case !st.Calibration.Widths.Valid():
		slog.WarnContext(ctx, "poses d'equipement : le decoupage du bloc de replication n'a pas ete tranche"+
			" sur ce film — AUCUNE pose publiee plutot que du bruit",
			"match_id", matchID, "ancres", st.Calibration.Anchors,
			"vies", st.Calibration.Lives, "chunksLus", st.Calibration.Chunks)
	default:
		slog.InfoContext(ctx, "poses d'equipement : records de creation ti=37",
			"decoupage", st.Calibration.Widths.String(), "accords", st.Calibration.Agree,
			"ancres", st.Anchors, "acceptes", st.Accepted,
			"confirmes", st.Confirmed, "poses", st.Placements)
	}
	return lu.poses, st
}

// lectureEvenementsPiece est ce que le balayage des evenements 103 a lu.
type lectureEvenementsPiece struct {
	evenements []types.EquipmentSpawnEvent
	stats      types.EquipmentSpawnStats
	err        error
}

// decodeFilmSpawnEventsLecture lit les evenements 103 « une PIECE a ete engendree ». Aucun journal.
func decodeFilmSpawnEventsLecture(fc *grammar.FilmContext) lectureEvenementsPiece {
	ev, st, err := grammar.ScanEquipmentSpawnEvents(fc)
	return lectureEvenementsPiece{evenements: ev, stats: st, err: err}
}

// journaliserEvenementsPiece ecrit le journal des evenements 103 et rend ce que le balayage publie.
//
// UN FILM SANS AUCUN EVENEMENT N'EST PAS UNE ERREUR, ET CE N'EST PAS NON PLUS UN SILENCE : les
// builds les plus anciens du corpus n'en rendent aucun, la ou les recents en rendent des dizaines.
// Le journal publie donc les DEUX denominateurs (listes traversees et evenements lus), et la
// couverture de l'artefact les publie tous deux (`coverage.placements.spawnLists` et `.spawnEvents`).
func journaliserEvenementsPiece(ctx context.Context, matchID string, lu lectureEvenementsPiece,
) ([]types.EquipmentSpawnEvent, types.EquipmentSpawnStats) {
	st := lu.stats
	if lu.err != nil {
		slog.WarnContext(ctx, "evenements de piece engendree illisibles — l'origine des poses retombe sur ses replis",
			"err", lu.err, "match_id", matchID)
		return nil, st
	}
	slog.InfoContext(ctx, "poses d'equipement : evenements 103 (piece engendree)",
		"match_id", matchID, "chunks", st.Chunks, "paquetsDelta", st.Packets,
		"listesNonVides", st.Lists, "evenements", st.Events,
		"refSource", st.WithSource, "refEngendree", st.WithSpawned, "ref2", st.Ref2)
	return lu.evenements, st
}

// ---------------------------------------------------------------------------
// SOCLES (armes au sol et power-ups)
// ---------------------------------------------------------------------------

// issueDeLecture dit comment la lecture d un calque d objets du monde s est arretee.
type issueDeLecture int

const (
	lectureComplete issueDeLecture = iota
	lectureBandeVide
	lectureCreationsIllisibles
	lecturePistesIllisibles
	lectureNuageIllisible
)

// lectureSocle est ce que la lecture d UN archetype de socle a rendu.
type lectureSocle struct {
	arch   padArchetype
	mpp    versementMPP
	kf     grammar.WorldObjectKeyframes
	issue  issueDeLecture
	cre    []types.EquipmentCreation
	st     types.EquipmentCreationStats
	tracks []types.ProjectileTrack
	err    error
}

// lecturePads est la lecture des deux voies de la chaine des socles.
type lecturePads struct {
	armes, powerups lectureSocle
}

// decodeFilmPadScansLecture lit LES DEUX voies des socles sur le meme film et aux memes largeurs MPP,
// armes puis power-ups, comme `decodeFilmPadScans`. Aucun journal, aucun repli verse.
func decodeFilmPadScansLecture(fc *grammar.FilmContext, wr *profile.Vec3Range, mpp profile.MPPWidths) lecturePads {
	return lecturePads{
		armes:    lireUnSocle(fc, wr, mpp, groundWeaponArchetype()),
		powerups: lireUnSocle(fc, wr, mpp, worldEquipmentArchetype()),
	}
}

// lireUnSocle lit UN archetype : bande des images-cles, creations, pistes delta, aux largeurs MPP du
// film (installees sur le contexte le temps de la lecture).
func lireUnSocle(fc *grammar.FilmContext, wr *profile.Vec3Range, mpp profile.MPPWidths, arch padArchetype,
) lectureSocle {
	largeurs, verse := gwReglageMPP(fc, mpp)
	restaurer := gwPoserLargeurs(fc, largeurs)
	defer restaurer()
	lu := lectureSocle{arch: arch, mpp: verse}
	lu.kf = grammar.ScanWorldObjectKeyframes(fc, arch.ti)
	if len(lu.kf.Band) == 0 {
		lu.issue = lectureBandeVide
		return lu
	}
	lu.cre, lu.st, lu.err = arch.scan(fc, wr, lu.kf.Band)
	if lu.err != nil {
		lu.issue = lectureCreationsIllisibles
		return lu
	}
	lu.tracks, lu.err = grammar.ScanWorldObjectsForBand(fc, wr, arch.ti, lu.kf.Band)
	if lu.err != nil {
		lu.issue = lecturePistesIllisibles
	}
	return lu
}

// journaliserSocle verse les replis des largeurs MPP, ecrit le journal de la lecture et rend le calque
// que le balayage publie : vide quand la lecture s est arretee.
func journaliserSocle(ctx context.Context, fc *grammar.FilmContext, matchID string, lu lectureSocle) WorldObjectScan {
	lu.mpp.verser(fc)
	switch lu.issue {
	case lectureBandeVide:
		slog.WarnContext(ctx, "socles : aucun slot de l archetype aux images-cles — rejeu sans ce calque",
			"archetype", lu.arch.label, "match_id", matchID, "imagesCles", len(lu.kf.TimesUS))
		return WorldObjectScan{}
	case lectureCreationsIllisibles:
		slog.WarnContext(ctx, "socles : records de creation illisibles — rejeu sans ce calque",
			"archetype", lu.arch.label, "err", lu.err, "match_id", matchID)
		return WorldObjectScan{}
	case lecturePistesIllisibles:
		slog.WarnContext(ctx, "socles : pistes delta illisibles — AUCUN socle publie (sans elles, toute"+
			" apparition passerait pour un objet apparu au repos)",
			"archetype", lu.arch.label, "err", lu.err, "match_id", matchID)
		return WorldObjectScan{}
	}
	slog.InfoContext(ctx, "socles : balayage d archetype",
		"archetype", lu.arch.label, "slots", lu.st.Slots, "ancres", lu.st.Anchors, "acceptees", lu.st.Accepted,
		"imagesCles", len(lu.kf.TimesUS), "viesRecensees", len(lu.kf.SeenUS), "pistesDelta", len(lu.tracks))
	return WorldObjectScan{Scanned: true, Creations: lu.cre, Stats: lu.st, Keyframes: lu.kf, Tracks: lu.tracks}
}

// journaliserPads publie les deux voies des socles, armes puis power-ups.
func journaliserPads(ctx context.Context, fc *grammar.FilmContext, matchID string, lu lecturePads) PadScans {
	return PadScans{
		Weapons:  journaliserSocle(ctx, fc, matchID, lu.armes),
		Powerups: journaliserSocle(ctx, fc, matchID, lu.powerups),
	}
}

// ---------------------------------------------------------------------------
// VEHICULES (ti=40)
// ---------------------------------------------------------------------------

// lectureVehicules est ce que le balayage des vehicules a rendu : les trois lectures fondatrices
// (images-cles, creations, nuage), les evenements quand elles ont abouti, et les visees sans position,
// lues sur tout film.
type lectureVehicules struct {
	mpp        versementMPP
	kf         grammar.WorldObjectKeyframes
	issue      issueDeLecture
	cre        []types.EquipmentCreation
	st         types.EquipmentCreationStats
	pos        []grammar.BipedPosition
	err        error
	evenements []types.VehicleEvent
	errEvents  error
	visees     []grammar.BipedAim
	errVisees  error
}

// decodeFilmVehicleScanLecture lit les lectures du calque des vehicules aux largeurs MPP du film, puis
// les visees sans position. Aucun journal, aucun repli verse.
//
// LES EVENEMENTS D EMBARQUEMENT ET DE SORTIE, ET LES VISEES SANS POSITION, SONT ADDITIFS ET NON FATAUX :
// l absence des evenements rend les episodes d occupation au seul trou de position (86,3 % des trous
// portent leur sortie, et 100 % de ces sorties ferment le trou a +/-2 s) ; celle des visees rend des
// episodes sans serie de visee (le cone retombe sur le cap du chassis) et des tirs sans la visee d un
// record sans position. Les visees sont une lecture a part du nuage de positions : les deux balayages
// n acceptent pas les memes records (le nuage exige un `i0` absolu et un masque commencant par 0 ; les
// visees acceptent precisement ceux que cette exigence ecarte).
//
// LES VISEES SE LISENT SUR TOUT FILM, ET LES TROIS ARRETS DU CALQUE NE LES TAISENT PAS : un biped a pied
// replique aussi sa visee dans des records sans position, vehicule ou non. Elles servent la direction
// des tirs (`shots_visee.go`) et le plein debit des vies (`tracks_visees_seules.go`), pas seulement
// les occupants. La lecture suit la lecture du calque, comme avant : seul son declenchement change.
func decodeFilmVehicleScanLecture(fc *grammar.FilmContext, wr *profile.Vec3Range, mpp profile.MPPWidths,
) lectureVehicules {
	largeurs, verse := gwReglageMPP(fc, mpp)
	restaurer := gwPoserLargeurs(fc, largeurs)
	defer restaurer()
	lu := lireLeCalqueDesVehicules(fc, wr, lectureVehicules{mpp: verse})
	lu.visees, lu.errVisees = grammar.ScanBipedAimOnly(fc)
	return lu
}

// lireLeCalqueDesVehicules lit les lectures propres aux vehicules (bande, creations, nuage, evenements) ;
// elle s arrete au premier echec fatal et le dit dans [lectureVehicules.issue].
func lireLeCalqueDesVehicules(fc *grammar.FilmContext, wr *profile.Vec3Range, lu lectureVehicules,
) lectureVehicules {
	lu.kf = grammar.ScanWorldObjectKeyframes(fc, grammar.VehicleTypeIndex)
	if len(lu.kf.Band) == 0 {
		lu.issue = lectureBandeVide
		return lu
	}
	lu.cre, lu.st, lu.err = grammar.ScanVehicleCreationsForBand(fc, wr, lu.kf.Band)
	if lu.err != nil {
		lu.issue = lectureCreationsIllisibles
		return lu
	}
	lu.pos, lu.err = nuageDesVehicules(fc, lu.kf.Band, wr)
	if lu.err != nil {
		lu.issue = lectureNuageIllisible
		return lu
	}
	lu.evenements, lu.errEvents = grammar.ScanVehicleEvents(fc)
	return lu
}

// journaliserVehicules verse les replis des largeurs MPP, ecrit le journal de la lecture et rend le
// calque que le balayage publie. Les visees sans position accompagnent le calque quelle que soit son issue
// (bande vide, creations ou nuage illisibles) : elles ne dependent d aucune de ces lectures.
func journaliserVehicules(ctx context.Context, fc *grammar.FilmContext, matchID string, lu lectureVehicules,
) VehicleScan {
	lu.mpp.verser(fc)
	visees := viseesSansPositionLues(ctx, matchID, lu)
	var out VehicleScan
	switch lu.issue {
	case lectureBandeVide:
		slog.InfoContext(ctx, "vehicules : aucun slot ti=40 aux images-cles — film lu, aucun vehicule",
			"match_id", matchID, "imagesCles", len(lu.kf.TimesUS))
		out = scanBandeVide(lu.kf)
	case lectureCreationsIllisibles:
		slog.WarnContext(ctx, "vehicules : records de creation illisibles — rejeu sans ce calque",
			"err", lu.err, "match_id", matchID)
	case lectureNuageIllisible:
		slog.WarnContext(ctx, "vehicules : nuage de positions illisible — AUCUN vehicule publie (sans lui, une"+
			" vie recensee n aurait ni trajectoire ni cap)", "err", lu.err, "match_id", matchID)
	default:
		out = calqueDesVehicules(ctx, matchID, lu)
	}
	out.Aims = visees
	return out
}

// viseesSansPositionLues rend les visees sans position de la lecture, nil quand elle a echoue : l echec est
// journalise et n arrete rien (les episodes d occupation sortent sans serie de visee, le cone retombe sur
// le cap du chassis, et les tirs gardent la visee de leur record).
func viseesSansPositionLues(ctx context.Context, matchID string, lu lectureVehicules) []grammar.BipedAim {
	if lu.errVisees != nil {
		slog.WarnContext(ctx, "visees sans position illisibles — episodes d occupation sans serie de visee, tirs"+
			" sans les visees des records sans position", "err", lu.errVisees, "match_id", matchID)
		return nil
	}
	slog.InfoContext(ctx, "visees sans position lues", "match_id", matchID, "viseesSansPosition", len(lu.visees))
	return lu.visees
}

// calqueDesVehicules rend le calque d une lecture aboutie : images-cles, creations, nuage et evenements.
func calqueDesVehicules(ctx context.Context, matchID string, lu lectureVehicules) VehicleScan {
	out := VehicleScan{Scanned: true, Keyframes: lu.kf, Creations: lu.cre, Stats: lu.st, Positions: lu.pos}
	// ADDITIFS ET NON FATAUX : leur absence rend les episodes d occupation au seul trou de position.
	if lu.errEvents != nil {
		slog.WarnContext(ctx, "vehicules : liste d evenements illisible — episodes d occupation bornes par le"+
			" seul trou de position", "err", lu.errEvents, "match_id", matchID)
	} else {
		out.Events = lu.evenements
	}
	slog.InfoContext(ctx, "vehicules : balayage ti=40",
		"slots", lu.st.Slots, "ancres", lu.st.Anchors, "creationsAcceptees", lu.st.Accepted,
		"imagesCles", len(lu.kf.TimesUS), "viesRecensees", len(lu.kf.SeenUS),
		"echantillons", len(lu.pos), "evenements", len(out.Events))
	return out
}
