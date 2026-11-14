package replay

// options_defauts.go — LES ACCESSEURS DE DEFAUT D `Options` : le pas de grille, le compteur de
// replis et le seuil de points publies, que l appelant peut laisser a zero.

import "github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"

func (o Options) frameIntervalMS() int {
	if o.FrameIntervalMS > 0 {
		return o.FrameIntervalMS
	}
	return DefaultFrameIntervalMS
}

// compteurDeReplis rend le compteur de la cuisson, en creant le sien quand l'appelant n'en a pas
// fourni. MEME PATRON QUE `frameIntervalMS` / `minPoints` : le defaut vit ici, une seule fois,
// et l'assemblage n'a pas a le connaitre.
func (o Options) compteurDeReplis() *fallback.Compteur {
	if o.Fallbacks != nil {
		return o.Fallbacks
	}
	return fallback.NouveauCompteur()
}

func (o Options) minPoints() int {
	if o.MinPoints > 0 {
		return o.MinPoints
	}
	return DefaultMinPoints
}
