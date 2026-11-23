package replay

// film_scan_capacites.go — L'EQUIPEMENT PORTE ET LES CAPACITES : identite, changements, equipement
// porte, camouflage, grappin, impulsions et charges. SORTI DE `film_scan.go` pour tenir son seuil de
// longueur quand chaque balayage s y coupe en `lire…` (le film, rien d autre) et `publier…` (journal,
// entrees, etape observee), appeles l un derriere l autre a la place d avant.
//
// CE QUE CHAQUE `lire…` TOUCHE ENCORE DU CONTEXTE DU FILM : les lectures bipedes que la marche des
// trames a recueillies (memo), le registre, le profil de balayage, et pour l equipement porte le memo
// `equipementPorte` de la marche. Aucune ne verse de repli ni n ecrit de journal.

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// balayerCapacites lit les canaux delta de l'equipement porte — identite, changements, equipement porte,
// camouflage, grappin — puis les impulsions et les charges.
func (s *filmScan) balayerCapacites() {
	s.publierRangsDeCapacite(s.lireRangsDeCapacite())
	// Le temoin de NAISSANCE vient des positions BRUTES lues plus haut : sans lui, une reapparition
	// equipee serait comptee comme un ramassage, ce qui double le decompte sur les modes ou les joueurs
	// renaissent equipes.
	s.publierChangementsDEquipement(s.lireChangementsDEquipement(birthOfLives(s.in.Positions)))
	s.publierEquipementPorte(s.lireEquipementPorte())
	s.balayerEtatsActifs()
	s.balayerImpulsionsEtCharges()
}

// lireRangsDeCapacite lit l'identite de la capacite portee : dans les paquets DELTA, sur la MEME
// horloge. Rare (une transmission par vie environ) mais elle porte le rang COMPLET, la ou les
// images-cles ne voient que 16..23.
func (s *filmScan) lireRangsDeCapacite() lectureStats[[]types.AbilityRank, types.AbilityRankStats] {
	return lireAvecStats(grammar.ScanAbilityRanks(s.fc))
}

// publierRangsDeCapacite journalise la lecture des identites, les pose et observe les etapes
// `abilityRanks` et `abilityRanks.stats`. Absence non fatale — le rejeu retombe sur la seule fenetre des
// images-cles.
func (s *filmScan) publierRangsDeCapacite(lu lectureStats[[]types.AbilityRank, types.AbilityRankStats]) {
	abilityRanks, aStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "identites de capacite illisibles — rejeu sans rang complet", "err", lu.err, "match_id", s.matchID)
		abilityRanks = nil
	} else {
		slog.InfoContext(s.ctx, "capacites : lectures d i48",
			"recordsDelta", aStats.Records, "masqueAvecI48", aStats.WithI48,
			"lues", aStats.Read, "illisibles", aStats.Unread, "sansIdentite", aStats.Gated)
	}
	s.in.AbilityRanks = abilityRanks
	s.opt.observe(s.ctx, "abilityRanks", s.in.AbilityRanks)
	s.opt.observe(s.ctx, "abilityRanks.stats", aStats)
}

// lireChangementsDEquipement lit les RAMASSAGES ET CONSOMMATIONS D'EQUIPEMENT : meme composant que les
// identites (i48), autre question — non plus « que porte ce joueur » mais « que vient-il de ramasser ou
// d'user ». `naissance` dit l instant de naissance d un slot (nil sans position).
func (s *filmScan) lireChangementsDEquipement(naissance func(uint32) (uint64, bool),
) lectureStats[[]types.EquipmentChange, types.EquipmentChangeStats] {
	return lireAvecStats(grammar.ScanEquipmentChanges(s.fc, naissance))
}

// publierChangementsDEquipement journalise la lecture des changements d'equipement, les pose et observe
// les etapes `equipmentChanges` et `equipmentChanges.stats`. Absence non fatale.
func (s *filmScan) publierChangementsDEquipement(lu lectureStats[[]types.EquipmentChange, types.EquipmentChangeStats]) {
	equipChanges, eStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "changements d equipement illisibles — rejeu sans ramassages d equipement",
			"err", lu.err, "match_id", s.matchID)
		equipChanges, eStats = nil, types.EquipmentChangeStats{}
	} else {
		slog.InfoContext(s.ctx, "equipement : changements lus",
			"emissions", eStats.Walk.Read, "vies", eStats.Lives,
			"ramassages", eStats.Taken, "consommations", eStats.Spent,
			"reapparitions", eStats.Spawned, "manqueesEstimees", eStats.MissedEstimate)
	}
	s.in.EquipmentChanges, s.in.EquipmentChangeStats = equipChanges, eStats
	s.opt.observe(s.ctx, "equipmentChanges", s.in.EquipmentChanges)
	s.opt.observe(s.ctx, "equipmentChanges.stats", s.in.EquipmentChangeStats)
}

// balayerEtatsActifs lit le camouflage et le grappin : deux voies des paquets DELTA, sur la MEME
// horloge que les positions.
func (s *filmScan) balayerEtatsActifs() {
	s.publierCamouflage(s.lireCamouflage())
	s.publierGrappin(s.lireGrappin())
}

// lireCamouflage lit l'etat du camouflage : la voie i28 queue[1] (cf. filmdec/camo_state.go).
func (s *filmScan) lireCamouflage() lectureStats[[]types.CamoRead, grammar.CamoStateStats] {
	return lireAvecStats(grammar.ScanCamoStates(s.fc))
}

// publierCamouflage journalise la lecture du camouflage, la pose et observe les etapes `camoStates` et
// `camoStates.stats`. Absence non fatale — le rejeu sort sans episodes de camouflage, jamais avec des
// episodes devines.
func (s *filmScan) publierCamouflage(lu lectureStats[[]types.CamoRead, grammar.CamoStateStats]) {
	camoStates, cStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "etat de camouflage illisible — rejeu sans episodes de camo", "err", lu.err, "match_id", s.matchID)
		camoStates = nil
	} else {
		slog.InfoContext(s.ctx, "camouflage : lectures d i28 queue[1]",
			"recordsDelta", cStats.Records, "masqueAvecI28", cStats.WithI28,
			"lues", cStats.Read, "illisibles", cStats.Unread, "sansVoie", cStats.NoChannel)
	}
	s.in.CamoStates = camoStates
	s.opt.observe(s.ctx, "camoStates", s.in.CamoStates)
	s.opt.observe(s.ctx, "camoStates.stats", cStats)
}

// lireGrappin lit les evenements de grappin : le corps tag==3 d'i59 (cf. filmdec/grapple_state.go).
func (s *filmScan) lireGrappin() lectureStats[[]types.GrappleRead, grammar.GrappleStats] {
	return lireAvecStats(grammar.ScanGrappleReads(s.fc))
}

// publierGrappin journalise la lecture du grappin, la pose et observe les etapes `grappleReads` et
// `grappleReads.stats`. Absence non fatale — le rejeu sort sans tractions de grappin, jamais avec des
// tractions devinees.
func (s *filmScan) publierGrappin(lu lectureStats[[]types.GrappleRead, grammar.GrappleStats]) {
	grappleReads, gStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "evenements de grappin illisibles — rejeu sans tractions", "err", lu.err, "match_id", s.matchID)
		grappleReads = nil
	} else {
		slog.InfoContext(s.ctx, "grappin : lectures d i59 tag==3",
			"recordsDelta", gStats.Records, "masqueAvecI59", gStats.WithI59,
			"lues", gStats.Read, "illisibles", gStats.Unread,
			"tag3", gStats.Tag3, "corpsCasses", gStats.BodyBroken)
	}
	s.in.GrappleReads = grappleReads
	s.opt.observe(s.ctx, "grappleReads", s.in.GrappleReads)
	s.opt.observe(s.ctx, "grappleReads.stats", gStats)
}

// balayerImpulsionsEtCharges lit l'USAGE des capacites : les impulsions et les charges
// restantes. L'IDENTITE, elle, vient d'i48 (deja balaye par balayerCapacites).
func (s *filmScan) balayerImpulsionsEtCharges() {
	s.publierImpulsions(s.lireImpulsions())
	s.publierCharges(s.lireCharges())
}

// lireImpulsions lit les IMPULSIONS DE CAPACITE : le corps tag==1 des MEMES composants (i57 et son
// jumeau non predit i59), lu dans les paquets DELTA sur la MEME horloge (cf.
// filmdec/ability_impulses.go). C'est le canal d'usage du PROPULSEUR, mesure au lot R8.
func (s *filmScan) lireImpulsions() lectureStats[[]types.AbilityImpulse, types.AbilityImpulseStats] {
	return lireAvecStats(grammar.ScanAbilityImpulses(s.fc))
}

// publierImpulsions journalise la lecture des impulsions, les pose et observe l'etape `abilityImpulses`.
// Absence non fatale — le rejeu sort sans impulsions, jamais avec des impulsions devinees.
func (s *filmScan) publierImpulsions(lu lectureStats[[]types.AbilityImpulse, types.AbilityImpulseStats]) {
	impulses, iStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "impulsions de capacite illisibles — rejeu sans impulsions", "err", lu.err, "match_id", s.matchID)
		impulses, iStats = nil, types.AbilityImpulseStats{}
	} else {
		slog.InfoContext(s.ctx, "capacites : lectures de tag d i57/i59",
			"recordsDelta", iStats.Records, "masqueAvecI57", iStats.WithI57,
			"masqueAvecI59", iStats.WithI59, "lues", iStats.Read, "illisibles", iStats.Unread,
			"tag1", iStats.Tag1, "composantAbsent", iStats.Absent)
	}
	s.in.AbilityImpulses, s.in.AbilityImpulseStats = impulses, iStats
	s.opt.observe(s.ctx, "abilityImpulses", s.in.AbilityImpulses)
}

// lireCharges lit les CHARGES D'EQUIPEMENT RESTANTES : les emplacements ARMES du composant i56, lus dans
// les paquets DELTA sur la MEME horloge (cf. filmdec/ability_charges.go). C'est le canal des charges
// mesure au lot R11.
func (s *filmScan) lireCharges() lectureStats[[]types.AbilityCharge, types.AbilityChargeStats] {
	return lireAvecStats(grammar.ScanAbilityCharges(s.fc))
}

// publierCharges journalise la lecture des charges, les pose et observe l'etape `abilityCharges`.
// Absence non fatale — le rejeu sort sans releve de charges, jamais avec des charges devinees.
func (s *filmScan) publierCharges(lu lectureStats[[]types.AbilityCharge, types.AbilityChargeStats]) {
	charges, chStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "charges d equipement illisibles — rejeu sans releve de charges", "err", lu.err, "match_id", s.matchID)
		charges, chStats = nil, types.AbilityChargeStats{}
	} else {
		slog.InfoContext(s.ctx, "capacites : lectures d i56",
			"recordsDelta", chStats.Records, "masqueAvecI56", chStats.WithI56,
			"lues", chStats.Read, "illisibles", chStats.Unread,
			"emplacementsArmes", chStats.Armed, "composantAbsent", chStats.Absent)
	}
	s.in.AbilityCharges, s.in.AbilityChargeStats = charges, chStats
	s.opt.observe(s.ctx, "abilityCharges", s.in.AbilityCharges)
}
