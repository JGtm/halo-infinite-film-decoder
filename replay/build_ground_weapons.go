package replay

// build_ground_weapons.go — LE CABLAGE du calque des SOCLES : ce que BuildFromFilm DECODE du
// film, et ce que BuildFromPositions en ASSEMBLE.
//
// Extrait de `build.go` et de `ground_weapon_objects.go` au correctif de revue du 2026-08-17 :
// le lot des socles avait pousse `build.go` de 621 a 640 lignes, au-dessus d un seuil deja gele
// par la baseline. Le decodage, le journal et les refus du calque vivent donc ici, ensemble ;
// `build.go` ne porte plus qu un appel par cote.
//
// TROIS LECTURES DU FILM PAR ARCHETYPE, ET PAS UNE DE PLUS : une marche des images-cles (bande
// de slots + recensement), une marche des paquets delta pour les records de CREATION, une pour
// les pistes de position. La marche des images-cles en rend deux pour le prix d une.
//
// DEUX ARCHETYPES, UN SEUL DECODEUR (2026-08-19). Les ARMES AU SOL (`ti=42`) et les objets
// d EQUIPEMENT du monde (`ti=37`, d ou sortent les socles de POWER-UP) se lisent par la MEME
// sequence : seuls changent le typeIndex et le deserialiseur du default-state, que
// `padArchetype` porte. Deux copies de cette sequence auraient diverge au premier correctif —
// et celui du 2026-08-17 (largeurs MPP a reinstaller) montre que ce correctif arrive.
//
// HORS LIGNE : `decodeFilmPadScans` fait de l I/O disque sur tout le film et n est appelee que
// par `BuildFromFilm`. `attachWeaponPads` est PUR.

import (
	"context"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// padArchetype dit CE QUI CHANGE d'un archétype d'objet du monde à l'autre : son typeIndex, le
// mot qui le nomme au journal, et le balayage de créations qui sait dérouler SON default-state.
//
// LE NOM EST AU JOURNAL, ET IL LE FAUT : « ti=42 » n'est pas un message d'exploitation. Un
// build qui se tait sur les socles doit dire LEQUEL des deux calques s'est tu.
type padArchetype struct {
	ti    int
	label string
	scan  func(fc *grammar.FilmContext, wr *profile.Vec3Range, band map[uint32]bool) (
		[]types.EquipmentCreation, types.EquipmentCreationStats, error)
}

// groundWeaponArchetype / worldEquipmentArchetype : les deux voies de la chaîne des socles.
func groundWeaponArchetype() padArchetype {
	return padArchetype{
		ti: grammar.GroundWeaponTypeIndex, label: "armes au sol (ti=42)",
		scan: grammar.ScanGroundWeaponCreationsForBand,
	}
}

func worldEquipmentArchetype() padArchetype {
	return padArchetype{
		ti: grammar.EquipmentTypeIndex, label: "power-ups de socle (ti=37)",
		scan: grammar.ScanEquipmentCreationsForBand,
	}
}

// decodeFilmPadScans décode LES DEUX voies de la chaîne des socles sur le même film et aux mêmes
// largeurs de bloc MPP.
//
// LES DEUX SORTENT ENSEMBLE parce qu'elles entrent ensemble dans l'assemblage : les socles des
// deux natures se publient dans le MÊME `weaponPads`, et une voie décodée sans l'autre
// laisserait l'artefact affirmer « aucun socle de power-up » là où il faudrait dire « pas lu ».
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPadScans(ctx context.Context,
	fc *grammar.FilmContext, matchID string, wr *profile.Vec3Range, mpp profile.MPPWidths,
) PadScans {
	return PadScans{
		Weapons:  decodeFilmPadScan(ctx, fc, matchID, wr, mpp, groundWeaponArchetype()),
		Powerups: decodeFilmPadScan(ctx, fc, matchID, wr, mpp, worldEquipmentArchetype()),
	}
}

// decodeFilmPadScan décode UN archétype d'objet du monde et JOURNALISE ce qu'il en est.
//
// TROIS PANNES, TROIS PHRASES, et la distinction est le point. Un film sans l'archétype aux
// images-clés (aucune bande) n'est pas un film dont les créations sont illisibles, et ni l'un ni
// l'autre n'est un film dont les pistes delta manquent — ce dernier cas est le plus TRAÎTRE,
// parce qu'une piste absente rend `HasDelta` faux et ferait passer TOUTE apparition `spawned`
// pour un objet apparu au repos. Le calque se tait donc entièrement plutôt que de publier des
// socles fabriqués par une lecture manquante.
//
// LES LARGEURS DU BLOC MPP SONT CELLES DE CE FILM, ET C'EST UN CORRECTIF DE REVUE (2026-08-17).
// Le mot d'identité de 32 bits se lit derrière deux champs de largeur VARIABLE, mesurés par la
// calibration des poses `ti=37` sur le MÊME film (9/5 en Quick Play, 8/3 sur les films BTB
// mesurés) — et `ScanFilmEquipmentPlacements` les RESTAURE en sortant. Sans les réinstaller ici,
// le balayage lisait l'identité aux largeurs PAR DÉFAUT : sur un film calibré autrement, aucune
// création n'aurait résolu de famille, le calque aurait publié zéro socle, et rien ne l'aurait
// dit (découverte 8 du plan des armes au sol). Largeurs non mesurées (calibration refusée) : on
// garde le défaut, et le compteur `kept` de la couverture reste le témoin.
//
// HORS LIGNE — appelée par BuildFromFilm.
func decodeFilmPadScan(ctx context.Context,
	fc *grammar.FilmContext, matchID string, wr *profile.Vec3Range, mpp profile.MPPWidths,
	arch padArchetype,
) WorldObjectScan {
	return journaliserSocle(ctx, fc, matchID, lireUnSocle(fc, wr, mpp, arch))
}

// gwWidthsForFilm rend les largeurs MPP a INSTALLER pour ce film : celles que la grammaire porte
// ([grammar.FilmContext.ResolutionMPP] — la version de format, ou la taille d etat de creation que
// le film declare), sinon les largeurs CALIBREES sur le film.
//
// Quand la grammaire decide, la calibration n est plus qu un CONTROLE (le rapport publie ce
// qu elle mesure) ; sinon elle decide encore, et c est le repli
// `repli_largeurs_mpp_calibrees_sur_le_film` du registre (ordre `apres_lecture`). Les poses
// d equipement passent par la MEME porte : dans une cuisson, socles, vehicules et poses se
// decoupent aux memes largeurs.
func gwWidthsForFilm(fc *grammar.FilmContext, calibrees profile.MPPWidths) profile.MPPWidths {
	w, verse := gwReglageMPP(fc, calibrees)
	verse.parDefaut = false
	verse.verser(fc)
	return w
}

// gwReglageMPP decide les largeurs MPP a installer pour ce film SANS rien verser : les largeurs, et
// les replis que la decision a pris ([versementMPP.verser] les compte). C est la decision de
// [gwWidthsForFilm] ; le repli sur les largeurs par defaut (calibration refusee) y est porte aussi,
// pour que la lecture qui installe les largeurs ne verse rien.
func gwReglageMPP(fc *grammar.FilmContext, calibrees profile.MPPWidths) (profile.MPPWidths, versementMPP) {
	res := fc.ResolutionMPP()
	if res.Decide() {
		return res.Widths, versementMPP{parDefaut: !res.Widths.Valid()}
	}
	// SITE 2 DU REPLI `repli_largeurs_mpp_calibrees_sur_le_film`, et il sert DEUX chemins de
	// cuisson (armes au sol et vehicules). Le compteur seul : l avertissement est emis une fois
	// par film par `avertirFormatSansProfil`.
	return calibrees, versementMPP{formatInconnu: res.FormatInconnu, version: res.FormatVersion,
		calibrees: calibrees.Valid(), parDefaut: !calibrees.Valid()}
}

// gwInstallMPPWidths installe les largeurs du bloc MPP MESURÉES sur ce film SUR LE CONTEXTE, et
// rend leur restauration. Largeurs non renseignées (calibration refusée) : rien n'est installé —
// l'invariant du profil vaut mieux qu'un découpage nul, qui ne lirait aucune identité du tout.
func gwInstallMPPWidths(fc *grammar.FilmContext, w profile.MPPWidths) func() {
	if !w.Valid() {
		// Ni relue ni calibree : le balayage se fait aux largeurs de l INVARIANT (9/5), repli
		// `repli_largeurs_mpp_par_defaut` (lot J8.7), au rapport du contexte.
		fc.NoterReplis(grammar.ComptesDesReplis{LargeursMPPParDefaut: 1})
	}
	return gwPoserLargeurs(fc, w)
}

// gwPoserLargeurs pose les largeurs du bloc MPP sur le contexte et rend leur restauration ; largeurs
// non renseignees : rien n est pose, la restauration ne fait rien. Aucun repli n est verse.
func gwPoserLargeurs(fc *grammar.FilmContext, w profile.MPPWidths) func() {
	if !w.Valid() {
		return func() {}
	}
	prev := fc.PoserMPP(w)
	return func() { fc.PoserMPP(prev) }
}

// attachWeaponPads pose le calque des SOCLES sur le document : les socles des DEUX natures,
// leurs occupations achevees, leur couverture, et le journal qui porte les memes denominateurs
// que l artefact.
//
// LE NUAGE EST CELUI QUI N EST PAS DECIME, et il le faut : la datation d une disparition se joue
// a la frame, la decimation perdrait le passage de joueur qui la borne. Le calque publie les
// socles du MATCH — aucun catalogue de carte, aucun ramasseur.
//
// `positions` doit etre TRIE par instant (c est le cas de `sorted` dans BuildFromPositions).
// LES DEUX TABLES D IDENTITE viennent du manifeste du titre : les objets d objectif (que la voie
// des ARMES ecarte) et les familles d equipement (dont la voie des POWER-UPS tire sa
// selectivite). Tables vides = comportement d avant le 2026-08-18 / le 2026-08-19.
// Le retour est la liste des OBJETS INDIVIDUELS de la voie des armes : le calque des armes au
// sol (schéma 27) les consomme après coup — même chaîne, deux publications.
func attachWeaponPads(ctx context.Context,
	doc *ReplayDocument, scans PadScans, positions []grammar.BipedPosition, clock replayClock,
	cat LabelCatalog,
) []gwPickupObject {
	// L ADAPTATION DU CATALOGUE SE FAIT ICI, PAS DANS L ASSEMBLAGE : `buildWeaponPads` est PUR
	// et ne prend que les deux tables qu il consomme, jamais le catalogue entier du titre.
	c := buildPadsAndCandidates(scans, positions, clock, padCatalogs{
		ObjectiveObjects: cat.ObjectiveObjects, EquipmentFamilies: cat.EquipmentFamilies,
		DeployedEquipment: cat.DeployedEquipment})
	doc.WeaponPads, doc.PadPickups, doc.Coverage.GroundWeapons = c.Pads, c.Picks, c.Cov
	doc.PadCandidates = c.Candidates
	logGroundWeaponCoverage(ctx, doc.Coverage.GroundWeapons)
	return c.Objs
}
