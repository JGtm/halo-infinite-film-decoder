package replay

// film_scan.go — LA PHASE DES CALQUES GARDES PAR LE MODE, ET L INDEX DES PHASES DE BALAYAGE D UN FILM.
//
// SORTI DE `build_from_film.go` le 2026-09-14 (lot 1.0 du PLAN_DECODEUR_FILM) : le fichier
// depassait les 500 lignes du depot une fois la sequence extraite de `BuildFromFilm`. La coupure
// suit une responsabilite — la, le CHEMIN (verrou, largeurs, contexte du film, puis assemblage) ;
// ici, ce que chaque phase LIT. Les autres phases vivent a cote, chacune dans son fichier, ou chaque
// balayage se coupe en `lire…` (le film, rien d autre) et `publier…` (journal, replis, entrees, etape
// observee) :
//
//   - positions : `film_scan_positions.go` (puis `film_scan_images_cles.go`, `film_scan_naissances.go`) ;
//   - monde : `film_scan_monde.go` ; etats de mouvement : `film_scan_mouvement.go` ;
//   - portage : `film_scan_portage.go` ; capacites : `film_scan_capacites.go` ;
//   - pont d identite : `film_scan_pont.go`.
//
// PRE-REQUIS DE TOUTES LES METHODES DE CES FICHIERS : les
// largeurs d axe de la carte installees. `scanFilmInputs` est leur unique appelant, et
// `BuildFromFilm` tient les deux (cf. build_from_film.go).
//
// GARDE-RAIL QUI LIT CES FICHIERS : `observe_test.go` — il descend de `scanFilmInputs` dans ces
// methodes pour reconstituer la liste ORDONNEE des etapes observees.

// balayerCalquesGardes lit les trois calques que l'APPELANT commande : le marqueur de portage du
// drapeau, l'etat des zones et l'anneau d'armement de la bombe. Sans la garde correspondante
// dans `Options`, aucun octet n'est lu — ce paquet ne devine aucun mode.
func (s *filmScan) balayerCalquesGardes() {
	// MARQUEUR DE PORTAGE : le controle independant du calque du drapeau, lu aux images-cles du
	// MEME film — sur les seuls films de CTF (cf. build_objectives_live.go).
	s.in.FlagMarks = decodeFilmCarrierMarks(s.ctx, s.etats.Marques, s.errEtats, s.matchID, s.opt.Flag,
		s.opt.consultations())
	s.opt.observe(s.ctx, "carrierMarks", s.in.FlagMarks)
	// PROPRIETES RESEAU ti=13 : UN SEUL BALAYAGE, DEUX CONSOMMATEURS ET DEUX GARDES. L'etat des
	// zones (jauge de capture, proprietaire) le veut sur les matchs dont l'appelant a fourni le
	// catalogue de zones ; la JAUGE DE RETOUR du drapeau le veut sur les films de CTF. Les deux
	// gardes s'excluent en pratique — un match n'est pas a la fois CTF et colline — mais elles ne
	// sont pas la MEME garde, et chaque canal ne recoit que ce que SA garde autorise : un CTF ne
	// doit pas se mettre a publier `coverage.zones.scanned = true` (cf. build_zones.go et
	// flag_return_gauge.go).
	s.balayerProprietesTi13()
	// ANNEAU D'ARMEMENT ti=12 : la jauge d'armement de la bombe, lue dans les paquets delta du
	// MEME film — sur les seuls matchs que l'appelant reconnait Assaut armable (cf.
	// bomb_armings.go ; jamais One Bomb, ou le canal ne tient pas).
	s.in.BombReads = decodeFilmBombReads(s.ctx, s.fc, s.matchID, s.opt.Bomb)
	s.opt.observe(s.ctx, "bombReads", s.in.BombReads)
}

// balayerPont, poserLeFilDesMorts : SORTIS le 2026-10-22 (lot detonations, correction CI) dans
// `film_scan_pont.go` — deplacement PUR, aucune ligne de logique changee. Ce fichier restait au
// plafond legal (500) avant que ce lot n'ajoute la lecture de la detonation ; l'ajouter a
// `plafondsParFichier` n'etait pas une reponse valide (table close au 2026-09-16), la scission en
// est une.
