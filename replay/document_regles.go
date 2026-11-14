package replay

// document_regles.go — LES REGLES DE LA VARIANTE QUI CONCERNENT LES REAPPARITIONS ET LES TIRS ALLIES.
//
// `rules` : ce que la variante de partie du film ecrit du delai de reapparition, des penalites ajoutees a la
// reapparition apres un suicide et apres une trahison, et du drapeau des tirs allies. Un champ que le film
// n ecrit pas est absent : sa valeur par defaut n est pas devinee.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// MatchRules est ce que la variante de partie du film ecrit des reapparitions et des tirs allies. Un
// champ absent n est pas ecrit par le film : sa valeur par defaut n est pas devinee.
type MatchRules struct {
	// MinRespawnS est le delai minimal entre la mort et la reapparition, en secondes.
	MinRespawnS *int64 `json:"minRespawnS,omitempty"`
	// SuicidePenaltyS, BetrayalPenaltyS sont les secondes que la variante ajoute a la reapparition apres un
	// suicide, apres une trahison.
	SuicidePenaltyS  *int64 `json:"suicidePenaltyS,omitempty"`
	BetrayalPenaltyS *int64 `json:"betrayalPenaltyS,omitempty"`
	// FriendlyFire est le drapeau des tirs allies de la variante.
	FriendlyFire *bool `json:"friendlyFire,omitempty"`
}

// vide dit que la variante n a donne aucune de ces regles.
func (r MatchRules) vide() bool {
	return r.MinRespawnS == nil && r.SuicidePenaltyS == nil && r.BetrayalPenaltyS == nil && r.FriendlyFire == nil
}

// rulesOf rend les regles que la variante du film ecrit, ou le zero quand le film n a pas d identite ou que
// sa variante n a pas ete lue.
func rulesOf(id *profile.FilmIdentity) MatchRules {
	if id == nil || !id.InfosDeMatch.VarianteLue {
		return MatchRules{}
	}
	i := id.InfosDeMatch
	return MatchRules{
		MinRespawnS:      i.DelaiDeReapparitionSecondes,
		SuicidePenaltyS:  i.PenaliteDeSuicideSecondes,
		BetrayalPenaltyS: i.PenaliteDeTrahisonSecondes,
		FriendlyFire:     i.TirsAllies,
	}
}
