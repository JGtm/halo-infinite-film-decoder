package replay

// projectiles_plein_debit.go — LE VOL D UN PROJECTILE A PLEINE CADENCE, ET SA HAUTEUR.
//
// `Projectile.P` ne porte que [dt, x, y], un point par frame de la grille. La 3D a besoin de deux
// choses de plus, que l echantillon decode portait deja (le composant de position quantifie les trois
// axes) :
//
//   - la HAUTEUR de chaque point de `P` : `Projectile.Z`, un tableau PARALLELE (meme indice, meme
//     arrondi au centieme que les points de piste), et non un quatrieme element de `P` : tout lecteur
//     de [dt, x, y] reste valide. Sans elle un lecteur ne sait que faire glisser une grenade en ligne
//     droite jusqu a son point d arrivee, et perd l arc et chaque rebond ;
//   - le vol AU DEBIT DU FILM : `Projectile.FullRate`, chaque echantillon repliquee a son instant, du
//     meme codage que `Track.FullRate` (cf. tracks_plein_debit.go). Un projectile suit une parabole
//     repliquee a ~60 Hz ; la grille n en retient qu un point par frame.
//
// LES DEUX SE PUBLIENT SUR TOUT DOCUMENT, et ne changent aucune des valeurs existantes : `P`, `T0`,
// `Rest`, la detonation et le lien vers le tir restent ceux de la decimation. Seul le document interne
// du sync, qui n est pas publie, ne les calcule pas.
//
// LA COUPURE EST CELLE DU VOL PUBLIE : la serie s arrete au premier pas impossible (cf.
// [projectileMaxSpeedMPS]), comme `P` ; un echantillon lu apres la coupure n est jamais publie.

import "github.com/JGtm/halo-infinite-film-decoder/types"

// FullRateFlight porte, pour UNE trajectoire, tous les echantillons que le film a repliquees avant
// le premier pas impossible, chacun a SON instant : le jumeau de [FullRateTrack], sans visee (un
// projectile n en porte pas).
//
//	t        instants en MILLISECONDES depuis l ORIGINE DE LA GRILLE, celle de `Point.T` (tronques,
//	         cf. [fullRateTimeUnitUS]) : meme unite que `FullRateTrack.T`
//	x, y, z  centiemes d unite monde, entiers : les MEMES valeurs que les points de `P` et `Z`
//
// Les quatre colonnes ont la meme longueur, et chacune est differentielle (la premiere valeur est
// absolue, les suivantes l ecart a la precedente).
type FullRateFlight struct {
	T []int64 `json:"t"`
	X []int32 `json:"x"`
	Y []int32 `json:"y"`
	Z []int32 `json:"z"`
}

// pleinDebitDeVol accumule la serie d un vol et en tient les derniers absolus, pour ecrire les
// ecarts. Un recepteur nil est valide et ne retient rien : c est le document interne du sync.
type pleinDebitDeVol struct {
	f       FullRateFlight
	t       int64
	x, y, z int32
}

// ajouter pose un echantillon a son instant.
func (d *pleinDebitDeVol) ajouter(p types.ProjectileSample, origineUS uint64) {
	if d == nil {
		return
	}
	t := int64((p.TimestampUS - origineUS) / fullRateTimeUnitUS)
	x, y, z := centiemes(p.X), centiemes(p.Y), centiemes(p.Z)
	d.f.T = append(d.f.T, t-d.t)
	d.f.X = append(d.f.X, x-d.x)
	d.f.Y = append(d.f.Y, y-d.y)
	d.f.Z = append(d.f.Z, z-d.z)
	d.t, d.x, d.y, d.z = t, x, y, z
}

// rendre rend la serie, nil quand le vol n a recu aucun echantillon (ou que c est le document interne du sync).
func (d *pleinDebitDeVol) rendre() *FullRateFlight {
	if d == nil || len(d.f.T) == 0 {
		return nil
	}
	out := d.f
	return &out
}

// poserLaPleineCadenceDesProjectiles pose la hauteur et le vol au debit du film sur les projectiles
// publies, et rend le nombre d echantillons de pleine cadence posees.
//
// ELLE REJOUE [volPublie] SUR LES PISTES PUBLIEES, avec la serie demandee, plutot que de la faire
// tourner dans [buildProjectiles] : la regle du vol (ou il commence, ou il se coupe) est UNIQUE et vit
// dans `volPublie`, et la construction des projectiles, appelee de partout (tests compris), garde ses cinq
// parametres, que le depot borne ; le document interne du sync ne l appelle pas. `publiesParPiste` est la table piste brute -> rang publie que
// `buildProjectiles` rend ; les pistes sont parcourues dans l ordre, donc la sortie ne depend pas de
// l ordre d une table.
func poserLaPleineCadenceDesProjectiles(publies []Projectile, pistes []types.ProjectileTrack,
	publiesParPiste map[int]int, origine, pas uint64) int {
	echantillons := 0
	for brut, tr := range pistes {
		rang, ok := publiesParPiste[brut]
		if !ok {
			continue
		}
		v, lu := volPublie(tr, origine, pas, true)
		if !lu || len(v.z) != len(publies[rang].P) {
			// Inatteignable : la piste est publiee, donc son vol est lu et ses points sont ceux de la
			// decimation. Rien ne se pose plutot qu une hauteur decalee d un point.
			continue
		}
		publies[rang].Z = v.z
		publies[rang].FullRate = v.plein.rendre()
		echantillons += len(v.plein.f.T)
	}
	return echantillons
}
