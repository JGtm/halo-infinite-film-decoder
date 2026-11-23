package decfilm

// options_vitesse.go — LES OPTIONS DE VITESSE, PAR LA FACADE.
//
// Re-export de replay.OptionsVitesse : le porteur unique du nombre de fils. La valeur zero vaut un
// fil (comportement actuel) ; le nombre effectif est plafonne par GOMAXPROCS et par MaxFils.

import "github.com/JGtm/halo-infinite-film-decoder/replay"

// OptionsVitesse : voir replay.OptionsVitesse (valeur zero = un fil).
type OptionsVitesse = replay.OptionsVitesse

// MaxFils : borne du nombre de fils, voir replay.MaxFils.
const MaxFils = replay.MaxFils
