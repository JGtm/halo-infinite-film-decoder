package replay

// document_regles.go — LES REGLES DE LA VARIANTE QUI CONCERNENT LES REAPPARITIONS, LES TIRS ALLIES ET LE
// RETOUR DU DRAPEAU.
//
// `rules` : ce que la variante de partie du film ecrit du delai de reapparition, des penalites ajoutees a la
// reapparition apres un suicide et apres une trahison, du drapeau des tirs allies et du delai de retour d un
// drapeau tombe. Un champ que le film n ecrit pas est absent : sa valeur par defaut n est pas devinee.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

// hachageDelaiDeRetourDuDrapeau est le hachage de l option de la variante qui donne le delai, en secondes, au bout
// duquel un drapeau tombe revient tout seul (`CTFOptions_FlagReturnTime`).
const hachageDelaiDeRetourDuDrapeau uint32 = 0x2b16b76f

// MatchRules est ce que la variante de partie du film ecrit des reapparitions et des tirs allies. Un
// champ absent n est pas ecrit par le film : sa valeur par defaut n est pas devinee.
type MatchRules struct {
	// MinRespawnS est le delai minimal entre la mort et la reapparition, en secondes.
	MinRespawnS *int64 `json:"minRespawnS,omitempty"`
	// SuicidePenaltyS, BetrayalPenaltyS sont les secondes que la variante ajoute a la reapparition apres un
	// suicide, apres une trahison.
	SuicidePenaltyS  *int64 `json:"suicidePenaltyS,omitempty"`
	BetrayalPenaltyS *int64 `json:"betrayalPenaltyS,omitempty"`
	// FlagReturnS est le delai, en secondes, au bout duquel un drapeau tombe revient tout seul : la jauge de
	// retour se remplit d une unite sur ce delai quand personne n est dans sa zone.
	FlagReturnS *float64 `json:"flagReturnS,omitempty"`
	// FriendlyFire est le drapeau des tirs allies de la variante.
	FriendlyFire *bool `json:"friendlyFire,omitempty"`
}

// vide dit que la variante n a donne aucune de ces regles.
func (r MatchRules) vide() bool {
	return r.MinRespawnS == nil && r.SuicidePenaltyS == nil && r.BetrayalPenaltyS == nil && r.FlagReturnS == nil &&
		r.FriendlyFire == nil
}

// rulesOf rend les regles que la variante du film ecrit, ou le zero quand le film n a pas d identite ou que
// sa variante n a pas ete lue.
func rulesOf(id *profile.FilmIdentity) MatchRules {
	if id == nil || !id.InfosDeMatch.VarianteLue {
		return MatchRules{}
	}
	i := id.InfosDeMatch
	retour := optionNombre(i, hachageDelaiDeRetourDuDrapeau)
	return MatchRules{
		MinRespawnS:      i.DelaiDeReapparitionSecondes,
		SuicidePenaltyS:  i.PenaliteDeSuicideSecondes,
		BetrayalPenaltyS: i.PenaliteDeTrahisonSecondes,
		FlagReturnS:      retour,
		FriendlyFire:     i.TirsAllies,
	}
}

// optionNombre rend la valeur de l option numerique de hachage `hachage` que la variante pose, et nil quand elle ne la pose
// pas ou qu elle n est pas numerique. Une option n est posee qu une fois ; la premiere l emporte.
func optionNombre(i profile.InfosDeMatch, hachage uint32) *float64 {
	for _, o := range i.Options {
		if o.Hachage == hachage && o.Genre == profile.OptionNombre {
			return &o.Valeur
		}
	}
	return nil
}
