package replay

// film_scan_portage.go — CE QU'UN JOUEUR PORTE : changements d'arme, ramassages natifs, inventaire
// d'image-cle et inventaire delta. SORTI DE `film_scan.go` pour tenir son seuil de longueur quand
// chaque balayage s y coupe en `lire…` (le film, rien d autre) et `publier…` (journal, replis, entrees,
// etape observee), appeles l un derriere l autre a la place d avant.
//
// CE QUE CHAQUE `lire…` TOUCHE ENCORE DU CONTEXTE DU FILM : les lectures bipedes que la marche des
// trames a recueillies (memo), le registre et le profil ; les ramassages natifs et l inventaire delta
// relisent les paquets delta. L inventaire des images-cles, lui, n a aucune lecture : la marche unique
// des images-cles l a deja rendu ([filmScan.lireLesImagesCles]).

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// balayerPortage lit ce qu'un joueur PORTE : changements d'arme, ramassages natifs, inventaire
// d'image-cle et inventaire delta.
func (s *filmScan) balayerPortage() {
	// LE PREDICAT DE SPAWN VIENT DES LOADOUTS QU'ON VIENT DE LIRE : sans lui, la PREMIERE emission d'un
	// emplacement serait comptee comme une prise alors qu'elle peut n'etre que la re-annonce d'une arme
	// deja portee.
	s.publierChangementsDArme(s.lireChangementsDArme(
		spawnSetFrom(s.in.Loadouts, s.in.BirthLoadouts, s.in.BipedCreations)))
	s.publierRamassages(s.lireRamassages())
	s.balayerInventaire()
}

// lireChangementsDArme lit les PRISES ET LACHERS D'ARME : le composant d'identite d'arme n'entre au
// masque du flux delta que lorsqu'un emplacement CHANGE (cf. filmdec/held_weapon_changes.go).
func (s *filmScan) lireChangementsDArme(spawn grammar.SpawnPredicate,
) lectureStats[[]types.HeldWeaponChange, grammar.HeldWeaponChangeStats] {
	return lireAvecStats(grammar.ScanHeldWeaponChanges(s.fc, spawn))
}

// publierChangementsDArme journalise la lecture des changements d'arme, les pose et observe les etapes
// `heldWeaponChanges` et `heldWeaponChanges.stats`. Absence non fatale — le rejeu sort sans ramassages,
// jamais avec des ramassages devines.
func (s *filmScan) publierChangementsDArme(lu lectureStats[[]types.HeldWeaponChange, grammar.HeldWeaponChangeStats]) {
	weaponChanges, wStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "changements d arme illisibles — rejeu sans ramassages", "err", lu.err, "match_id", s.matchID)
		weaponChanges = nil
	} else {
		slog.InfoContext(s.ctx, "ramassage : changements d arme lus",
			"recordsDelta", wStats.Records, "masquePorteur", wStats.WithComponent,
			"emissions", wStats.Emissions, "repetitions", wStats.Repeats)
	}
	s.in.WeaponChanges = weaponChanges
	s.opt.observe(s.ctx, "heldWeaponChanges", s.in.WeaponChanges)
	s.opt.observe(s.ctx, "heldWeaponChanges.stats", wStats)
}

// lireRamassages lit les RAMASSAGES NATIFS : l'evenement `biped_pickup` de la liste d'evenements, en
// tete des paquets delta. AUTRE SOURCE que le canal des changements d'arme (qui lit un composant du
// bipede PENDANT la traversee d'un record) : celui-ci lit des bits que personne d'autre ne lit, avant la
// trame. Il date a la milliseconde ET nomme le ramasseur.
func (s *filmScan) lireRamassages() lectureStats[[]types.BipedPickup, types.BipedPickupStats] {
	return lireAvecStats(grammar.ScanBipedPickups(s.fc))
}

// publierRamassages journalise la lecture des ramassages natifs, les pose et observe les etapes
// `pickups` et `pickups.stats`. Absence non fatale.
func (s *filmScan) publierRamassages(lu lectureStats[[]types.BipedPickup, types.BipedPickupStats]) {
	pickups, pStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "ramassages natifs illisibles — rejeu sans ramassages natifs", "err", lu.err, "match_id", s.matchID)
		pickups, pStats = nil, types.BipedPickupStats{}
	} else {
		slog.InfoContext(s.ctx, "ramassage natif : evenements lus",
			"paquets", pStats.Packets, "type9", pStats.Type9, "type8", pStats.Type8,
			"publies", pStats.Published, "listesMultiples", pStats.MultiEvent,
			"refusesSansRef", pStats.RefusedNoRef, "refusesSansIdentifiant", pStats.RefusedNoCatalog,
			"refusesHorsBande", pStats.RefusedOffBand, "refLargeInattendue", pStats.UnexpectedWideRef)
	}
	s.in.Pickups, s.in.PickupStats = pickups, pStats
	s.opt.observe(s.ctx, "pickups", s.in.Pickups)
	s.opt.observe(s.ctx, "pickups.stats", s.in.PickupStats)
}

// balayerInventaire publie l'inventaire complet des images-cles, puis lit et publie son suivi dans les
// paquets delta. MÊMES images-clés, MÊME horloge, même record de biped que les armes portées.
func (s *filmScan) balayerInventaire() {
	s.publierInventaire()
	s.publierInventaireDelta(s.lireInventaireDelta())
}

// publierInventaire pose l'inventaire des images-cles que la marche unique a rendu, compte le repli du
// plafond de grenades et observe les etapes `inventory` et `inventory.stats`. Rien n est lu ici.
func (s *filmScan) publierInventaire() {
	// LE PLAFOND DE GRENADES N EST PAS FOURNI (0) : la lecture applique `grammar.DefaultGrenadeMax`,
	// et c est un REPLI NOMME ET COMPTE (D14) — le plafond est une donnee de MODE, pas une
	// constante. Il se compte ICI depuis le lot J4.2 : la lecture est descendue en `grammar`, qui
	// nomme ses replis et ne les compte pas (ADR 0034 D-4). Depuis le lot D1.2 de 2.7.d1, seule la
	// FENETRE derriere la lecture l applique (la grammaire lit les compteurs sans plafond) : il ne se
	// compte que si un record non admis lui a ete donne. Un catalogue de familles vide ne lit rien.
	familles := loadoutFamilies()
	if len(familles) > 0 && s.etats.Admission.FenetresInventaire > 0 {
		s.opt.Fallbacks.Declenche(fallback.NomPlafondGrenadeParDefaut)
	}
	// Absence non fatale — un rejeu sans grenades reste un rejeu valide. L'inventaire vient de la
	// marche unique des images-clés (`balayerPositions`).
	inventory, invStats, err := s.etats.Inventaire, s.etats.StatsInventaire, s.errEtats
	if len(familles) == 0 {
		inventory, invStats, err = nil, types.KeyframeInventoryStats{}, nil
	}
	if err != nil {
		slog.WarnContext(s.ctx, "inventaire illisible — rejeu sans grenades ni munitions", "err", err, "match_id", s.matchID)
		inventory = nil
	} else {
		slog.InfoContext(s.ctx, "inventaire : lectures de keyframe",
			"chunks", invStats.Chunks, "chunksIllisibles", invStats.ChunksUnread,
			"imagesCles", invStats.Keyframes, "records", invStats.Records,
			"grenadesParAncre", invStats.GrenadesByAnchor, "grenadesParPosition", invStats.GrenadesByPosition)
	}
	s.in.Inventory = inventory
	s.opt.observe(s.ctx, "inventory", s.in.Inventory)
	s.opt.observe(s.ctx, "inventory.stats", invStats)
}

// lireInventaireDelta lit l'inventaire suivi dans les paquets DELTA : les compteurs de grenades (i22) et
// le jeu selectionne (i47), transmis AU CHANGEMENT donc places la ou l'etat bouge.
func (s *filmScan) lireInventaireDelta() lectureStats[[]types.InventoryDelta, grammar.InventoryDeltaStats] {
	return lireAvecStats(grammar.ScanInventoryDeltas(s.fc))
}

// publierInventaireDelta journalise la lecture de l'inventaire delta, le pose et observe les etapes
// `inventoryDeltas` et `inventoryDeltas.stats`. Absence non fatale — l'axe des grenades retombe sur les
// seules images-cles.
func (s *filmScan) publierInventaireDelta(lu lectureStats[[]types.InventoryDelta, grammar.InventoryDeltaStats]) {
	invDeltas, dStats := lu.val, lu.st
	if lu.err != nil {
		slog.WarnContext(s.ctx, "inventaire delta illisible — grenades sans rafraichissement entre images-cles",
			"err", lu.err, "match_id", s.matchID)
		invDeltas = nil
	} else {
		slog.InfoContext(s.ctx, "inventaire delta : lectures",
			"recordsDelta", dStats.Records, "masqueAvecI22", dStats.WithI22,
			"i22Lues", dStats.I22Read, "i22Implausibles", dStats.Implausible,
			"masqueAvecI47", dStats.WithI47, "i47Lues", dStats.I47Read,
			"accord", dStats.Accord, "accordVerifies", dStats.AccordChecked,
			"canalMunitionsRefuse", dStats.AmmoRefused)
	}
	s.in.InventoryDeltas = invDeltas
	s.in.InventoryDeltaAmmoRefused = dStats.AmmoRefused
	s.opt.observe(s.ctx, "inventoryDeltas", s.in.InventoryDeltas)
	s.opt.observe(s.ctx, "inventoryDeltas.stats", dStats)
}
