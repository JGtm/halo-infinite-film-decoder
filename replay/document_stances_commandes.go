package replay

// document_stances_commandes.go — LES COMMANDES TENUES, PUBLIEES COMME DES ETATS DE MOUVEMENT : le deplacement,
// l accroupi et le saut, lus dans la vue de controle (`stances[]`, genres `moveCommand`, `crouchCommand`,
// `jumpCommand`).
//
// CE QUE LA LECTURE DONNE. La grammaire rend, par place et par genre, l intervalle ou la commande est tenue :
// de la premiere entree lue qui la pose a la premiere entree lue qui ne la pose plus, ses trous portes
// (`grammar/commandes_de_controle.go`). La place est l index de controle de l entree : l occupant de la place a
// cet instant est le joueur, et une vie du joueur porte l intervalle ([pontDesPlaces.vieDeLaPlace]).
//
// CE QU ON EN FAIT. Un intervalle lu se coupe aux limites de chaque vie publiee dont le corps est celui de
// l occupant, et se tait sur les passages que la lecture n a pas atteints (les trous, comme pour le tir
// continu : une frame n est muette que si un trou la couvre entiere). Une commande lue dont aucune vie
// publiee ne recouvre l intervalle, ou dont aucune des vies qui le recouvrent n est celle de l occupant, n est
// pas publiee et se compte. Rien n est devine : jamais de serie brute des valeurs, jamais un intervalle sans
// la lecture qui le borne.
//
// LES GENRES NEUFS ONT DES CONDITIONS DE PLUS ([regleDeGenre]), parce qu ils disent ce que le JOUEUR demande et
// que la lecture n en vaut que sous trois conditions :
//
//	la porte de cadence   le film enregistre les commandes a pleine cadence ([types.ControlHoldStats.PleineCadence]) :
//	                      un appui d un tick n est vu qu une fois sur deux a la moitie de la cadence. Sous la
//	                      cadence, ni accroupi ni saut ne se publient, et chaque commande lue se compte (`gated`) ;
//	le vehicule           rien ne se pose pendant un episode de vehicule de l occupant (`vehicles[].rides`) : le
//	                      bouton n y commande pas le meme geste (`inVehicle`) ;
//	le debut du saut      l instant d un saut est son debut : une commande dont le debut suit un trou de lecture
//	                      n a pas de debut connu, et se compte (`afterHole`). L accroupi est un intervalle : le
//	                      debut apres un trou est la premiere entree qui le lit, le trou reste muet.
//
// Les genres neufs portent en plus `ms0` et `ms1` : l intervalle a la milliseconde depuis la frame 0, sans deborder
// ni l intervalle lu ni les frames publiees. Un saut d un tick dure moins qu une frame du document : l instant du
// bouton est `ms0`.

import (
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// StanceControlCoverage dit ce que la lecture des commandes tenues a rendu et ce que la publication en a fait.
//
// L IDENTITE : holds = published + noTrack + noPlayer. Une commande publiee donne un intervalle par vie et par
// passage lu ; `intervals` les compte, et `coverage.stances.byKind` les ventile. Ces compteurs sont ceux de la
// commande de deplacement ; l accroupi et le saut ont chacun les leurs ([ControlKindCoverage]).
type StanceControlCoverage struct {
	// Holds : les commandes tenues lues ; WithHole : celles qu un trou touche ; InnerHoles : leurs passages
	// interieurs muets.
	Holds      int `json:"holds"`
	WithHole   int `json:"withHole"`
	InnerHoles int `json:"innerHoles"`
	// Entries : les entrees de controle lues qui portent le bloc de commande.
	Entries int `json:"entries"`
	// Published : les commandes qui donnent au moins un intervalle ; Intervals : les intervalles publies.
	Published int `json:"published"`
	Intervals int `json:"intervals"`
	// NoTrack : les commandes qu aucune vie publiee ne recouvre ; NoPlayer : celles que recouvrent des vies
	// dont aucune n est celle de l occupant de la place.
	NoTrack  int `json:"noTrack"`
	NoPlayer int `json:"noPlayer"`
	// StepUS : le pas des entrees de controle du film, en microsecondes (l ecart le plus frequent entre deux
	// entrees lues d une meme place, sans trou entre elles ; zero quand aucun ecart n est mesure). FullCadence :
	// ce pas est mesurable et ne depasse pas `types.PasDeCommande60HzMaxUS` — la porte de l accroupi et du saut.
	StepUS      int64 `json:"stepUs"`
	FullCadence bool  `json:"fullCadence"`
	// CrouchCommand et JumpCommand : le sort des commandes d accroupi et de saut lues.
	CrouchCommand ControlKindCoverage `json:"crouchCommand"`
	JumpCommand   ControlKindCoverage `json:"jumpCommand"`
}

// ControlKindCoverage dit ce que la lecture a rendu d un genre de commande et ce que la publication en a fait.
//
// L IDENTITE : holds = published + noTrack + noPlayer + inVehicle + afterHole + gated. Chaque commande lue a UN
// sort, dans cet ordre : `gated` (film sous la cadence), `afterHole` (saut : debut apres un trou), puis, vie par
// vie, `published` (au moins un intervalle), `inVehicle` (les passages de l occupant que la lecture couvre sont
// tous en vehicule), `noPlayer` (aucune des vies qui la recouvrent n est celle de l occupant, ou rien n y est
// lu), `noTrack` (aucune vie publiee ne la recouvre).
type ControlKindCoverage struct {
	// Holds : les commandes tenues lues ; WithHole : celles qu un trou touche ; InnerHoles : leurs passages
	// interieurs muets.
	Holds      int `json:"holds"`
	WithHole   int `json:"withHole"`
	InnerHoles int `json:"innerHoles"`
	// Published : les commandes qui donnent au moins un intervalle ; Intervals : les intervalles publies.
	Published int `json:"published"`
	Intervals int `json:"intervals"`
	// Les commandes lues ecartees, par raison.
	NoTrack   int `json:"noTrack"`
	NoPlayer  int `json:"noPlayer"`
	InVehicle int `json:"inVehicle"`
	AfterHole int `json:"afterHole"`
	Gated     int `json:"gated"`
}

// regleDeGenre dit ce que la publication exige d un genre de commande en plus de la lecture.
type regleDeGenre struct {
	// porte : le film doit etre a pleine cadence.
	porte bool
	// sansVehicule : rien ne se pose pendant un episode de vehicule de l occupant.
	sansVehicule bool
	// debutConnu : une commande dont le debut suit un trou n est pas publiee.
	debutConnu bool
	// avecMs : l intervalle porte `ms0` et `ms1`.
	avecMs bool
}

// regleDe rend la regle de publication d un genre : le deplacement n en a aucune.
func regleDe(kind string) regleDeGenre {
	switch kind {
	case types.MovementCrouchCommand:
		return regleDeGenre{porte: true, sansVehicule: true, avecMs: true}
	case types.MovementJumpCommand:
		return regleDeGenre{porte: true, sansVehicule: true, debutConnu: true, avecMs: true}
	default:
		return regleDeGenre{}
	}
}

// controlStanceInputs porte ce dont la publication des commandes a besoin.
type controlStanceInputs struct {
	holds    []types.ControlHold
	stats    types.ControlHoldStats
	tracks   []Track
	vehicles []VehicleTrack
	pont     pontDesPlaces
}

// buildControlStances replie les commandes tenues en intervalles par vie et rend leur couverture.
func buildControlStances(in controlStanceInputs) ([]Stance, StanceControlCoverage) {
	cov := StanceControlCoverage{Entries: in.stats.Entries, StepUS: in.stats.StepUS, FullCadence: in.stats.PleineCadence()}
	move := ControlKindCoverage{}
	sorts := in.compteursParGenre(&cov, &move)
	var out []Stance
	if in.pont.clock.step != 0 {
		rides := ridesParSlot(in.vehicles)
		for _, h := range in.holds {
			if c, ok := sorts[h.Kind]; ok { // un genre inconnu de cette version n a pas de compteur : il n est pas lu
				in.poserLaCommande(h, rides, c, &out)
			}
		}
	}
	cov.Holds, cov.WithHole, cov.InnerHoles = move.Holds, move.WithHole, move.InnerHoles
	cov.Published, cov.Intervals, cov.NoTrack, cov.NoPlayer = move.Published, move.Intervals, move.NoTrack, move.NoPlayer
	sortStances(out)
	return out, cov
}

// compteursParGenre rend le compteur de chaque genre, deja garni de ce que la lecture a compte : celui du
// deplacement est `move`, que l appelant recopie dans les champs de tete de la couverture.
func (in controlStanceInputs) compteursParGenre(cov *StanceControlCoverage,
	move *ControlKindCoverage) map[string]*ControlKindCoverage {
	sorts := map[string]*ControlKindCoverage{types.MovementMoveCommand: move,
		types.MovementCrouchCommand: &cov.CrouchCommand, types.MovementJumpCommand: &cov.JumpCommand}
	lus := map[string]types.ControlHoldCount{types.MovementMoveCommand: in.stats.Move,
		types.MovementCrouchCommand: in.stats.Crouch, types.MovementJumpCommand: in.stats.Jump}
	for kind, c := range sorts {
		c.Holds, c.WithHole, c.InnerHoles = lus[kind].Holds, lus[kind].WithHole, lus[kind].InnerHoles
	}
	return sorts
}

// poserLaCommande decide du sort d UNE commande lue, la pose si elle se publie, et la compte.
func (in controlStanceInputs) poserLaCommande(h types.ControlHold, rides map[uint32][][2]int, c *ControlKindCoverage,
	out *[]Stance) {
	regle := regleDe(h.Kind)
	switch {
	case regle.porte && !in.stats.PleineCadence():
		c.Gated++
		return
	case regle.debutConnu && h.StartBound == types.ContinuousFireBoundHole:
		c.AfterHole++
		return
	}
	r := in.publier(h, regle, rides, out)
	switch {
	case r.made > 0:
		c.Published++
		c.Intervals += r.made
	case r.enVehicule:
		c.InVehicle++
	case r.recouvre:
		c.NoPlayer++
	default:
		c.NoTrack++
	}
}

// poseDeLaCommande est ce qu une commande a donne : le nombre d intervalles, si une vie publiee la recouvre, et si
// des passages ont ete retires parce que l occupant etait en vehicule.
type poseDeLaCommande struct {
	made       int
	recouvre   bool
	enVehicule bool
}

// publier pose la commande `h` sur les vies de son occupant et rend ce qu elle a donne.
func (in controlStanceInputs) publier(h types.ControlHold, regle regleDeGenre, rides map[uint32][][2]int,
	out *[]Stance) poseDeLaCommande {
	var r poseDeLaCommande
	clock := in.pont.clock
	f0, f1 := clock.frame(h.StartUS), clock.frame(h.EndUS)
	pieces := in.sansLesTrous(f0, f1, h.Holes)
	for _, tr := range in.tracks {
		a, z := max(f0, tr.StartFrame), min(f1, tr.EndFrame)
		if z < a {
			continue
		}
		r.recouvre = true
		if !in.pont.vieDeLaPlace(tr, h.FilmIndex, a) {
			continue
		}
		for _, p := range pieces {
			t0, t1 := max(p[0], a), min(p[1], z)
			if t0 > t1 {
				continue
			}
			parts := [][2]int{{t0, t1}}
			if regle.sansVehicule {
				parts = sansLesRides(t0, t1, rides[tr.Slot])
				r.enVehicule = r.enVehicule || len(parts) != 1 || parts[0] != [2]int{t0, t1}
			}
			for _, q := range parts {
				*out = append(*out, in.stance(tr.Slot, h, regle, q))
				r.made++
			}
		}
	}
	return r
}

// stance rend l intervalle de frames `q` de la commande `h` sur la vie `slot`.
func (in controlStanceInputs) stance(slot uint32, h types.ControlHold, regle regleDeGenre, q [2]int) Stance {
	s := Stance{Slot: slot, Kind: h.Kind, T0: q[0], T1: q[1]}
	if !regle.avecMs {
		return s
	}
	c := in.pont.clock
	de := max(h.StartUS, instantDeFrame(c, q[0]))
	a := min(h.EndUS, instantDeFrame(c, q[1]+1))
	ms0, ms1 := msDepuisLOrigine(c, de), msDepuisLOrigine(c, max(a, de))
	s.Ms0, s.Ms1 = &ms0, &ms1
	return s
}

// msDepuisLOrigine rend l instant en millisecondes depuis la frame 0 ; zero avant l origine.
func msDepuisLOrigine(c replayClock, us uint64) int64 {
	if us <= c.origin {
		return 0
	}
	return int64((us - c.origin) / 1000) //nolint:gosec // duree d un film, bien sous 2^63 µs
}
