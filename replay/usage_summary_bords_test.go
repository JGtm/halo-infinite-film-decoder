package replay

// usage_summary_bords_test.go — deux cas de bord de l'attribution des poses d'un slot :
// la place reprise à l'image même du lâcher, et le mur déployé juste après la mort de son
// poseur.

import "testing"

func usageParXUID(s UsageSummary) map[string]UsagePlayerSummary {
	out := map[string]UsagePlayerSummary{}
	for _, p := range s.Players {
		out[p.XUID] = p
	}
	return out
}

// TestUsageSummary_PlaceReprisePileALImageDuLacher — l'objet lâché à la mort de 111 porte
// `t0 = finVie + 1`. Quand le repreneur du slot (222) commence sa vie à cette même image, c'est
// 111 qui l'a lâché : le repreneur n'a encore rien porté.
func TestUsageSummary_PlaceReprisePileALImageDuLacher(t *testing.T) {
	doc := &ReplayDocument{
		SchemaVersion: SchemaVersion, FrameIntervalMS: 100, FrameCount: 1000,
		Roster: []RosterEntry{{XUID: "111", FilmIndex: 0}, {XUID: "222", FilmIndex: 1}},
		Tracks: []Track{
			{Slot: 5, XUID: "111", StartFrame: 0, EndFrame: 100},
			{Slot: 5, XUID: "222", StartFrame: 101, EndFrame: 400},
		},
		EquipmentPlacements: []EquipmentPlacement{
			{Family: "grapple", Origin: OriginDropped, Owner: 5, T0: 101, ID: "0x1"},
		},
	}
	by := usageParXUID(BuildUsageSummary(doc))
	if got := by["111"].DroppedObjects; got != 1 {
		t.Errorf("111 : %d lâcher(s), attendu 1 — l'objet lâché à sa mort lui revient même "+
			"quand le repreneur du slot naît à la même image", got)
	}
	if got := by["222"].DroppedObjects; got != 0 {
		t.Errorf("222 : %d lâcher(s), attendu 0 — il n'avait rien porté", got)
	}
}

// TestUsageSummary_MurDeployeApresLaMortDuPoseur — un mur dont le panneau naît une à trois
// images après la mort de son poseur reste crédité à ce poseur (même règle que les objets
// lâchés en mourant), jamais au repreneur du slot ni à personne.
func TestUsageSummary_MurDeployeApresLaMortDuPoseur(t *testing.T) {
	doc := &ReplayDocument{
		SchemaVersion: SchemaVersion, FrameIntervalMS: 100, FrameCount: 1000,
		Roster: []RosterEntry{{XUID: "111", FilmIndex: 0}, {XUID: "222", FilmIndex: 1}},
		Tracks: []Track{
			{Slot: 5, XUID: "111", StartFrame: 0, EndFrame: 100},
			{Slot: 5, XUID: "222", StartFrame: 200, EndFrame: 400},
		},
		EquipmentPlacements: []EquipmentPlacement{
			{Family: "wall", Origin: OriginDeployed, Owner: 5, T0: 101, ID: "0x528fce46"},
			{Family: "wall", Origin: OriginDeployed, Owner: 5, T0: 103, ID: "0x528fce46"},
		},
	}
	by := usageParXUID(BuildUsageSummary(doc))
	if got := by["111"].DeployedByFamily["wall"]; got != 2 {
		t.Errorf("111 : %d mur(s) déployé(s), attendu 2 — crédités au poseur mort", got)
	}
	if got := by["222"].DeployedByFamily["wall"]; got != 0 {
		t.Errorf("222 : %d mur(s), attendu 0 — le repreneur du slot n'a rien posé", got)
	}
}
