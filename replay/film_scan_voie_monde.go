package replay

// film_scan_voie_monde.go — LA VOIE DU MONDE : LES LECTURES DES OBJETS DU MONDE ET DES LANCERS.
//
// Elle lit, dans l ordre de leur dependance et sur son contexte derive : la lunette, les poses
// d equipement, les evenements de piece engendree, les socles, les vehicules, puis les lancers de grenade,
// les projectiles et les detonations. Les socles et les vehicules heritent des largeurs MPP que la
// calibration des poses vient de mesurer : la voie les prend dans la lecture des poses qu elle vient de
// ranger, qui porte les memes statistiques que celles que le fil principal publie (`journaliserPoses`
// les rend telles quelles).
//
// LA VOIE EST L UNIQUE PROPRIETAIRE des memos des pistes et des creations du monde : la lecture des poses
// les releve pour tous les archetypes, les socles, les vehicules et les projectiles les trouvent chauds,
// et le fil principal ne les demande jamais (la fusion les reprend a la fin ; cf. la limite du contexte de
// voie dans `grammar/film_context_voie.go`). Le repli « apres la marche » de chacun se compte donc une seule
// fois, dans les replis de la voie.
//
// Chaque lecture est rendue a SON rang de publication : lunette, poses, pieces, socles et vehicules par
// `balayerMonde`, lancers, projectiles et detonations par `balayerPont`. Les `lire…` de la chaine attendent la valeur
// de la voie ici ([attendreVoie]) ; les lectures elles-memes sont les memes fonctions que celles de la chaine
// en serie (`lireLaLunette`, `decodeFilm…Lecture`, `lireLesGrenades`, `lireLesProjectiles`,
// `lireLesDetonations`).

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// voieDuMonde est la voie du monde : ses valeurs, une par lecture, chacune prete a son tour.
type voieDuMonde struct {
	*voie
	lunette     *attendue[[]grammar.ZoomEvent]
	poses       *attendue[lecturePoses]
	pieces      *attendue[lectureEvenementsPiece]
	socles      *attendue[lecturePads]
	vehicules   *attendue[lectureVehicules]
	grenades    *attendue[lectureGrenades]
	projectiles *attendue[lectureProjectiles]
	detonations *attendue[[]types.ProjectileDetonation]
	// largeurs : les largeurs MPP que la calibration des poses a mesurees, ecrites par la voie avant de rendre les
	// poses ; les socles et les vehicules les lisent. Un test les compare a celles que le fil principal publie.
	largeurs profile.MPPWidths
}

// lancerLaVoieDuMonde ouvre la voie du monde sur `fc` (contexte derive, qu elle seule touche) et la lance.
// `world` est une copie des bornes de la carte : la voie ne partage rien avec la chaine.
func lancerLaVoieDuMonde(fc *grammar.FilmContext, world profile.Vec3Range) *voieDuMonde {
	l := &voie{fc: fc}
	m := &voieDuMonde{voie: l,
		lunette:     nouvelleAttendue[[]grammar.ZoomEvent](l),
		poses:       nouvelleAttendue[lecturePoses](l),
		pieces:      nouvelleAttendue[lectureEvenementsPiece](l),
		socles:      nouvelleAttendue[lecturePads](l),
		vehicules:   nouvelleAttendue[lectureVehicules](l),
		grenades:    nouvelleAttendue[lectureGrenades](l),
		projectiles: nouvelleAttendue[lectureProjectiles](l),
		detonations: nouvelleAttendue[[]types.ProjectileDetonation](l),
	}
	l.lancer(func() { m.lire(&world) })
	return m
}

// lire enchaine les lectures du monde, dans l ordre de leurs dependances. Fil de la voie. La voie s arrete
// entre deux lectures quand le balayage la lui demande.
func (m *voieDuMonde) lire(wr *profile.Vec3Range) {
	fc := m.fc
	pas := []func(){
		func() { livrer(m.lunette, lireLaLunette(fc)) },
		func() {
			p := decodeFilmPlacementsLecture(fc, wr)
			m.largeurs = p.stats.Calibration.Widths
			livrer(m.poses, p)
		},
		func() { livrer(m.pieces, decodeFilmSpawnEventsLecture(fc)) },
		func() { livrer(m.socles, decodeFilmPadScansLecture(fc, wr, m.largeurs)) },
		func() { livrer(m.vehicules, decodeFilmVehicleScanLecture(fc, wr, m.largeurs)) },
		func() { livrer(m.grenades, lireLesGrenades(fc)) },
		func() { livrer(m.projectiles, lireLesProjectiles(fc, wr)) },
		func() { livrer(m.detonations, lireLesDetonations(fc)) },
	}
	for _, p := range pas {
		if m.arretee() {
			return
		}
		p()
	}
}

// Les accesseurs du fil principal : la valeur de la voie a son rang de publication, et faux sans voie (la
// lecture se fait alors en serie, dans la chaine). Un pointeur nil est l absence de voie.

func (v *voiesDuBalayage) lunette() ([]grammar.ZoomEvent, bool) {
	if v == nil {
		return nil, false
	}
	return attendreVoie(v, v.monde.lunette), true
}

func (v *voiesDuBalayage) poses() (lecturePoses, bool) {
	if v == nil {
		return lecturePoses{}, false
	}
	return attendreVoie(v, v.monde.poses), true
}

func (v *voiesDuBalayage) pieces() (lectureEvenementsPiece, bool) {
	if v == nil {
		return lectureEvenementsPiece{}, false
	}
	return attendreVoie(v, v.monde.pieces), true
}

func (v *voiesDuBalayage) socles() (lecturePads, bool) {
	if v == nil {
		return lecturePads{}, false
	}
	return attendreVoie(v, v.monde.socles), true
}

func (v *voiesDuBalayage) vehicules() (lectureVehicules, bool) {
	if v == nil {
		return lectureVehicules{}, false
	}
	return attendreVoie(v, v.monde.vehicules), true
}

func (v *voiesDuBalayage) grenades() (lectureGrenades, bool) {
	if v == nil {
		return lectureGrenades{}, false
	}
	return attendreVoie(v, v.monde.grenades), true
}

func (v *voiesDuBalayage) projectiles() (lectureProjectiles, bool) {
	if v == nil {
		return lectureProjectiles{}, false
	}
	return attendreVoie(v, v.monde.projectiles), true
}

func (v *voiesDuBalayage) detonations() ([]types.ProjectileDetonation, bool) {
	if v == nil {
		return nil, false
	}
	return attendreVoie(v, v.monde.detonations), true
}
