package replay

// vehicle_destruction_killed.go — QUI EST MORT AVEC UN VEHICULE DETRUIT, LU DANS LE FILM.
//
// LE FILM ECRIT LE LIEN, SANS FENETRE DE TEMPS NI DISTANCE. Un occupant tue avec son vehicule ecrit
// SON PROPRE dead-state (`object-dead-state`, meme composant que celui du vehicule), et son etat de
// parente (`object-parent-state`, une transition : rattache a un parent, ou libre) dit de quel
// vehicule il etait rattache. Le dead-state d un bipede porte l index de la VICTIME (`EnumA`), celui du
// tueur (`EnumB`) et l effet de degat (`SrcTag0`).
//
// LA REGLE. Soit V un chassis dont la vie finit `destroyed`, P le paquet (`ObjectDeath.TimestampUS`)
// qui porte son dead-state, et G l ensemble des vies (slot, generation) dont le dead-state est ecrit
// dans P et qui sont V ou ses pieces montees (celles dont `Carrier` est V). Le bipede B est mort avec V
// quand il a un dead-state dans un paquet Pb >= P tel que :
//
//  1. la derniere lecture de parente de B (meme slot, meme generation) STRICTEMENT avant Pb est
//     « attache » a un parent de G ;
//  2. Pb == P, OU le paquet Pb porte la lecture « libre » de B : le film detache B de son parent en
//     meme temps qu il le tue, ce qui ecarte un rattachement perime (la lecture de sortie perdue).
//
// L IDENTITE DE LA VICTIME est celle que le film ecrit : `roster[].filmIndex == DeadState.EnumA`. Ni
// les episodes (`rides[]`) ni leur tolerance « a bord » n y entrent : un mort que les episodes ne
// portent pas se publie. Une victime sans joueur nommable (bot sans xuid, index absent ou hors roster,
// index ambigu) ne se publie pas ; elle se compte (`killedNoIdentity`).
//
// LE NETTOYAGE DE FIN DE MANCHE N EST PAS UNE MORT AVEC LE VEHICULE : un bipede dont l effet de degat est
// [effetNettoyageDeFinDeManche] est ecarte et compte a part (`killedCleanup`).

import (
	"cmp"
	"slices"
	"sort"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// effetNettoyageDeFinDeManche est l identifiant d effet de degat (`DeadState.SrcTag0`) que le film
// ecrit sur un bipede retire a la fin d une manche : une mort sans tueur, qui n a rien d une mort avec
// le vehicule ou il se trouvait.
const effetNettoyageDeFinDeManche = uint32(4208020)

// occupantReads range, par vie de bipede, les lectures de parente (`object-parent-state`) triees par
// instant, et les dead-states des bipedes triees par instant : ce que la regle lit du film.
type occupantReads struct {
	reads  map[types.LifeKey][]types.VehicleOccupancy
	deaths []types.ObjectDeath
}

// newOccupantReads indexe les lectures et les morts de bipede du film.
func newOccupantReads(occupancy []types.VehicleOccupancy, deaths []types.ObjectDeath) occupantReads {
	r := occupantReads{reads: map[types.LifeKey][]types.VehicleOccupancy{}, deaths: slices.Clone(deaths)}
	for _, o := range occupancy {
		k := types.LifeKey{Slot: o.Slot, Gen: o.Gen}
		r.reads[k] = append(r.reads[k], o)
	}
	for _, l := range r.reads {
		slices.SortStableFunc(l, func(a, b types.VehicleOccupancy) int { return cmp.Compare(a.TimestampUS, b.TimestampUS) })
	}
	slices.SortStableFunc(r.deaths, func(a, b types.ObjectDeath) int { return cmp.Compare(a.TimestampUS, b.TimestampUS) })
	return r
}

// deathGroup est G : le paquet P et les vies dont le dead-state y est ecrit.
type deathGroup struct {
	atUS  uint64
	lives []types.LifeKey
}

// has dit si la vie (slot, gen) est dans G.
func (g deathGroup) has(slot, gen uint32) bool {
	return slices.Contains(g.lives, types.LifeKey{Slot: slot, Gen: gen})
}

// mountedPartDeaths range, par chassis porteur, les dead-states des pieces montees dont la mort est
// ecrite.
func mountedPartDeaths(tracks []VehicleTrack) map[VehicleLifeRef][]vehicleDeath {
	out := map[VehicleLifeRef][]vehicleDeath{}
	for _, t := range tracks {
		if t.Part != "" && t.Carrier != nil && t.mort != nil {
			out[*t.Carrier] = append(out[*t.Carrier], *t.mort)
		}
	}
	return out
}

// groupOf rend G pour le chassis `tr` : sa propre vie de mort, et celles de ses pieces mortes dans le
// meme paquet.
func (in vehicleDestructionInputs) groupOf(tr VehicleTrack) deathGroup {
	g := deathGroup{atUS: tr.mort.TimestampUS, lives: []types.LifeKey{{Slot: tr.mort.Slot, Gen: tr.mort.Gen}}}
	for _, p := range in.pieces[VehicleLifeRef{Slot: tr.Slot, Gen: tr.Gen}] {
		if p.TimestampUS == g.atUS {
			g.lives = append(g.lives, types.LifeKey{Slot: p.Slot, Gen: p.Gen})
		}
	}
	return g
}

// diedWith rend, par vie de bipede, la premiere mort ecrite qui satisfait la regle de G, hors
// nettoyage de fin de manche, et le nombre de vies que seule une mort de nettoyage satisfait.
func (r occupantReads) diedWith(g deathGroup) (killed []types.ObjectDeath, cleanup int) {
	done := map[types.LifeKey]bool{}
	cleaned := map[types.LifeKey]bool{}
	start := sort.Search(len(r.deaths), func(i int) bool { return r.deaths[i].TimestampUS >= g.atUS })
	for _, b := range r.deaths[start:] {
		life := types.LifeKey{Slot: b.Slot, Gen: b.Gen}
		if done[life] || !r.leftWithGroup(b, g) {
			continue
		}
		if b.Dead.SrcTag0 == effetNettoyageDeFinDeManche {
			cleaned[life] = true
			continue
		}
		done[life] = true
		killed = append(killed, b)
	}
	for life := range cleaned {
		if !done[life] {
			cleanup++
		}
	}
	return killed, cleanup
}

// leftWithGroup dit si la mort `b` satisfait les deux conditions de la regle : attache a un parent de
// G juste avant, et ecrite dans P ou accompagnee de la lecture « libre » du bipede.
func (r occupantReads) leftWithGroup(b types.ObjectDeath, g deathGroup) bool {
	reads := r.reads[types.LifeKey{Slot: b.Slot, Gen: b.Gen}]
	i := sort.Search(len(reads), func(i int) bool { return reads[i].TimestampUS >= b.TimestampUS })
	if i == 0 {
		return false
	}
	if last := reads[i-1]; !last.Attached || !g.has(last.ParentSlot, last.ParentGen) {
		return false
	}
	if b.TimestampUS == g.atUS {
		return true
	}
	return slices.ContainsFunc(reads[i:], func(o types.VehicleOccupancy) bool {
		return o.TimestampUS == b.TimestampUS && !o.Attached
	})
}

// killedWith rend les xuids, tries, des joueurs morts avec le chassis `tr`, et compte dans `cov`
// ce que la regle lit : tues publies, tues sans identite, morts de nettoyage ecartees.
func (in vehicleDestructionInputs) killedWith(tr VehicleTrack, cov *VehicleDestructionCoverage) []string {
	killed, cleanup := in.reads.diedWith(in.groupOf(tr))
	cov.KilledCleanup += cleanup
	var xuids []string
	for _, b := range killed {
		x := victimXUID(in.roster, b.Dead.EnumA, tr.endFrame())
		switch {
		case x == "":
			cov.KilledNoIdentity++
		case !slices.Contains(xuids, x):
			xuids = append(xuids, x)
		}
	}
	slices.Sort(xuids)
	if len(xuids) > 0 {
		cov.Killed++
		cov.KilledOccupants += len(xuids)
	}
	return xuids
}

// victimXUID nomme la victime d un dead-state de bipede par l index que le film ecrit (`EnumA`), avec
// la resolution du roster que le destructeur emploie ([destroyerOf]). Vide quand le roster ne nomme
// aucun joueur humain a cet index.
func victimXUID(roster []RosterEntry, idx int32, frame int) string {
	by, fate := destroyerOf(roster, idx, frame)
	if fate != destroyerNamed {
		return ""
	}
	return by.XUID
}
