package replay

// film_scan_monde.go — LE BALAYAGE DU MONDE : LUNETTE, POSES, PIECES ENGENDREES, SOCLES, VEHICULES.
//
// Chaque balayage se coupe en deux methodes, appelees l une derriere l autre a la place qu il occupait :
// `lire…` ne fait que lire le film et rend une valeur (resultat, erreur, ce qu il faudra journaliser ou
// verser) ; `publier…` prend cette valeur et fait, dans l ordre d avant, ce qui s observe — journal,
// versement des replis, entrees du balayage, etape observee. Les lectures vivent dans `lecture_monde.go`
// (poses, pieces, socles, vehicules). SORTI DE `film_scan.go` pour tenir son seuil de longueur.
//
// CE QUE CHAQUE `lire…` TOUCHE ENCORE DU CONTEXTE DU FILM : ses memos (bande bipede, registre, profil,
// pistes et creations du monde), et pour les socles et les vehicules les largeurs du bloc MPP, posees
// sur le contexte le temps de la lecture puis restituees. Aucun ne verse de repli ni n ecrit de journal.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// balayerMonde lit la lunette, puis les objets du monde : poses d'equipement, socles, vehicules,
// et les trois calques gardes par le mode (marqueur de portage, zones, anneau de la bombe).
func (s *filmScan) balayerMonde() {
	s.publierLunette(s.lireLunette())
	s.publierPoses(s.lirePoses())
	s.publierPiecesEngendrees(s.lirePiecesEngendrees())
	// LES LARGEURS MPP DES SOCLES ET DES VEHICULES SONT CELLES QUE LA CALIBRATION DES POSES VIENT DE
	// MESURER sur ce film (cf. build_ground_weapons.go) : les poses se publient avant.
	mpp := s.in.PlacementStats.Calibration.Widths
	s.publierSocles(s.lireSocles(mpp))
	s.publierVehicules(s.lireVehicules(mpp))
	s.balayerCalquesGardes()
}

// lireLunette lit les bascules de LUNETTE (schema 24) : elles vivent dans la liste d'evenements en
// tete de paquet, pas dans les records — un balayage separe, sans verrou (il ne touche aucun etat
// global de decodage). Le maintien est borne : au-dela, on cesse d'affirmer plutot que de prolonger
// une entree dont la sortie n'a pas ete lue (cf. grammar.ZoomStateAt).
func (s *filmScan) lireLunette() []grammar.ZoomEvent {
	return grammar.ScanZoomEvents(s.fc)
}

// publierLunette pose les bascules de lunette et observe l'etape `zoomEvents`.
//
// LA RECONSTRUCTION (`buildScopedLookup`) VIT DANS `FilmInputs.applyTo` (lot 1.0) : elle est PURE, et
// la porter la permet au fixture de figer une LISTE d'evenements plutot qu'une fermeture. Son cout
// quitte donc la mesure de l'etape `zoomEvents` ; il ne lit aucun octet de film.
func (s *filmScan) publierLunette(ev []grammar.ZoomEvent) {
	s.in.ZoomEvents = ev
	s.opt.observe(s.ctx, "zoomEvents", s.in.ZoomEvents)
}

// lirePoses lit les POSES d'equipement : records de CREATION de l'archetype 37, sur la MEME horloge
// (cf. equipment_placements.go).
func (s *filmScan) lirePoses() lecturePoses {
	return decodeFilmPlacementsLecture(s.fc, &s.world)
}

// publierPoses verse et journalise ce que la lecture des poses a vu, puis pose les poses et leurs
// statistiques et observe les etapes `placements` et `placements.stats`.
func (s *filmScan) publierPoses(lu lecturePoses) {
	s.in.Placements, s.in.PlacementStats = journaliserPoses(s.ctx, s.matchID, lu)
	s.opt.observe(s.ctx, "placements", s.in.Placements)
	s.opt.observe(s.ctx, "placements.stats", s.in.PlacementStats)
}

// lirePiecesEngendrees lit « UNE PIECE A ETE ENGENDREE » : l'evenement de liste 103, qui DESIGNE la vie
// de l'objet cree (lot 1.9.1, cf. equipment_origin.go). Lecture de TETE de liste, sans verrou de
// decodage — elle ne touche aucun global de filmdec — et sans cout de chargement : le contexte du film
// est deja ouvert.
func (s *filmScan) lirePiecesEngendrees() lectureEvenementsPiece {
	return decodeFilmSpawnEventsLecture(s.fc)
}

// publierPiecesEngendrees journalise la lecture des evenements 103, pose les evenements et leurs
// statistiques et observe l'etape `spawnEvents`.
func (s *filmScan) publierPiecesEngendrees(lu lectureEvenementsPiece) {
	s.in.SpawnEvents, s.in.SpawnStats = journaliserEvenementsPiece(s.ctx, s.matchID, lu)
	s.opt.observe(s.ctx, "spawnEvents", s.in.SpawnEvents)
}

// lireSocles lit les SOCLES : archetypes 42 (armes) et 37 (power-ups), sur la MEME horloge, AUX LARGEURS
// MPP que la calibration des POSES vient de mesurer sur ce film (cf. build_ground_weapons.go).
func (s *filmScan) lireSocles(mpp profile.MPPWidths) lecturePads {
	return decodeFilmPadScansLecture(s.fc, &s.world, mpp)
}

// publierSocles verse les replis des largeurs, journalise la lecture des deux voies, pose les socles et
// observe l'etape `pads`.
func (s *filmScan) publierSocles(lu lecturePads) {
	s.in.Pads = journaliserPads(s.ctx, s.fc, s.matchID, lu)
	s.opt.observe(s.ctx, "pads", s.in.Pads)
}

// lireVehicules lit les VEHICULES : archetype 40, sur la MEME horloge et AUX MEMES largeurs MPP que les
// socles — le mot d'identite du chassis se lit derriere les memes deux champs de largeur variable
// (cf. build_vehicles.go). CINQ lectures du film deja charge, aucune E/S.
func (s *filmScan) lireVehicules(mpp profile.MPPWidths) lectureVehicules {
	return decodeFilmVehicleScanLecture(s.fc, &s.world, mpp)
}

// publierVehicules verse les replis des largeurs, journalise la lecture, pose les vehicules et observe
// l'etape `vehicles`.
func (s *filmScan) publierVehicules(lu lectureVehicules) {
	s.in.Vehicles = journaliserVehicules(s.ctx, s.fc, s.matchID, lu)
	s.opt.observe(s.ctx, "vehicles", s.in.Vehicles)
}
