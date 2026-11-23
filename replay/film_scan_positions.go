package replay

// film_scan_positions.go — LA PHASE DES POSITIONS : version du film, pont d identite, poussees,
// positions, creations de bipede, tirs. SORTI DE `film_scan.go` pour tenir son seuil de longueur quand
// chaque balayage s y coupe en deux.
//
// Chaque balayage se coupe en deux methodes appelees l une derriere l autre a la place qu il occupait :
// `lire…` ne fait que lire le film et rend une valeur (resultat, statistiques, erreur) ; `publier…`
// prend cette valeur et fait, dans l ordre d avant, ce qui s observe — journal, versement des replis,
// entrees du balayage, etape observee. Une lecture ne verse aucun repli et n ecrit aucun journal.
//
// CE QUE CHAQUE `lire…` TOUCHE ENCORE DU CONTEXTE DU FILM :
//   - la version : le profil du contexte (memo, deja resolu) ;
//   - le pont d identite : tous les memos du contexte (registre, bande bipede, generations vivantes,
//     creations), et la tache de fond (`fond.lesFaits`, un point d attente) ;
//   - les poussees et les tirs : les memos de lectures bipedes de la marche des trames.

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// balayerPositions lit la version du film, les teleportations, les positions, les creations de
// bipede, les tirs et les armes de depart. C'EST LA SEULE PHASE QUI PEUT ECHOUER : sans
// positions, il n'y a pas de rejeu.
func (s *filmScan) balayerPositions() error {
	s.publierLaVersion(s.lireLaVersion())
	s.publierPontDIdentite(s.lirePontDIdentite())
	s.publierPoussees(s.lirePoussees())
	if s.pont.ErrPositions != nil {
		return s.pont.ErrPositions
	}
	s.publierPositions()
	s.balayerCreations()
	// Les tirs sont décodés du MÊME film et sur la MÊME horloge que les positions ; leur
	// absence n'est pas fatale (un film sans event de tir reste un rejeu valide).
	s.publierTirs(s.lireTirs())
	// Armes portées : lues dans les keyframes du MÊME film, sur la MÊME horloge. Leur
	// absence n'est pas fatale (un rejeu sans armes reste un rejeu valide).
	//
	// LA MARCHE D'IMAGE-CLÉ SE COMPTE ICI (lot M3.1) : ce balayage marche chaque payload
	// d'image-clé du film exactement une fois. L'élection de l'ancre suivante est un REPLI nommé,
	// et son compte voyage dans `coverage.fallbacks` ; la santé complète de la marche est
	// portée par les faits (`KeyframeWalk`). La marche est UNIQUE ([filmScan.lireLesImagesCles]).
	s.publierLesImagesCles(s.lireLesImagesCles())
	s.publierLesLoadouts()
	s.publierNaissances(s.lireNaissances(s.in.BipedCreations))
	return nil
}

// lireLaVersion rend le profil de version du film. ELLE VIENT DU PROFIL DU CONTEXTE, PAS D'UNE
// RELECTURE (lot 2.2.d) : le contexte a resolu le profil du film UNE fois, a sa construction (D1), et
// `Highlight` porte la version avec le drapeau qui dit si elle a ete LUE. Rouvrir le registre ici
// rendrait la MEME valeur par le MEME chemin — `HighlightProfileOfFilm` et `ResolveProfile` composent
// l'une comme l'autre depuis `FilmMajorVersionFromHeader` — pour une seconde localisation de `chunk_00`.
func (s *filmScan) lireLaVersion() profile.HighlightProfile {
	return s.fc.Profile().Highlight()
}

// publierLaVersion pose la version du film dans les entrees. LA VERSION DU FILM VOYAGE AVEC
// L'ARTEFACT : elle commande deja le decoupage du gamertag du fil des morts, et rien dans l'artefact ne
// disait sous quelle grammaire il avait ete cuit.
//
// FILM SANS REGISTRE : nil dans la couverture, ET UN WARN ICI. C'est cette phase qui le porte parce
// qu'elle est la seule du chemin a connaitre le `match_id` — `ScanDeaths`, qui applique le decoupage
// historique dans ce cas, recoit un film deja charge et est appelee deux fois par cuisson (revue
// adversariale du 2026-09-12, constat P2-4 : le WARN y etait sans match et en double).
func (s *filmScan) publierLaVersion(hl profile.HighlightProfile) {
	if hl.Lue {
		v := hl.MajorVersion
		s.in.FilmMajorVersion = &v
		return
	}
	slog.WarnContext(s.ctx, "rejeu : version de film illisible — le fil des morts retombe sur le decoupage "+
		"historique du gamertag", "match_id", s.matchID)
}

// lirePontDIdentite lit, d'un seul appel a l'etage unique, les six lectures du pont d'identite.
//
// TÉLÉPORTATIONS DU TRANSLOCATEUR : lues AVANT les positions, parce qu'elles servent
// deux fois — le calque `translocations` du document, et l'EXEMPTION du filtre de
// vitesse (décision D2) : une arrivée de téléportation part à 193-1540 m/s, le filtre à
// 100 m/s la rejetait à tort (R3 : 51/51 rejets mesurés, tous à ±200 ms d'un événement
// 117 du même slot). Sur un film sans tête 117, la liste est vide et le filtre est
// bit à bit identique à l'actuel — invariance prouvée par test.
//
// L'ENTRÉE DE CATALOGUE Y DESCEND parce que la CHARGE de l'événement porte les deux
// positions du va-et-vient, quantifiées aux bornes de la carte (R6 §1, validé 18/18) :
// sans elle le scanner rendrait des quanta invérifiables, donc rien. Elle est garantie
// non nulle ici (refus en tête de BuildFromFilm).
//
// L'ETAGE UNIQUE DU PONT D'IDENTITE (lot J4.3, 2026-09-26) : teleportations, positions avec
// leurs exemptions, creations de bipede, fil des morts, table d'index et origine d'horloge sont
// lus ICI, d'un seul appel — le MEME que le collecteur killsource. Seules les POLITIQUES de la
// cuisson restent ici : le roster de l'index (fil des morts + roster de l'appelant), la capture des
// directions (dans `s.scan`), l'injectivite et la fatalite des erreurs.
//
// LES LECTURES DU FILM SEUL VIENNENT DE LA TACHE DE FOND quand il y en a une (cf.
// film_scan_fond.go) : on les attend ICI, juste avant que le pont les verse ; sans tache, le pont
// les lit lui-meme sur le roster de cette cuisson.
func (s *filmScan) lirePontDIdentite() grammar.LecturesDuPont {
	return grammar.ScanPontDIdentite(s.fc, grammar.OptionsDuPont{
		Balayage: s.scan, Carte: s.opt.MapQuant,
		RosterDesMorts: func(d []types.Death) []uint64 { return rosterOf(d, s.opt.RosterXUIDs) },
		FaitsDuFilm:    s.fond.lesFaits(),
	})
}

// publierPontDIdentite pose le pont lu et ses TELEPORTATIONS, et observe l'etape `translocations`
// (premiere etape observee). Les autres lectures du pont sont posees par les phases, a leur place
// d'avant.
func (s *filmScan) publierPontDIdentite(pont grammar.LecturesDuPont) {
	s.pont = pont
	s.in.Translocations = s.pont.Translocations
	if len(s.in.Translocations) > 0 {
		slog.InfoContext(s.ctx, "translocateur : teleportations lues", "evenements", len(s.in.Translocations))
	}
	s.opt.observe(s.ctx, "translocations", s.in.Translocations)
}

// lirePoussees lit les poussees (knockbacks) du film.
func (s *filmScan) lirePoussees() lectureStats[[]types.Knockback, types.KnockbackStats] {
	k, st := grammar.ScanKnockbacks(s.fc)
	return lectureStats[[]types.Knockback, types.KnockbackStats]{val: k, st: st}
}

// publierPoussees pose les poussees et observe l'etape `knockbacks`.
func (s *filmScan) publierPoussees(lu lectureStats[[]types.Knockback, types.KnockbackStats]) {
	s.in.Knockbacks, s.in.KnockbackStats = lu.val, lu.st
	s.opt.observe(s.ctx, "knockbacks", s.in.Knockbacks)
}

// publierPositions pose les positions que le pont a lues et observe l'etape `positions`. Rien n est lu
// ici ; l'echec de la lecture (`ErrPositions`) est rendu par l'appelant avant.
//
// LE REPLI DU FILTRE DE GENERATION SE COMPTE ICI (lot J5.2) : un slot dont aucune generation n est
// connue a ete lu a la seule generation 1. `grammar` le nomme, la cuisson le compte.
func (s *filmScan) publierPositions() {
	s.in.Positions = s.pont.Positions
	s.opt.Fallbacks.DeclencheN(fallback.NomGenerationVivanteInconnueTag1,
		s.pont.Generations.SlotsEnRepli(s.in.Positions))
	s.opt.observe(s.ctx, "positions", s.in.Positions)
}

// balayerCreations publie les CRÉATIONS DE BIPÈDE : le lien DIRECT corps -> joueur, lu dans le
// default-state du record NEW `ti=35` (lot E2, 2026-09-08). MÊME bande de slots que les
// positions, pour que les deux lectures parlent des mêmes corps. Absence NON fatale — le
// registre dégrade alors sur le pont par morts et le PUBLIE
// (`coverage.bridge.bridgeNamedLives`). Elles ont ete lues par le pont : rien n est lu ici.
func (s *filmScan) balayerCreations() {
	creations, creaStats, err := s.pont.Creations, s.pont.StatsCreations, s.pont.ErrCreations
	if err != nil {
		slog.WarnContext(s.ctx, "creations de bipede illisibles — le registre degrade sur le pont par morts",
			"err", err, "match_id", s.matchID)
		creations, creaStats = nil, types.BipedCreationStats{}
	} else {
		slog.InfoContext(s.ctx, "creation de bipede : records lus",
			"corps", creaStats.Slots, "ancres", creaStats.Anchors, "acceptes", creaStats.Accepted,
			"formeRefusee", creaStats.ShapeBad, "signatureEtrangere", creaStats.SignatureMismatch,
			"motAlternatif", creaStats.OtherWord, "motAlternatifCompte", creaStats.OtherWordCount,
			"porteFermee", creaStats.GateClosed, "tronques", creaStats.Truncated)
	}
	if creaStats.Anchors > 0 && creaStats.Accepted == 0 {
		// L'ALARME DU LECTEUR, ET C'EST LA SEULE : des ancres de la bonne FORME dont aucune ne
		// porte la constante de représentation. Un film dont les bipèdes portent un autre corps
		// que le Spartan multijoueur se lirait ainsi (cf. biped_creation.go).
		slog.WarnContext(s.ctx, "creation de bipede : aucune signature reconnue sur des ancres presentes — "+
			"ce film porte-t-il une autre representation ?",
			"match_id", s.matchID, "ancres", creaStats.Anchors,
			"motAlternatifModal", creaStats.OtherWord, "compte", creaStats.OtherWordCount)
	}
	s.in.BipedCreations = creations
	s.opt.observe(s.ctx, "bipedCreations", s.in.BipedCreations)
}

// lireTirs lit les tirs du film.
func (s *filmScan) lireTirs() lectureNue[[]grammar.FireEvent] {
	v, err := grammar.ScanFireEvents(s.fc)
	return lectureNue[[]grammar.FireEvent]{val: v, err: err}
}

// publierTirs journalise la lecture des tirs, les pose et observe l'etape `fire`.
func (s *filmScan) publierTirs(lu lectureNue[[]grammar.FireEvent]) {
	shots := lu.val
	if lu.err != nil {
		slog.WarnContext(s.ctx, "events de tir illisibles — rejeu sans tirs", "err", lu.err, "match_id", s.matchID)
		shots = nil
	}
	s.in.Fire = shots
	s.opt.observe(s.ctx, "fire", s.in.Fire)
}
