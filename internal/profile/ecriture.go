package profile

// ecriture.go — CE QUE L ECRIVAIN D UN BUILD ECRIT ET QUE LE FLUX NE DIT PAS, LA OU C EST PROUVE.
//
// Trois dispositions de la grammaire ne se lisent ni dans l executable lu (HI_1_13_0) ni dans le flux :
// elles se mesurent sur des films, et la mesure dit sur quels builds elles valent. Une disposition mesuree
// sur un build ne vaut pas pour un autre : un ecrivain plus ancien a ecrit autre chose, et la lire sous la
// grammaire d un autre build lit des bits a un mauvais alignement sans que rien ne le signale. Chaque champ de
// [EcritureDuBuild] est donc vrai sur les builds ou une ligne de [tableProfilEcriture] l etablit, et faux
// partout ailleurs : un build absent de la table, un build vide (film sans section d identification), un
// build futur. La lecture s arrete alors la ou elle ne sait plus, ou garde la lecture qui ferme les paquets.

// EcritureDuBuild est ce que l ecrivain d un build ecrit et que le flux ne dit pas.
type EcritureDuBuild struct {
	// ChargesDeVueAMesurees : les charges des genres 12 (`biped_melee_clang`), 14 (`PlayEffectOnObject`) et 106
	// (`ObjectCollisionDamage`) de la vue A se lisent sous leur disposition mesuree
	// (`grammar/vue_a_charges_mesurees.go`). Faux : la vue A s arrete sur le message d un de ces genres.
	ChargesDeVueAMesurees bool
	// CreationParPorte : un record NEW ne lit sa liste de composants que sous une porte a 1, et ne lit pas de
	// porte pour un archetype dont le masque de creation est vide (`grammar/traverse.go`). Faux : la porte et
	// le masque se lisent toujours.
	CreationParPorte bool
	// ProprieteGereeComplete : l etat de creation d un record NEW de l archetype 13 (`managed-object-property-name`)
	// porte la charge de chacune de ses etiquettes, comme celui d une image-cle
	// (`grammar/default_state_ti13_neuf.go`). Faux : il n en lit que les etiquettes.
	ProprieteGereeComplete bool
}

// Les champs de la table du profil que ce fichier pose, et la date de leur mesure.
const (
	champChargesDeVueA    = "VueA.ChargesMesurees"
	champCreationParPorte = "Creation.ParPorte"
	champProprieteGeree   = "Creation.ProprieteGereeComplete"
	dateEcritureMesuree   = "2026-11-12"
	valeurEcritureVraie   = "vrai"
)

// EcritureDuBuildPour rend ce que l ecrivain du build `build` ecrit, ou la valeur zero pour un build que la
// mesure n etablit pas. Les trois champs valent sur les memes builds ; les preuves sont dans
// [tableProfilEcriture], une ligne par build et par champ.
func EcritureDuBuildPour(build string) EcritureDuBuild {
	switch build {
	case buildHI1131, buildHI1120, buildHI1110:
		return EcritureDuBuild{ChargesDeVueAMesurees: true, CreationParPorte: true, ProprieteGereeComplete: true}
	}
	return EcritureDuBuild{}
}

// tableProfilEcriture : les lignes de [EcritureDuBuildPour], une par build et par champ. Un build absent de
// cette table n a aucune des trois dispositions.
//
//nolint:funlen // une table de donnees : la decouper masquerait la lecture ligne a ligne
func tableProfilEcriture() []LigneProfil {
	return []LigneProfil{
		{
			Cle: cleBuildHI1131, Champ: champChargesDeVueA, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "86 paquets dont la vue A est lue au-dela d un message de genre 12, 14 ou 106, fermes au bit 86 " +
				"sur 86 (53 films) ; fin de vue A egale au debut de vue B du decodeur tiers sur 4 paquets sur 4",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1120, Champ: champChargesDeVueA, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "99 paquets a charge mesuree fermes au bit 99 sur 99 (96 films) ; fin de vue A egale au debut " +
				"de vue B du decodeur tiers sur 10 paquets sur 10 de cinq films",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1110, Champ: champChargesDeVueA, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "56 paquets a charge mesuree fermes au bit 56 sur 56 (38 films) ; fin de vue A egale au debut " +
				"de vue B du decodeur tiers sur 5 paquets sur 5 d un film. Le build HI_1_10_0 (562 paquets sur 760) " +
				"et HI_1_9_0 (9 sur 18) n y sont pas : la fin de vue A lue y contredit le debut de vue B donne par " +
				"la signature du slot 123 dans 47 cas testables sur 47, et aucune largeur constante ne le retrouve",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1131, Champ: champCreationParPorte, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "chaine des records egale a celle du decodeur tiers sur 43 512 paquets sur 43 513 d un film ; " +
				"paquets fermes 99,955 % -> 99,985 % sur 53 films, aucune perte",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1120, Champ: champCreationParPorte, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "chaine des records egale a celle du decodeur tiers sur 100 % des paquets de cinq films ; paquets " +
				"fermes 99,917 % -> 99,942 % sur 96 films, une perte qui est une fermeture fausse (debut de vue B " +
				"different de celui du decodeur tiers)",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1110, Champ: champCreationParPorte, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "chaine des records plus proche de celle du decodeur tiers que sous la lecture precedente sur cinq " +
				"films sur cinq ; paquets fermes 99,498 % -> 99,699 % sur 38 films, six pertes qui sont des fermetures " +
				"fausses (debut de vue B a plus de mille bits de celui du decodeur tiers). Sur HI_1_10_0 la meme " +
				"lecture ferme 3 571 paquets de plus et en perd 420, sur HI_1_8_0 810 et 172 : elle n y est pas prouvee",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1131, Champ: champProprieteGeree, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "le gain propre a la charge des etiquettes est de 56 paquets fermes sur 49 films, aucune perte",
			Date:   dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1120, Champ: champProprieteGeree, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "113 paquets fermes de plus sur 96 films, aucune perte ; un paquet que la lecture precedente ne fermait " +
				"pas (un record de ti=13 de 1 218 bits, lu sur 195) se ferme au bit",
			Date: dateEcritureMesuree,
		},
		{
			Cle: cleBuildHI1110, Champ: champProprieteGeree, Valeur: valeurEcritureVraie, Source: ProvenanceMesuree,
			Preuve: "33 paquets fermes de plus sur 38 films, aucune perte. Sur HI_1_10_0 la meme lecture ferme 62 paquets " +
				"de plus et en perd 22 : elle n y est pas prouvee",
			Date: dateEcritureMesuree,
		},
	}
}
