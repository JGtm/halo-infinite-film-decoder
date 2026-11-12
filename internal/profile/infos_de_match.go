package profile

// infos_de_match.go — CE QUE `chunk_00` DIT DU MATCH QU IL ENREGISTRE, COMME VALEUR.
//
// C est un fait du film, persiste avec son identite ([FilmIdentity.InfosDeMatch]) : la variante de
// partie (nom, description, score de victoire, durees, dotation, options), les identifiants de la
// variante de carte et l heure de debut.
//
// UN CHAMP NON LU EST ABSENT, JAMAIS ZERO. Un pointeur nil dit que le film ne porte pas la valeur :
// soit l enregistreur ne l ecrit pas (un champ optionnel de la structure Bond egal a son defaut n est
// pas ecrit, et la valeur par defaut n est pas devinee ici), soit la variante n a pas pu etre lue
// ([InfosDeMatch.VarianteLue] faux). La disposition des champs est mesuree contre un decodeur tiers
// (libfilmshell, licence MIT) ; elle n est pas relue dans l executable du jeu.

// GUID est un identifiant de 16 octets tel que le film l ecrit : champs petit-boutistes (u32, u16,
// u16, puis huit octets), dans l ordre du flux.
type GUID [16]byte

// Genres de la valeur d une option de variante : l octet 4 de ses huit octets de valeur.
const (
	OptionBooleenne uint8 = 2
	OptionNombre    uint8 = 3
)

// OptionDeVariante est une option que la variante pose : le hachage du nom de l option et sa valeur.
type OptionDeVariante struct {
	// Hachage : le murmur3 du nom de l option en minuscules (le film ne porte que lui).
	Hachage uint32
	// Genre : [OptionBooleenne] ou [OptionNombre]. Une option d un autre genre n est pas rendue.
	Genre uint8
	// Valeur : 0 ou 1 pour une option booleenne, le nombre (flottant de 32 bits) sinon.
	Valeur float64
}

// DotationDeDepart est la dotation du premier chargement de la variante.
type DotationDeDepart struct {
	// Armes : les identifiants de tag des armes de depart que la variante nomme.
	Armes []uint32
	// Aleatoire : la variante tire les armes de depart au hasard.
	Aleatoire bool
	// Equipement : l identifiant de tag de l equipement de depart, nil quand elle n en nomme pas.
	Equipement *uint32
}

// InfosDeMatch : les infos de match que le film porte.
type InfosDeMatch struct {
	// DebutUnix : l heure de debut, secondes Unix (l horodatage de l enregistreur, quelques secondes
	// apres le debut du match). nil : le champ n est pas dans le tampon ou vaut zero.
	DebutUnix *uint32
	// CarteAssetID, CarteVersionID : les identifiants (asset et version) de la variante de carte.
	// nil : hors du tampon, ou tous les octets a zero.
	CarteAssetID   *GUID
	CarteVersionID *GUID
	// VarianteLue : la variante de partie a ete lue en entier. Faux : tous les champs ci-dessous
	// sont absents.
	VarianteLue bool
	// Nom, Description : le nom et la description de la variante, dans la premiere langue de ses
	// chaines (l anglais). nil : absents, vides ou pas du texte UTF-8.
	Nom         *string
	Description *string
	// ScoreDeVictoire, DureeSecondes : le score a atteindre et la duree limite de la manche.
	ScoreDeVictoire *int64
	DureeSecondes   *int64
	// DelaiDeReapparitionSecondes : le delai minimal entre la mort et la reapparition.
	DelaiDeReapparitionSecondes *int64
	// ProlongationSecondes : la duree de la prolongation.
	ProlongationSecondes *int64
	// DerniereChanceSecondes : la duree de la derniere chance, seulement quand la prolongation en
	// porte une (le drapeau de la variante est leve).
	DerniereChanceSecondes *int64
	// Dotation : le premier chargement de la variante ; nil quand elle n en ecrit aucun.
	Dotation *DotationDeDepart
	// Options : les options que la variante pose, dans l ordre du film.
	Options []OptionDeVariante
}
