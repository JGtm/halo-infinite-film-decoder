package replay

// options_vitesse.go — LE PORTEUR UNIQUE DES OPTIONS DE VITESSE DU DECODEUR.
//
// Toute option de vitesse (nombre de fils, plus tard le choix des canaux) vit dans OptionsVitesse :
// aucune piste de vitesse ne cree son propre reglage. Ce fichier ne fait que PORTER et NORMALISER
// la valeur demandee et la transmettre au contexte du film ; le document produit ne
// depend pas d'elle.

import (
	"runtime"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// MaxFils est la borne ecrite dans la brique du nombre de fils : aucune option ne depasse cette
// valeur, quelle que soit la machine.
const MaxFils = 8

// OptionsVitesse regle la vitesse du decodage sans en changer le resultat.
//
// La valeur zero est valide et vaut le comportement actuel : un seul fil.
type OptionsVitesse struct {
	// Fils : nombre de fils que le decodeur a le droit d'employer. Borne reelle : a un fil, ou 0 ou
	// negatif, aucune goroutine n'est lancee ; a `Fils` >= 2, au plus `Fils`+3 goroutines calculent en meme
	// temps (le pre-passage des images-cles lance `Fils` ouvriers pendant que la tache de fond et les deux
	// voies du balayage calculent ; la chaine principale attend alors ses ouvriers). Les trois points de
	// lancement sont comptes par `TestFilsBorneReelleDesFilsQuiCalculent`. Le nombre reellement employe
	// est FilsEffectifs. La sortie ne depend pas de cette valeur.
	Fils int
}

// FilsEffectifs rend le nombre de fils que le decodeur a le droit d'employer : le minimum de
// l'option, de runtime.GOMAXPROCS(0) et de MaxFils, et au moins 1. Une option a zero ou negative
// donne 1. La fonction est pure : meme option et meme GOMAXPROCS, meme resultat.
func (o OptionsVitesse) FilsEffectifs() int {
	return filsEffectifs(o.Fils, runtime.GOMAXPROCS(0))
}

// filsEffectifs normalise une valeur demandee, `gomaxprocs` etant passe en parametre pour rester
// testable sans changer le reglage du processus.
func filsEffectifs(demande, gomaxprocs int) int {
	n := demande
	if n > gomaxprocs {
		n = gomaxprocs
	}
	if n > MaxFils {
		n = MaxFils
	}
	if n < 1 {
		return 1
	}
	return n
}

// poserLesFils transmet au contexte du film le nombre de fils que l'hote demande. Une demande a 0
// (ou negative) ne touche pas le contexte : un contexte partage que l'hote a deja regle garde son
// reglage. Une demande positive est normalisee par [OptionsVitesse.FilsEffectifs].
func poserLesFils(fc *grammar.FilmContext, v OptionsVitesse) {
	if v.Fils > 0 {
		fc.PoserFils(v.FilsEffectifs())
	}
}
