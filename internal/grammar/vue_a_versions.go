package grammar

import "github.com/JGtm/halo-infinite-film-decoder/internal/profile"

// vue_a_versions.go — LA VERSION DE CHAQUE GENRE DE MESSAGE : CELLE DU JEU, CELLE DU FILM, ET LA
// REGLE QUI DIT SI LA GRAMMAIRE DU JEU LIT LA VUE A D UN FILM (lot LN, regle en deux classes du lot
// VA de la campagne de grammaire).
//
// # CE QUE LE JEU LIT
//
// Le lecteur d un message consulte la version de SON genre par `FUN_141102ed0(genre)` : en rejeu
// de film (`FUN_1404f2b4c`), `FUN_1428e1c64` rend le u32 `film + 0xCB208 + genre * 4` — la table
// par genre de `chunk_00` ([profile.FilmIdentity.TypeVersions]) ; hors film, la table NATIVE
// `DAT_14474cd90` (paires {version, taille de la structure du message}, la taille etant nulle
// exactement pour les treize genres vides de [tableDesGenresVueA]). Dix lecteurs la consultent (35,
// 36, 40, 48, 89, 90, 91, 93, 97, 114), et seulement pour separer une version ancienne de la leur.
//
// # LA REGLE, EN DEUX CLASSES (decisions de l utilisateur du 2026-10-04)
//
// La grammaire de l executable est celle de SES versions : un ecrivain d une autre version d un genre
// a pu ecrire autre chose, et le lecteur du jeu ne sait pas le lire (les branches qu il porte
// s arretent a sa propre version). Deux tables lues se comparent, sans seuil, sans build nomme et sans
// version majeure :
//
//	EGALE     le film declare les [GenresVueA] genres, chacun a sa version native : l ecrivain du film
//	          ecrit les messages que le lecteur porte. Film RECENT : la fin de sa vue A est le debut
//	          de sa vue B.
//	PREFIXE   le film declare MOINS de genres, chacun a sa version native — un ecrivain plus ancien,
//	          que la table ne suffit pas a identifier au lecteur porte (le tir a composantes des
//	          films a 121 genres en differe, mesure). La fin de sa vue A ne vaut que prouvee paquet
//	          par paquet.
//	ILLISIBLE section d identification absente, table vide ou plus longue, une version differente :
//	          la vue A n est pas lue au-dela de sa tete.
//
// Les deux classes lisent la vue A jusqu a son terminateur ([lireLaVueA]), et sa fin est rangee avec
// elle (`lecture.Paquet.VueA`). Ce que la marche fait de cette fin depend de la classe
// ([debutParLaVueA]) : le debut de la vue B. Un genre au-dela du cardinal du film n existe pas chez son
// ecrivain : le lire arrete la vue A.
//
// # LA VERSION MAJEURE DU FILM N ENTRE PAS DANS LA REGLE (lu ; mesure)
//
// Avant de jouer un film, le jeu lit sa version majeure, le u32 de tete de `chunk_00`
// ([FilmMajorVersion]) : `FUN_1428e219c` (appele par `FUN_140ba23e4` @1423b9bd7) ne lit le film, et
// ne reporte son bit de controle de corruption (`singleton + 0x1AE` <- `film + 0xCB45C`), que sous
// `*film == 0x29`. Cette garde decide de ce que L EXECUTABLE joue ; elle ne dit rien de ce que
// l ecrivain d un film a ecrit. Ce que l ecrivain declare de ses messages est la table par genre de
// `chunk_00` (`film + 0xCB208`), et c est elle que le lecteur du jeu consulte en rejeu de film pour
// separer les versions d un genre ([FilmIdentity.TypeVersions]).
//
// Un film HI_1_12_0 (majeure 0x28, une majeure que l executable lu ne joue pas) declare les 123 genres
// de la table native, chacun a sa version native : sa table est IDENTIQUE a celle de l executable, genre
// par genre — vrai des huit films de test, et des films de ce build releves dans le cache. Il est donc
// de classe EGALE : un ecrivain dont la table est celle du lecteur ecrit les messages que le lecteur
// lit. La mesure le confirme sans lecture du jeu en 0x28 : sur ces films, la vue A que nous lisons
// finit au bit ou le temoin independant place le debut de la vue B, paquet apres paquet, et la
// ranger comme prouvee seulement (classe PREFIXE) refusait cette fin chaque fois que la marche qui la
// suit ne fermait pas le paquet pour une autre cause — puis retombait sur une localisation fausse. La
// version majeure ne separe donc rien que la table ne separe deja : les films des majeures 0x25 a 0x27 et
// le film HI_1_11_0 (0x28, 122 genres) sont classes par leur table.
//
// # LA NUMEROTATION DES GENRES, PRESUMEE POUR LES FILMS PREFIXE
//
// Comparer les versions genre par genre suppose que le genre i du film est le message i de
// l executable : des genres ajoutes en fin de table, aucun insere. Ce n est pas lu (seul l executable
// HI_1_13_0 est lu, `FUN_140e453b4` y enregistre ses descripteurs). La colonne des versions vaut 1 sur
// 104 genres sur 123, et la derniere version differente de 1 est au genre 107
// ([dernierGenreAVersionDistinctive]) : une insertion avant lui deplacerait sa version et rendrait la
// table illisible, une insertion entre 108 et 120 decalerait les genres d un film PREFIXE sans changer
// sa classe. Pour un film PREFIXE, les genres 108 a `genres - 1` sont donc lus sous les charges
// natives sur cette seule presomption, et seule la preuve de fermeture ([debutParLaVueA]) protege
// l usage de E. Aucun canal de production ne lit un genre de vue A au-dela de la tete (lu : [teteDe]
// prend le premier, [listeAnnoncee] le compte) ; la suite des genres rangee dans `lecture.Paquet.VueA`
// dit ou la presomption commence (`lecture.VueA.PremierPresume`, [premierGenrePresume]).

// classeDeLaVueA est la classe de la table des genres que le film declare.
type classeDeLaVueA uint8

// Les classes de la table des genres d un film.
const (
	// vueAIllisible : la vue A du film ne se lit pas au-dela de sa tete.
	vueAIllisible classeDeLaVueA = iota
	// vueAPrefixe : la fin de la vue A ne vaut que prouvee paquet par paquet — la table du film est
	// un prefixe STRICT de la table native.
	vueAPrefixe
	// vueAEgale : la table du film est EGALE a la table native, genre par genre.
	vueAEgale
)

// dernierGenreAVersionDistinctive est le plus haut genre dont la version native differe de 1
// ([versionsNativesDesGenres]) : au-dela, la numerotation des genres d un film PREFIXE est presumee
// (cf. l en-tete).
const dernierGenreAVersionDistinctive = 107

// grammaireDeLaVueA est ce que le film declare de la grammaire de sa vue A, et ce que la carte de son
// match en fait connaitre. Elle vient du film ([grammaireDeLaVueASousFilm]) et de nulle part
// ailleurs : ce n est pas un reglage du balayage, et elle ne voyage pas avec le profil de balayage
// ([ProfilDeBalayage]) — seulement avec le lecteur de la vue A ([Lecteur.vueA]).
type grammaireDeLaVueA struct {
	// classe et genres : la classe de la table des genres du film et son cardinal
	// ([classeDesGenres]) ; [vueAIllisible] : la vue A ne se lit que jusqu a sa tete.
	classe classeDeLaVueA
	genres int
	// script : ce que le film declare du prefixe du message Script ([scriptDuFilm]).
	script etatDuScript
	// positions : les tables de position de la region jouee de la carte du match, pour les
	// positions a index des genres 5 et 6 ([tablesDeLaRegionJouee]) ; vides sans carte.
	positions tablesDePosition
	// variante : ce que la variante de partie du film decide du genre 116 ([varianteDuFilm]).
	variante varianteDeLaVueA
}

// grammaireDeLaVueASousFilm derive d un profil la grammaire de la vue A que le film declare : sa table
// des genres et la simulation de son enregistreur, lues dans la section d identification de
// `chunk_00`, sa variante de partie, lue dans le corps de `chunk_00`, et les tables de la region
// jouee de l entree de catalogue du profil.
func grammaireDeLaVueASousFilm(p profile.Profile) grammaireDeLaVueA {
	g := grammaireDeLaVueA{script: scriptDuFilm(p), positions: tablesDeLaRegionJouee(p.Map()),
		variante: varianteDuFilm(p)}
	g.classe, g.genres = tableDesGenresDuFilm(p)
	return g
}

// versionsNativesDesGenres est la colonne version de `DAT_14474cd90` (123 paires de 8 octets), un
// chiffre par genre dans l ordre des genres.
const versionsNativesDesGenres = "" +
	"1111111111" + "1111111121" + "1111111111" + "3111143111" + "2111112121" + // 0-49
	"1111112111" + "2611111112" + "1111111111" + "1515111112" + "3312111411" + // 50-99
	"1111111311" + "1111111111" + "111" // 100-122

// versionNative rend la version native d un genre (0 <= genre < GenresVueA).
func versionNative(genre int) uint32 {
	return uint32(versionsNativesDesGenres[genre] - '0')
}

// tableDesGenresDuFilm applique la regle au profil d un film : la classe de la table des genres qu il
// declare (section d identification de `chunk_00`, [profile.Profile.Identity]) et son cardinal (0 pour
// un film illisible). La version majeure n y entre pas (cf. l en-tete). C est le seul site ou la
// classe d un film est decidee.
func tableDesGenresDuFilm(p profile.Profile) (classeDeLaVueA, int) {
	if !p.IdentityRead() {
		return vueAIllisible, 0
	}
	return classeDesGenres(p.Identity().TypeVersions)
}

// premierGenrePresume rend le rang, dans la suite `genres` lue dans la vue A d un film de classe
// `classe`, du premier genre dont la numerotation est presumee et non lue : le premier genre au-dela
// de [dernierGenreAVersionDistinctive] d un film PREFIXE (cf. l en-tete) : le message de
// l executable qu il est est presume, et la position des genres qui le suivent depend de sa charge,
// lue sous la presomption. `len(genres)` quand aucun ne l est.
func premierGenrePresume(classe classeDeLaVueA, genres []int) int {
	if classe == vueAPrefixe {
		for i, g := range genres {
			if g > dernierGenreAVersionDistinctive {
				return i
			}
		}
	}
	return len(genres)
}

// classeDesGenres est la regle sur la table que le film declare (`film + 0xCB208`) : sa classe et
// son cardinal.
func classeDesGenres(declarees []uint32) (classeDeLaVueA, int) {
	if len(declarees) == 0 || len(declarees) > GenresVueA {
		return vueAIllisible, 0
	}
	for genre, v := range declarees {
		if v != versionNative(genre) {
			return vueAIllisible, 0
		}
	}
	if len(declarees) == GenresVueA {
		return vueAEgale, GenresVueA
	}
	return vueAPrefixe, len(declarees)
}
