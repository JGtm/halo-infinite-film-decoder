package replay

import "github.com/JGtm/halo-infinite-film-decoder/internal/flatjson"

// document_medal_labels.go — LES LIBELLÉS ET VISUELS DES MÉDAILLES, résolus par l'hôte.
//
// `ReplayDocument.MedalLabels` nomme et dessine les médailles de `Medals`, comme `VehicleLabels` le
// fait des familles de châssis : calque À LA REQUÊTE, REMPLI PAR L'HÔTE d'après le référentiel du
// titre, jamais écrit par la cuisson (le référentiel peut s'améliorer, les artefacts sont cuits
// hors ligne). Ces types sont les jumeaux de `domain/replaydoc/medals.go` de LevelUp ; la parité
// des deux formes est tenue par le golden de forme et par les gardes de l'hôte.

// MedalLabel est ce qu'il faut pour NOMMER et DESSINER une médaille du document : son libellé par
// langue, sa description, son image (ou sa découpe de feuille de sprites) et sa rareté. La table
// `medalLabels` est clé par `Medal.Name`. Une médaille que le référentiel ne connaît pas n'a AUCUNE
// entrée : le lecteur garde le nom du film, il n'invente ni libellé ni visuel.
type MedalLabel struct {
	// Text est le libellé de la médaille, une valeur par langue ; écrit à plat, une langue sans
	// texte n'est pas émise.
	Text Text `json:"i18n_text,omitempty"`
	// Description : ce que la médaille récompense, absente quand le référentiel n'en porte aucune.
	Description *MedalDescription `json:"description,omitempty"`
	// Img est l'URL de l'icône ; vide quand le titre sert une découpe de feuille (`Sprite`).
	Img string `json:"img,omitempty"`
	// Sprite : la découpe de feuille de sprites du titre qui en sert (l'inverse de `Img`).
	Sprite *MedalSprite `json:"sprite,omitempty"`
	// Difficulty est la rareté du référentiel (`Normal`, `Heroic`, `Legendary`, `Mythic`) ; vide
	// quand le référentiel n'en dit rien.
	Difficulty string `json:"difficulty,omitempty"`
}

// MedalDescription est la description d'une médaille, une valeur par langue.
type MedalDescription struct {
	Text Text `json:"i18n_text,omitempty"`
}

// MedalSprite est la découpe d'une feuille de sprites : l'URL de la feuille, le décalage et la
// taille native de la cellule, en pixels.
type MedalSprite struct {
	Sheet  string `json:"sheet"`
	Left   int    `json:"left"`
	Top    int    `json:"top"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type (
	medalLabelWire       MedalLabel
	medalDescriptionWire MedalDescription
)

// Une langue sans texte n'est pas émise : une médaille que le référentiel ne nomme pas dans une
// langue laisse le lecteur replier sur la langue par défaut.
var (
	medalLabelFlat       = flatjson.FlatLayout{After: flatjson.FlatEnd, OmitEmpty: true}
	medalDescriptionFlat = flatjson.FlatLayout{OmitEmpty: true}
)

// MarshalJSON écrit les langues à plat (cf. `document_labels_json.go`).
func (m MedalLabel) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](medalLabelWire(m), medalLabelFlat)
}

// UnmarshalJSON lit les langues à plat.
func (m *MedalLabel) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*medalLabelWire)(m))
}

// MarshalJSON écrit les langues à plat (cf. `document_labels_json.go`).
func (d MedalDescription) MarshalJSON() ([]byte, error) {
	return flatjson.MarshalFlat[Text](medalDescriptionWire(d), medalDescriptionFlat)
}

// UnmarshalJSON lit les langues à plat.
func (d *MedalDescription) UnmarshalJSON(b []byte) error {
	return flatjson.UnmarshalFlat[Text](b, (*medalDescriptionWire)(d))
}
