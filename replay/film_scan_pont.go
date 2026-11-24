package replay

// film_scan_pont.go — LA PUBLICATION DU PONT D IDENTITE (lot J4.3, 2026-09-26) : fil des morts, sieges
// et occupants, index des joueurs, origine d horloge, et les lancers, projectiles et detonations. La
// lecture du pont lui-meme ([filmScan.lirePontDIdentite]) vit dans `film_scan_positions.go`. Chaque
// balayage s y coupe en `lire…` (le film, rien d autre) et `publier…` (journal, replis, entrees, etape
// observee), appeles l un derriere l autre a la place d avant.
//
// CE QUE CHAQUE `lire…` TOUCHE ENCORE DU CONTEXTE DU FILM : les lancers, projectiles et detonations
// relisent les paquets delta (memos de lectures bipedes, registre, profil) ; les sieges et occupants
// lisent en plus les memos de leurs deux lectures et attendent la table de la tache de fond. Les
// publications du fil des morts, de l index et de l origine ne lisent rien : le pont les a rendues.
//
// `balayerPont` ET `poserLeFilDesMorts` Y REJOIGNENT LEURS DEUX APPELES le 2026-10-22 (lot
// detonations, correction CI) : la lecture de la detonation de projectile
// (`grammar.ScanProjectileDetonations`) avait fait depasser `film_scan.go` au-dela du plafond de
// 500 lignes du depot (CLAUDE.md regle 5). DEPLACEMENT PUR, aucune ligne de logique ne change ;
// `balayerPont` appelait deja [lireSiegesEtOccupants], defini juste au-dessus dans ce meme
// fichier — le deplacement les rapproche plutot que d ouvrir un troisieme fichier.

import (
	"context"
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// siegesEtOccupants porte les deux lectures du film qui nomment un corps SANS le fil des morts :
// la table des sieges que `chunk_00` ecrit (le lien direct `index <-> xuid <-> gamertag`) et les
// occupants par entite ti=9, lus dans la meme passe que l'equipe de chaque index.
type siegesEtOccupants struct {
	table   grammar.FilmPlayerTable
	equipes map[int]int
	rapport grammar.TeamScanReport
	entites grammar.PlayerEntityScan
	// listages : les plages de listage des humains (paquets de type 8) et leurs denominateurs.
	listages      []types.RosterListing
	statsListages types.RosterListingStats
	// entrees : de qui est chaque ligne du tableau des scores, et leurs denominateurs.
	entrees      []types.StatborgEntry
	statsEntrees types.StatborgEntryStats
}

// lectureSieges est ce que la lecture des sieges et des occupants a rendu, sans journal ni compteur :
// la table du film lue aux listes de la session avec l erreur de sa lecture, les equipes et les
// occupants par entite, les entrees du tableau des scores.
type lectureSieges struct {
	table    grammar.TableDuFilmAuxListages
	errTable error
	lectureOccupants
}

// lectureOccupants est la part de cette lecture qui passe par le contexte du film : l equipe de chaque index
// et les occupants par entite, et les entrees du tableau des scores. La voie des sieges la rend d avance
// (film_scan_voie_sieges.go).
type lectureOccupants struct {
	equipes      map[int]int
	rapport      grammar.TeamScanReport
	entites      grammar.PlayerEntityScan
	entrees      []types.StatborgEntry
	statsEntrees types.StatborgEntryStats
}

// lireSiegesEtOccupantsLecture est LA lecture de ces deux entrees du registre d identite, et la seule :
// la cuisson ([filmScan.lireSieges]) et le document interne du sync ([lireSiegesEtOccupants]) passent
// tous deux par elle, pour que les deux registres d un meme film nomment les memes vies. Le registre
// emploie la table du film AVANT la table d index des chunks (identity_registry_film_table.go) et les
// entites pour le corps d un index que plusieurs occupants se relaient (identity_registry_entites.go) :
// une lecture omise d un cote laisse ce cote sans nom la ou l autre nomme (garde-rail :
// `TestLeSyncEtLaCuissonLisentLesMemesSiegesEtOccupants`).
//
// Elle ne fait que lire : aucun journal, aucun compteur, aucun repli verse. Aucun refus n est fatal : la
// table refusee et son erreur voyagent dans la valeur rendue, que [journaliserSieges] consigne.
//
// `avance` est la table que la tache de fond a deja lue sur CE film ([fondDuFilm.laTable]) ; nil, la
// fonction la lit elle-meme. La table ne depend que du film : lue d avance ou ici, elle est la meme.
// CE QU ELLE TOUCHE DU CONTEXTE : les memos de ses deux lectures (`ScanPlayerTeams`,
// `ScanStatborgEntries`), dont les parcours de corps d image-cle posent les largeurs MPP du format le
// temps de leur boucle puis les restituent.
func lireSiegesEtOccupantsLecture(film *source.Film, fc *grammar.FilmContext, avance *tableLueDAvance,
) lectureSieges {
	var lu lectureSieges
	if avance != nil {
		lu.table, lu.errTable = avance.lu, avance.err
	} else {
		lu.table, lu.errTable = grammar.ScanFilmPlayerTableAuxListages(film)
	}
	lu.lectureOccupants = lireLesOccupants(fc)
	return lu
}

// lireLesOccupants lit les equipes, les occupants par entite et les entrees du tableau des scores sur un
// contexte. La cuisson en serie ([lireSiegesEtOccupantsLecture]), le document interne du sync et la voie des
// sieges de la cuisson a plusieurs fils passent par elle.
func lireLesOccupants(fc *grammar.FilmContext) lectureOccupants {
	var lu lectureOccupants
	lu.equipes, lu.rapport, lu.entites = grammar.ScanPlayerTeams(fc)
	lu.entrees, lu.statsEntrees = grammar.ScanStatborgEntries(fc)
	return lu
}

// journaliserSieges consigne la table du film (journal, compteur du build inconnu), journalise les
// sieges corriges aux listes et rend ce que le balayage publie.
func journaliserSieges(ctx context.Context, matchID string, lu lectureSieges) siegesEtOccupants {
	so := siegesEtOccupants{
		table:         consignerLaTableDuFilm(ctx, lu.table.Table, lu.errTable, matchID),
		equipes:       lu.equipes,
		rapport:       lu.rapport,
		entites:       lu.entites,
		listages:      lu.table.Listages,
		statsListages: lu.table.StatsListages,
		entrees:       lu.entrees,
		statsEntrees:  lu.statsEntrees,
	}
	if lu.table.Corriges > 0 {
		slog.InfoContext(ctx, "rejeu : sieges de la table du film lus aux listes de la session", "match_id", matchID,
			"sieges", lu.table.Corriges, "occupants", len(lu.table.Table.Seats))
	}
	return so
}

// lireSiegesEtOccupants est la lecture des sieges et des occupants avec son journal : la composition de
// [lireSiegesEtOccupantsLecture] et de [journaliserSieges], pour le document interne du sync
// ([PortagesAuSync]), qui n a pas d etage de publication.
func lireSiegesEtOccupants(ctx context.Context, film *source.Film, fc *grammar.FilmContext, matchID string,
	avance *tableLueDAvance) siegesEtOccupants {
	return journaliserSieges(ctx, matchID, lireSiegesEtOccupantsLecture(film, fc, avance))
}

// balayerPont lit les lancers, les projectiles, et publie LE PONT D'IDENTITE : le fil des morts, les
// sieges et occupants, la table d'index de joueur et l'origine d'horloge.
func (s *filmScan) balayerPont() {
	s.publierGrenades(s.lireGrenades())
	s.publierProjectiles(s.lireProjectiles())
	s.publierDetonations(s.lireDetonations())
	s.publierLeFilDesMorts()
	s.publierSieges(s.lireSieges())
	s.publierLIndexDesJoueurs()
	s.publierLOrigineDHorloge()
}

// publierLeFilDesMorts publie le fil des morts que l etage du pont a lu : il NOMME les vies par le pont
// par morts, cale l'horloge des morts et ouvre la lecture de la table d index (cf.
// [filmScan.poserLeFilDesMorts]) ; puis les medailles du meme fil. Rien n est lu ici.
func (s *filmScan) publierLeFilDesMorts() {
	s.poserLeFilDesMorts(s.pont.Morts, s.pont.ErrMorts)
	s.opt.observe(s.ctx, "deaths", s.in.Deaths)
	s.in.Medals, s.in.MedalsRead = s.pont.Medailles, s.pont.MedaillesLues
}

// lireSieges lit la table des joueurs que le film ecrit et les occupants (lot 1.6), par la lecture que
// le document interne du sync emploie aussi. Elle attend la table de la tache de fond, quand il y en a
// une, la ou le balayage la lisait.
func (s *filmScan) lireSieges() lectureSieges {
	if occ, ok := s.voies.occupants(); ok {
		// La voie des sieges a lu les equipes et le tableau des scores ; la table vient de la tache de fond.
		avance := s.fond.laTable()
		return lectureSieges{table: avance.lu, errTable: avance.err, lectureOccupants: occ}
	}
	return lireSiegesEtOccupantsLecture(s.film, s.fc, s.fond.laTable())
}

// publierSieges consigne la table du film puis pose et observe les sieges et les occupants.
//
// LA TABLE DES JOUEURS QUE LE FILM ÉCRIT (lot 1.6) : `chunk_00` porte les 32 sièges du match avec leur
// XUID et leur gamertag, le lien DIRECT que le registre emploie AVANT la table des chunks de
// réplication (cf. film_player_table.go). Un refus est NOMMÉ, journalisé et publié.
//
// L'EQUIPE DE CHAQUE JOUEUR (lot 1.7) : le composant i0 de ti=9 de la trame d'etat, a une position
// DERIVEE de la grammaire. C'est la SEULE source d'equipe du document (V4) ; la base ne fait que
// controler. Un refus est NOMME et publie (`coverage.teams.refusal`). LES ENTITES SORTENT DE LA MEME
// PASSE (lot M2.1) : un occupant par entite, sa presence au pas des images-cles et son equipe. L'etape
// observee reste la TABLE par index, a l'octet : elle est le controle, et son empreinte inchangee prouve
// que la passe unique ne l'a pas touchee.
func (s *filmScan) publierSieges(lu lectureSieges) {
	so := journaliserSieges(s.ctx, s.matchID, lu)
	s.in.FilmTable = so.table
	s.opt.observe(s.ctx, "filmTable", s.in.FilmTable)
	s.in.PlayerTeams, s.in.TeamScan, s.in.PlayerEntities = so.equipes, so.rapport, so.entites
	s.in.RosterListings, s.in.RosterListingStats = so.listages, so.statsListages
	s.in.StatborgEntries, s.in.StatborgEntryStats = so.entrees, so.statsEntrees
	// L'apparence se lit dans les memes paquets de type 8, a la suite, sur le fil de l'appelant : aucune voie parallele.
	s.in.Appearances, s.in.AppearanceStats = grammar.ScanRosterAppearances(s.film)
	s.opt.observe(s.ctx, "playerTeams", s.in.PlayerTeams)
	s.opt.observe(s.ctx, "rosterListings", s.in.RosterListings)
	s.opt.observe(s.ctx, "rosterListings.stats", s.in.RosterListingStats)
	s.opt.observe(s.ctx, "statborgEntries", s.in.StatborgEntries)
	s.opt.observe(s.ctx, "statborgEntries.stats", s.in.StatborgEntryStats)
	s.opt.observe(s.ctx, "appearances", s.in.Appearances)
	s.opt.observe(s.ctx, "appearances.stats", s.in.AppearanceStats)
}

// publierLIndexDesJoueurs publie la table d index de joueur que l etage du pont a lue sur le roster de
// cette cuisson (`rosterOf`) : un fil des morts non vide l a toujours fait lire. Rien n est lu ici.
//
// L'index de joueur SE LIT dans le film (cf. player_index.go) : le roster vient du fil des
// morts, et les 5 bits qui précèdent chaque xuid donnent son index. Sans cette table, un tir
// ou un lancer reste publie des que le registre d'identite nomme son slot par ses AUTRES
// lectures — la table des sieges du film (`FilmTable`) en tete.
func (s *filmScan) publierLIndexDesJoueurs() {
	if len(s.in.Deaths) > 0 {
		idx, err := s.pont.Index, s.pont.ErrIndex
		if err != nil {
			slog.WarnContext(s.ctx, "index de joueur illisible — les tireurs ne seront nommes que par les autres "+
				"lectures du registre (table des sieges du film)", "err", err, "match_id", s.matchID)
		}
		table, collisions := injectiveOrEmpty(idx)
		if collisions > 0 {
			slog.WarnContext(s.ctx, "index de joueur NON INJECTIF — table ecartee",
				"collisions", collisions, "match_id", s.matchID)
		}
		s.in.PlayerIndices = table
	}
	s.opt.observe(s.ctx, "playerIndices", s.in.PlayerIndices)
}

// publierLOrigineDHorloge publie l'origine d'horloge du film : deux en-têtes de paquet, aucune
// estimation (cf. origin.go). Son absence n'est pas fatale — le document sort sans origine, et le
// client retombe sur l'appariement. Rien n est lu ici.
func (s *filmScan) publierLOrigineDHorloge() {
	clockUS, err := s.pont.OrigineUS, s.pont.ErrOrigine
	if err != nil {
		slog.WarnContext(s.ctx, "origine d'horloge illisible — rejeu sans origine publiee", "err", err, "match_id", s.matchID)
		clockUS = 0
	}
	s.in.FilmClockOriginUS = clockUS
	s.opt.observe(s.ctx, "clockOrigin", s.in.FilmClockOriginUS)
}

// lireGrenades lit les lancers de grenade : décodés des paquets delta du MÊME film, sur la MÊME
// horloge. Absence non fatale, comme les tirs et les armes portées.
func (s *filmScan) lireGrenades() lectureGrenades {
	if lu, ok := s.voies.grenades(); ok {
		return lu
	}
	return lireLesGrenades(s.fc)
}

// lireLesGrenades est la lecture des lancers sur un contexte, que la chaine et la voie du monde emploient.
func lireLesGrenades(fc *grammar.FilmContext) lectureGrenades {
	g, err := grammar.ScanGrenadeThrows(fc)
	return lectureGrenades{lancers: g, err: err}
}

// publierGrenades journalise la lecture des lancers, les pose et observe l'etape `grenades`.
func (s *filmScan) publierGrenades(lu lectureGrenades) {
	grenades := lu.lancers
	if lu.err != nil {
		slog.WarnContext(s.ctx, "paquets delta illisibles — rejeu sans lancers de grenade", "err", lu.err, "match_id", s.matchID)
		grenades = nil
	}
	s.in.Grenades = grenades
	s.opt.observe(s.ctx, "grenades", s.in.Grenades)
}

// lireProjectiles lit les trajectoires de projectile : memes chunks, meme horloge. Absence non fatale.
func (s *filmScan) lireProjectiles() lectureProjectiles {
	if lu, ok := s.voies.projectiles(); ok {
		return lu
	}
	return lireLesProjectiles(s.fc, &s.world)
}

// lireLesProjectiles est la lecture des trajectoires sur un contexte, que la chaine et la voie du monde
// emploient.
func lireLesProjectiles(fc *grammar.FilmContext, wr *profile.Vec3Range) lectureProjectiles {
	p, err := grammar.ScanProjectiles(fc, wr)
	return lectureProjectiles{pistes: p, err: err}
}

// publierProjectiles journalise la lecture des projectiles, les pose et observe l'etape `projectiles`.
func (s *filmScan) publierProjectiles(lu lectureProjectiles) {
	proj := lu.pistes
	if lu.err != nil {
		slog.WarnContext(s.ctx, "projectiles illisibles — rejeu sans trajectoires", "err", lu.err, "match_id", s.matchID)
		proj = nil
	}
	s.in.Projectiles = proj
	s.opt.observe(s.ctx, "projectiles", s.in.Projectiles)
}

// lireDetonations lit les detonations et impacts de projectile (genres 5/6 de la vue A) : lecture
// seule, sans chunk supplementaire — jamais d'erreur, juste moins de couverture sur un film qui ne les
// ecrit pas.
func (s *filmScan) lireDetonations() []types.ProjectileDetonation {
	if d, ok := s.voies.detonations(); ok {
		return d
	}
	return lireLesDetonations(s.fc)
}

// lireLesDetonations est la lecture des detonations sur un contexte, que la chaine et la voie du monde
// emploient.
func lireLesDetonations(fc *grammar.FilmContext) []types.ProjectileDetonation {
	d, _ := grammar.ScanProjectileDetonations(fc)
	return d
}

// publierDetonations pose les detonations et observe l'etape `detonations`.
func (s *filmScan) publierDetonations(d []types.ProjectileDetonation) {
	s.in.Detonations = d
	s.opt.observe(s.ctx, "detonations", s.in.Detonations)
}

// lectureGrenades est ce que la lecture des lancers de grenade a rendu.
type lectureGrenades struct {
	lancers []grammar.GrenadeThrow
	err     error
}

// lectureProjectiles est ce que la lecture des trajectoires de projectile a rendu.
type lectureProjectiles struct {
	pistes []types.ProjectileTrack
	err    error
}

// poserLeFilDesMorts pose le fil des morts LU par l etage du pont (lot J4.3), son VERDICT
// (`coverage.bridge.deathsFeed`, lot M5.2 des retours rejeu) et le fil lui-meme dans les entrees.
// Un fil VIDE est une mesure, un fil ILLISIBLE une panne, et le document les distingue au lieu de
// les laisser aux seuls journaux.
// Le verdict et sa cause sont des ENTREES depuis le lot M8 (2026-09-24) : ils voyagent dans les
// faits persistes (cf. fil_des_morts_verdict.go).
//
// CE QUE LE FIL ILLISIBLE NE COUPE PLUS (message corrige le 2026-09-23) : les tirs et les
// lancers. Ils sont nommes par la table des sieges que le film ecrit (`chunk_00`, lot 1.6) —
// mesure : 2 838 tirs publies sur `ab526724` quand son fil etait illisible.
//
// UNE METHODE A PART (revue adverse M5, constat R1, 2026-09-24) pour que le trajet « octets ->
// verdict -> document » se teste sur la bobine du depot, que le balayage des positions refuse.
func (s *filmScan) poserLeFilDesMorts(deaths []types.Death, err error) {
	s.in.DeathsFeed = verdictDeLaLecture(deaths, err)
	switch s.in.DeathsFeed.Verdict {
	case DeathsFeedUnreadable:
		slog.WarnContext(s.ctx, "fil des morts illisible — ni calage d horloge des morts, ni table d index, ni "+
			"pont par morts (coverage.bridge.deathsFeed = unreadable)", "err", err, "match_id", s.matchID)
		deaths = nil
	case DeathsFeedEmpty:
		slog.InfoContext(s.ctx, "fil des morts lu et VIDE — aucune mort a nommer (coverage.bridge.deathsFeed = empty)",
			"err", err, "match_id", s.matchID)
		deaths = nil
	}
	s.in.Deaths = deaths
}
