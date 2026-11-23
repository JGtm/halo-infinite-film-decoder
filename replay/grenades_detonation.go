package replay

// grenades_detonation.go — L'EXPLOSION D'UN LANCER DONT LE VOL N'EST PAS PUBLIÉ.
//
// CE QUI MANQUAIT. L'explosion d'une grenade se publie sur son projectile (`Projectile.DetoT/X/Y`,
// cf. [attachDetonation]), et le lancer y renvoie par `Grenade.Proj`. Un projectile qui colle à bout
// portant ne réplique pas trois positions avant de suivre son parent : aucun vol n'est publié, le
// lancer n'a pas de `Proj`, et son explosion — que le film écrit pourtant — n'avait aucun lieu où
// se publier. Ce fichier la pose sur le lancer lui-même, `Grenade.Deto`, et seulement là : une
// explosion ne se publie qu'une fois, sur le projectile quand il y en a un.
//
// LE LIEN EST LE HANDLE DU PROJECTILE, PAS UNE PROXIMITÉ. Le lancer est le record NEW du projectile
// et en porte le handle ([grammar.GrenadeThrow.Life]) ; le message de détonation nomme la même vie,
// en domaine 5 de la table des références — plage 0x100, base 0x400 (`varwidth.go`) —, donc le slot
// du projectile vaut la référence du message plus [detonationHandleBase], à génération égale.
//
// CE QUE LA MESURE DIT DE CE LIEN. Sur les lancers que leur handle lie à une piste, la détonation du
// handle est celle que le rattachement par proximité de la tête ([detonationLaPlusProche]) choisit
// pour la même piste ; les messages qui nomment un handle de lancer sont `projectile_detonate` seul,
// ou `projectile_impact_effect` puis `projectile_detonate` pour la grenade à pointes, qui colle puis
// explose 1,2 s plus tard. D'où le choix du GENRE : seul `projectile_detonate` est l'explosion,
// l'effet d'impact de la grenade à pointes n'en est que le collage.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// GrenadeDetonation est l'explosion d'un lancer dont le vol n'est pas publié : le message de
// détonation du projectile que le lancer a créé. T est l'index de frame, sur l'axe de Point.T
// (absolu, comme Projectile.DetoT) ; X, Y, Z la position, mêmes axes que les pistes, Z la hauteur.
type GrenadeDetonation struct {
	T int     `json:"t"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// detonationHandleBase est ce que la référence de projectile d'un message de détonation omet du
// slot du projectile : la base de la catégorie 5 de la table des références (`varwidth.go`, 0x400).
const detonationHandleBase = 0x400

// genreDetonation est le genre de la vue A du message `projectile_detonate` : l'explosion. Le genre 6,
// `projectile_impact_effect`, est l'effet d'un impact — pour la grenade à pointes, le collage, 1,2 s
// avant l'explosion.
const genreDetonation uint8 = 5

// grenadeDetonationWindowUS borne la recherche après le lancer : la détonation du handle la plus
// tardive vue sur un lancer lié, plus une marge. Il faut une borne parce que le pool de projectiles
// rend le même handle (le slot, et une génération sur deux bits) après la fin d'une vie : sans elle,
// un lancer dont l'explosion n'a pas été lue se lierait à celle d'un projectile postérieur.
//
// MESURE QUI LA FONDE (7 films de 7 cartes, builds HI_1_8_0 à HI_1_13_0, environ 1 200 lancers liés à
// une piste par leur handle ; chiffres exacts dans le compte rendu du lot). Premier message
// `projectile_detonate` du handle après le lancer : médiane 1,7 s, 99e centile 4,6 s, maximum 6,77 s
// (une fragmentation, sur une carte) ; la grenade à retardement fixe (Dynamo) explose à 3,1 s. Aucun
// lancer ne porte deux `projectile_detonate` du même handle dans les 30 s : la borne n'a pas à
// départager des explosions, seulement à écarter la réutilisation du handle, observée à 869 s sur un
// film. 8 s laisse 1,2 s au-dessus du plus long délai mesuré.
const grenadeDetonationWindowUS uint64 = 8_000_000

// detonationOfThrow rend la première détonation à position qui nomme le handle du projectile créé par
// le lancer `th`, dans [grenadeDetonationWindowUS] après lui. Faux quand le lancer n'a pas de handle
// lu ou que le film n'écrit aucune détonation pour ce projectile.
func detonationOfThrow(th grammar.GrenadeThrow, detos []types.ProjectileDetonation) (types.ProjectileDetonation, bool) {
	var best types.ProjectileDetonation
	if !th.HasLife {
		return best, false
	}
	found := false
	for _, d := range detos {
		if d.Genre != genreDetonation || !d.HasProjectile || !d.HasPosition ||
			d.Projectile.Gen != th.Life.Gen || d.Projectile.Slot+detonationHandleBase != th.Life.Slot ||
			d.TimestampUS < th.TimestampUS || d.TimestampUS-th.TimestampUS > grenadeDetonationWindowUS {
			continue
		}
		if !found || d.TimestampUS < best.TimestampUS {
			best, found = d, true
		}
	}
	return best, found
}

// grenadeDetonation est la forme publiée de [detonationOfThrow] sur la grille du document, nil quand
// le film n'écrit aucune détonation pour le projectile du lancer, ou qu'elle précède l'origine de l'axe.
func grenadeDetonation(th grammar.GrenadeThrow, detos []types.ProjectileDetonation, origin, step uint64) *GrenadeDetonation {
	d, ok := detonationOfThrow(th, detos)
	if !ok || d.TimestampUS < origin {
		return nil
	}
	return &GrenadeDetonation{
		T: int((d.TimestampUS - origin) / step), //nolint:gosec // index de frame d'un match
		X: round2(d.X), Y: round2(d.Y), Z: round2(d.Z),
	}
}
