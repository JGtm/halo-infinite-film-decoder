package replay

// film_scan_naissances.go — LE BALAYAGE DES DOTATIONS DE NAISSANCE (lot M3.2 de la campagne
// « retours rejeu », 2026-09-23).
//
// Il vit hors de `film_scan.go` pour la raison de `film_scan_mouvement.go` : le fichier frôlait
// le seuil de 500 lignes (règle 5 de CLAUDE.md). Les méthodes sont appelées par `balayerPositions`,
// juste après les images-clés.

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// lireNaissances lit la DOTATION DE NAISSANCE de chaque corps : les emplacements d'arme de son
// record NEW de création (cf. `grammar/birth_loadouts.go`). `creations` sont les créations de bipède
// publiées plus haut, qui l'ancrent. CE QU'ELLE TOUCHE DU CONTEXTE : le registre et les lectures bipèdes.
//
// ELLE VIENT APRÈS LES CRÉATIONS, QUI L'ANCRENT, ET AVANT LES PRISES D'ARME, dont elle qualifie la
// première émission de chaque vie (`spawnSetFrom`). Une dotation dont le record ne se ferme pas
// n'est PAS publiée — aucune lecture de repli — et le refus est compté par cause.
func (s *filmScan) lireNaissances(creations []grammar.BipedCreation,
) lectureStats[[]types.BirthLoadout, types.BirthLoadoutStats] {
	return lireAvecStats(grammar.ScanBirthLoadouts(s.fc, creations))
}

// publierNaissances journalise la lecture des dotations, les pose et observe les etapes `birthLoadouts`.
//
// ABSENCE NON FATALE : le rejeu sort sans armes de naissance, jamais avec des armes devinées.
func (s *filmScan) publierNaissances(lu lectureStats[[]types.BirthLoadout, types.BirthLoadoutStats]) {
	births, st := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "dotations de naissance illisibles — rejeu sans armes de naissance",
			"err", lu.err, "match_id", s.matchID)
		births, st = nil, types.BirthLoadoutStats{Creations: len(s.in.BipedCreations)}
	} else {
		slog.InfoContext(s.ctx, "naissance : dotations lues",
			"creations", st.Creations, "lues", st.Read, "desynchronisees", st.Desync,
			"debordantes", st.Overflow, "nonConfirmees", st.Unconfirmed,
			"sansEmplacement", st.NoWeaponComponent, "fermeesParDelta", st.ClosedByDelta,
			"fermeesParNouveauLie", st.ClosedByBoundNew,
			"fermeesParNouveauAnticipe", st.ClosedByAnticipatedNew, "match_id", s.matchID)
	}
	s.in.BirthLoadouts, s.in.BirthLoadoutStats = births, st
	s.opt.observe(s.ctx, "birthLoadouts", s.in.BirthLoadouts)
	s.opt.observe(s.ctx, "birthLoadouts.stats", s.in.BirthLoadoutStats)
}
