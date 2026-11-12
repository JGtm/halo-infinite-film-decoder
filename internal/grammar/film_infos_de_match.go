package grammar

// film_infos_de_match.go — LES INFOS DE MATCH QUE `chunk_00` PORTE : LA VARIANTE DE PARTIE, LA
// VARIANTE DE CARTE ET L HEURE DE DEBUT.
//
// # D OU VIENT LA DISPOSITION
//
// Les chemins Bond ci-dessous sont MESURES CONTRE UN DECODEUR TIERS (libfilmshell, licence MIT) et
// verifies sur le parc de films ; ils ne sont pas relus dans l executable du jeu. Ils sont relatifs a
// la structure de la variante de partie, le champ 1 d une liste de la structure de tete (la marche
// de [lireLaVarianteDePartie] y entre par les champs 0 a 3, comme ici) :
//
//	3.0.0.0         liste d octets : le bloc des chaines de la variante, chacune finie par un octet nul
//	3.0.1.0.i.1.j   le decalage dans ce bloc de la chaine i (0 : le nom, 1 : la description) pour la
//	                langue j ; la langue 0 est l anglais. Une chaine ecrite une seule fois pour toutes
//	                les langues (une variante sans traduction) a le meme decalage pour chacune
//	1.2.3           delai minimal de reapparition (secondes)
//	1.7.0, 1.7.3    score de victoire, duree limite de la manche (secondes)
//	1.7.9           duree de la prolongation (secondes) ; 1.7.10 : la derniere chance est active ;
//	                1.7.11 : sa duree (secondes)
//	1.6.1.i         les chargements ; 1.6.1.0.1.j.0.0 : tag de l arme j ; 1.6.1.0.1.j.2 : arme
//	                aleatoire ; 1.6.1.0.2.0.0 : tag de l equipement
//	3.9.0.i         les options : .0.0 le hachage du nom, .1.0 huit octets de valeur
//
// L enregistreur n ecrit pas un champ optionnel egal a son defaut : un champ absent d une structure
// lue jusqu a sa fin reste ABSENT dans le resultat, sa valeur par defaut n est pas devinee.
//
// # LES DEUX IDENTIFIANTS DE LA VARIANTE DE CARTE ET L HEURE
//
// Les deux blocs bruts de 16 octets qui ferment la section d identification, juste avant le corps
// ([FilmIdentity.BodyBit]), sont l identifiant d asset et l identifiant de version de la variante de
// carte (mesure contre le meme decodeur tiers). L heure de debut est celle de [lireHorodatage].
//
// # QUAND LA VARIANTE N EST PAS LUE
//
// La variante n est rendue que si la marche du corps ([marcherLeCorps]) l a lue jusqu a la table des
// joueurs ET que cette lecture parcourt la variante sans que le curseur tombe. Sinon
// [profile.InfosDeMatch.VarianteLue] est faux et tous ses champs sont absents.

import (
	"bytes"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

const (
	// octetsDeValeurDOption : la taille de la valeur d une option ; le genre est l octet 4.
	octetsDeValeurDOption = 8
	octetGenreDOption     = 4
	// dernierChampListeDeVariantes : les champs 0 a 3 de la structure de tete sont des listes de variantes.
	dernierChampListeDeVariantes = 3
	// profondeurDeChemin : un chemin complet commence par le champ de liste et le rang de la variante.
	profondeurDeChemin = 2
	// octetsDuGUID : un identifiant de la variante de carte.
	octetsDuGUID = 16
)

// modelesDeChemins : les chemins que la lecture suit, relatifs a la structure de la variante ;
// jokerDeChemin tient la place d un rang. Ils sont poses par [nouveauxModelesDeChemins] et non dans des
// variables de paquet : le decodeur n a pas d etat global.
type modelesDeChemins struct {
	chaines, decalageNom, decalageDesc, reapparition, score, duree, prolongation []int
	chanceActive, derniereChance, chargement, arme, aleatoire, equipement        []int
	optionNom, optionValeur                                                      []int
	suivis                                                                       [][]int
}

// nouveauxModelesDeChemins rend les modeles de chemins de la lecture.
func nouveauxModelesDeChemins() modelesDeChemins {
	m := modelesDeChemins{
		chaines:        []int{3, 0, 0, 0},
		decalageNom:    []int{3, 0, 1, 0, 0, 1, 0},
		decalageDesc:   []int{3, 0, 1, 0, 1, 1, 0},
		reapparition:   []int{1, 2, 3},
		score:          []int{1, 7, 0},
		duree:          []int{1, 7, 3},
		prolongation:   []int{1, 7, 9},
		chanceActive:   []int{1, 7, 10},
		derniereChance: []int{1, 7, 11},
		chargement:     []int{1, 6, 1, 0},
		arme:           []int{1, 6, 1, 0, 1, jokerDeChemin, 0, 0},
		aleatoire:      []int{1, 6, 1, 0, 1, jokerDeChemin, 2},
		equipement:     []int{1, 6, 1, 0, 2, 0, 0},
		optionNom:      []int{3, 9, 0, jokerDeChemin, 0, 0},
		optionValeur:   []int{3, 9, 0, jokerDeChemin, 1, 0},
	}
	m.suivis = [][]int{m.chaines, m.decalageNom, m.decalageDesc, m.reapparition, m.score, m.duree,
		m.prolongation, m.chanceActive, m.derniereChance, m.chargement, m.arme, m.aleatoire, m.equipement,
		m.optionNom, m.optionValeur}
	return m
}

// lectureDesInfos est la visite qui range les valeurs suivies dans les infos de match.
type lectureDesInfos struct {
	m            modelesDeChemins
	infos        *profile.InfosDeMatch
	variantes    int
	chaines      []byte
	decalageNom  *int64
	decalageDesc *int64
	hachage      *uint32
	chanceActive bool
	chance       *int64
	dotation     *profile.DotationDeDepart
}

// suivre dit si le chemin est sur un modele suivi. Le chemin complet est [champ, rang de variante,
// chemin relatif...] : le champ doit etre une liste de variantes, et une seule variante est suivie.
func (a *lectureDesInfos) suivre(chemin []int) bool {
	switch len(chemin) {
	case 0:
		return true
	case 1:
		return chemin[0] >= 0 && chemin[0] <= dernierChampListeDeVariantes
	case profondeurDeChemin:
		a.variantes++
		return true
	}
	rel := chemin[profondeurDeChemin:]
	if cheminEgal(rel, a.m.chargement) && a.dotation == nil {
		a.dotation = &profile.DotationDeDepart{}
	}
	return slices.ContainsFunc(a.m.suivis, func(m []int) bool { return cheminAvecJokers(rel, m) })
}

// feuille range une valeur suivie.
func (a *lectureDesInfos) feuille(chemin []int, e int64, octets []byte) {
	if len(chemin) <= profondeurDeChemin {
		return
	}
	rel := chemin[profondeurDeChemin:]
	switch {
	case cheminEgal(rel, a.m.chaines):
		a.chaines = octets
	case cheminEgal(rel, a.m.decalageNom):
		a.decalageNom = new(e)
	case cheminEgal(rel, a.m.decalageDesc):
		a.decalageDesc = new(e)
	case cheminEgal(rel, a.m.reapparition):
		a.infos.DelaiDeReapparitionSecondes = new(e)
	case cheminEgal(rel, a.m.score):
		a.infos.ScoreDeVictoire = new(e)
	case cheminEgal(rel, a.m.duree):
		a.infos.DureeSecondes = new(e)
	case cheminEgal(rel, a.m.prolongation):
		a.infos.ProlongationSecondes = new(e)
	case cheminEgal(rel, a.m.chanceActive):
		a.chanceActive = e == 1
	case cheminEgal(rel, a.m.derniereChance):
		a.chance = new(e)
	default:
		a.feuilleDeDotationOuDOption(rel, e, octets)
	}
}

// feuilleDeDotationOuDOption range les valeurs du premier chargement et des options.
func (a *lectureDesInfos) feuilleDeDotationOuDOption(rel []int, e int64, octets []byte) {
	switch {
	case cheminEgal(rel, a.m.arme):
		a.dotation.Armes = append(a.dotation.Armes, uint32(e))
	case cheminEgal(rel, a.m.aleatoire):
		a.dotation.Aleatoire = a.dotation.Aleatoire || e == 1
	case cheminEgal(rel, a.m.equipement):
		a.dotation.Equipement = new(uint32(e))
	case cheminEgal(rel, a.m.optionNom):
		a.hachage = new(uint32(e))
	case cheminEgal(rel, a.m.optionValeur):
		a.ajouterUneOption(octets)
	}
}

// ajouterUneOption pose l option dont le hachage vient d etre lu ; un genre inconnu ou une valeur de
// longueur inattendue n est pas rendu.
func (a *lectureDesInfos) ajouterUneOption(octets []byte) {
	h := a.hachage
	a.hachage = nil
	if h == nil || len(octets) != octetsDeValeurDOption {
		return
	}
	o := profile.OptionDeVariante{Hachage: *h, Genre: octets[octetGenreDOption]}
	switch o.Genre {
	case profile.OptionBooleenne:
		o.Valeur = float64(min(octets[0], 1))
	case profile.OptionNombre:
		o.Valeur = float64(math.Float32frombits(uint32(octets[0]) | uint32(octets[1])<<8 |
			uint32(octets[2])<<16 | uint32(octets[3])<<24))
	default:
		return
	}
	a.infos.Options = append(a.infos.Options, o)
}

// texteADecalage rend la chaine qui commence au decalage `decalage` du bloc de chaines, jusqu a son octet
// nul ; nil quand le decalage manque ou sort du bloc, que la chaine n est pas finie par un octet nul, est
// vide ou n est pas de l UTF-8.
func texteADecalage(bloc []byte, decalage *int64) *string {
	if decalage == nil || *decalage < 0 || *decalage > int64(len(bloc)) {
		return nil
	}
	reste := bloc[*decalage:]
	n := bytes.IndexByte(reste, 0)
	if n <= 0 || !utf8.Valid(reste[:n]) {
		return nil
	}
	return new(string(reste[:n]))
}

// lireLaVarianteDesInfos lit la variante de partie depuis son premier bit et rend ce qu elle declare ; rien
// n est rendu (VarianteLue faux) quand la lecture tombe ou que le message porte plus d une variante.
func lireLaVarianteDesInfos(l *lecteurDuCorps) profile.InfosDeMatch {
	a := &lectureDesInfos{m: nouveauxModelesDeChemins(), infos: &profile.InfosDeMatch{}}
	l.valeurBond(bondStructure, nil, visiteBond{suivre: a.suivre, feuille: a.feuille}, 0)
	if !l.ok || a.variantes != 1 {
		return profile.InfosDeMatch{}
	}
	a.infos.VarianteLue = true
	a.infos.Nom = texteADecalage(a.chaines, a.decalageNom)
	a.infos.Description = texteADecalage(a.chaines, a.decalageDesc)
	if a.chanceActive {
		a.infos.DerniereChanceSecondes = a.chance
	}
	a.infos.Dotation = a.dotation
	return *a.infos
}

// lireLesInfosDeMatch lit les infos de match d un `chunk_00` dont l identite `id` est lue : l heure de
// debut et les identifiants de la variante de carte, puis la variante de partie quand la marche du corps
// l a lue en entier.
func lireLesInfosDeMatch(chunk0 []byte, id profile.FilmIdentity) profile.InfosDeMatch {
	var infos profile.InfosDeMatch
	if id.MatchStartUnix != 0 {
		infos.DebutUnix = new(id.MatchStartUnix)
	}
	infos.CarteAssetID = lireUnGUID(chunk0, id.BodyBit-2*8*octetsDuGUID)
	infos.CarteVersionID = lireUnGUID(chunk0, id.BodyBit-8*octetsDuGUID)
	if !id.Variante.Lue || !id.Variante.Presente {
		return infos
	}
	l := &lecteurDuCorps{br: LecteurSur(chunk0), ok: id.BodyBit > 0 && id.BodyBit < len(chunk0)*8}
	if !l.ok || !l.jusquALaVariante(id.BodyBit) {
		return infos
	}
	variante := lireLaVarianteDesInfos(l)
	variante.DebutUnix, variante.CarteAssetID, variante.CarteVersionID =
		infos.DebutUnix, infos.CarteAssetID, infos.CarteVersionID
	return variante
}

// lireUnGUID lit les 16 octets qui commencent au bit `bit` ; nil hors du tampon ou quand ils sont tous
// a zero.
func lireUnGUID(chunk0 []byte, bit int) *profile.GUID {
	if !tientDansLeTampon(chunk0, bit, 8*octetsDuGUID) {
		return nil
	}
	br := LecteurSur(chunk0)
	br.SetBitPos(bit)
	var g profile.GUID
	for i := range g {
		g[i] = byte(br.ReadBits(8))
	}
	if g == (profile.GUID{}) {
		return nil
	}
	return &g
}
