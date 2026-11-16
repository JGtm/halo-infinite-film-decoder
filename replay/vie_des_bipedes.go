package replay

// vie_des_bipedes.go — LA VIE D'UN BIPEDE, RELUE A UN INSTANT.
//
// Le film écrit la santé et le bouclier d'un bipède dans ses deux composants de vitalité, quand l'un d'eux change
// ([grammar.LectureVitale]) ; une vie commence pleine. Ce fichier range ces lectures par vie (le slot et la génération)
// en états COMPLETS, pour dire ce que valait la vie juste avant un instant ou à cet instant : une lecture partielle
// garde la valeur de l'autre composant, et rien n'est supposé qui ne soit écrit (pas de régénération par délai).

import (
	"math"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// cleDeVie identifie la vie d'un bipède : son slot dans la table d'entités et la génération du slot.
type cleDeVie struct {
	slot uint32
	gen  uint8
}

// etatVital est la vie d'un bipède à un instant : la santé de -1 à 1, le bouclier de 0 à 4 (1 est plein).
type etatVital struct {
	ts       uint64
	sante    float64
	bouclier float64
}

// vieDUnBiped est la suite, par instant croissant, des états que le film écrit pour un bipède.
type vieDUnBiped []etatVital

// registreDesVies range les lectures du film par vie.
type registreDesVies map[cleDeVie]vieDUnBiped

// registreDesViesDe construit le registre : une vie naît pleine, une lecture partielle complète l'état précédent.
func registreDesViesDe(lectures []grammar.LectureVitale) registreDesVies {
	reg := registreDesVies{}
	for _, l := range lectures {
		k := cleDeVie{l.Slot, l.Gen}
		cur := etatVital{sante: 1, bouclier: 1}
		if suite := reg[k]; len(suite) > 0 && !l.Creation {
			cur = suite[len(suite)-1]
		}
		cur.ts = l.TS
		if l.ASante {
			cur.sante = float64(l.Sante)
		}
		if l.ABouclier {
			cur.bouclier = float64(l.Bouclier)
		}
		reg[k] = append(reg[k], cur)
	}
	return reg
}

// vieDeLUnite rend la vie du bipède de l'unité `unite` (index d'unité du message de dégât) de génération `gen`.
func (r registreDesVies) vieDeLUnite(unite uint16, gen uint8) vieDUnBiped {
	return r[cleDeVie{uint32(unite) + decalageDesSlotsDUnite, gen}]
}

// etat rend l'état de la vie à `ts` : le dernier écrit à `ts` ou avant (`inclus`), ou le dernier écrit strictement avant.
// Une vie sans lecture est pleine ; une santé négative se replie sur zéro.
func (v vieDUnBiped) etat(ts uint64, inclus bool) (sante, bouclier float64) {
	n := sort.Search(len(v), func(i int) bool {
		if inclus {
			return v[i].ts > ts
		}
		return v[i].ts >= ts
	})
	if n == 0 {
		return 1, 1
	}
	return math.Max(v[n-1].sante, 0), v[n-1].bouclier
}

// avant rend la vie juste avant `ts` ; a rend la vie à `ts`, les écritures de l'instant comprises.
func (v vieDUnBiped) avant(ts uint64) (sante, bouclier float64) { return v.etat(ts, false) }
func (v vieDUnBiped) a(ts uint64) (sante, bouclier float64)     { return v.etat(ts, true) }
