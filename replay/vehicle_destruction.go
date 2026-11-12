package replay

// vehicle_destruction.go — QUI A DETRUIT UN VEHICULE, AVEC QUOI, ET QUI ETAIT A BORD.
//
// CE QUE CE FICHIER AJOUTE A LA FIN `destroyed` : la fin d une vie (`VehicleTrack.End`, `TEnd`) dit
// QUE le film ecrit sa mort et A QUELLE IMAGE ; la destruction (`VehicleTrack.Destruction`) dit ce que
// CE MEME enregistrement ecrit d autre. L instant de la destruction est `TEnd` : date au paquet qui
// porte le dead-state (`vehicleEndOf`), pas a la derniere image vue, et sur le meme axe que `Point.T`.
// Il n est pas recopie dans le bloc : deux dates du meme fait finiraient par diverger.
//
// CE QUE LE FILM ECRIT, SANS FENETRE DE TEMPS NI PROXIMITE (mesure du 2026-11-11, 193 films, 4 465
// morts de vehicule) :
//
//	destructeur   le SECOND champ du dead-state (`DeadState.EnumB`) est l index de joueur du film de
//	              celui qui a detruit le vehicule : sur 1 465 occupants morts dans la demi-seconde, leur
//	              tueur est ce joueur dans 98,0 % des cas. Le premier champ (`EnumA`) est vide. Absent
//	              sur 4,4 % des morts (environnement, chute, cause non etablie) : le bloc n a alors pas
//	              de `by`, jamais un joueur suppose. Le joueur se nomme par le roster du document
//	              (`roster[].filmIndex`), le MEME rattachement qu un lancer de grenade a son auteur ;
//	cause         la source de degat est lue dans `DeadState.SrcTag0` — l identifiant `jpt!` d un effet
//	              de degat, le MEME espace que `SourceTruth.Tag` d un frag (resolu dans 97,5 % des cas).
//	              `GlobalID` n y est pour rien : il est absent sur 100 % des morts de vehicule.
//
// CE QUE LE FILM N ECRIT PAS, ET QUI EST DEDUIT ICI : QUI ETAIT A BORD et QUI EN EST MORT. L occupant
// est un episode du calque (`rides[]`) qui couvre l instant ; les occupants tues sont ceux d entre eux
// dont le fil des morts porte la mort dans la demi-seconde — un REPLI NOMME ET COMPTE
// (`repli_occupants_tues_par_proximite_de_mort`), jamais une lecture.
//
// UNE SEULE DESTRUCTION PAR VEHICULE. Les tourelles et pieces montees (`VehicleTrack.Part`) meurent
// au meme instant que leur porteur, avec le meme destructeur : seule la vie du chassis porte le bloc.
// Les pieces `destroyed` ne portent rien et se comptent (`coverage.vehicles.destructions.pieces`).
//
// PUR : aucune I/O, aucune lecture de film — il range ce que `VehicleScan.Deaths` a rendu.

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"strconv"

	"github.com/JGtm/halo-infinite-film-decoder/damagetag"
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// vehicleDeath est le dead-state d une entite `ti=40` tel que la marche l a lu.
type vehicleDeath = types.ObjectDeath

// VehicleDestruction est ce que le film ecrit de la destruction d un vehicule, et ce qu on en deduit.
// Present sur une vie `destroyed` qui n est pas une piece montee, quand AU MOINS UN de ses champs est
// etabli ; chaque champ absent veut dire « non etabli », jamais 0 ni chaine vide.
type VehicleDestruction struct {
	// By est le destructeur. Absent : le dead-state n en porte pas, ou son index n est pas un
	// joueur du roster (cf. `coverage.vehicles.destructions`).
	By *VehicleDestroyer `json:"by,omitempty"`
	// Cause est l identifiant `jpt!` de l effet de degat, au meme format que la source d un frag
	// (entier 32 bits non signe). Absent : le film n en ecrit pas, ou la valeur n est pas un effet
	// de degat connu de `damagetag` — elle ne se publie pas, elle se compte.
	Cause *uint32 `json:"cause,omitempty"`
	// Occupant est le xuid de l occupant a l instant de la destruction, le conducteur (siege 0) de
	// preference. Absent : aucun episode NOMME ne couvre l instant. Il vient du calque des episodes
	// (`rides[]`), dont la couverture est celle de la primitive d occupation (15,6 a 21,1 % des vies).
	Occupant string `json:"occupant,omitempty"`
	// Killed sont les xuids des occupants a bord dont le fil des morts porte la mort dans la
	// demi-seconde de la destruction, tries. DEDUIT : cf. le repli nomme en tete de fichier. Absent :
	// aucun occupant nomme n est mort a cet instant, ou l horloge du fil des morts n est pas calee.
	Killed []string `json:"killed,omitempty"`
}

// VehicleDestroyer designe le joueur qui a detruit le vehicule.
type VehicleDestroyer struct {
	// I est l index de joueur ECRIT dans le film (`roster[].filmIndex`), toujours renseigne — meme
	// champ que `Grenade.Idx`.
	I int `json:"i"`
	// XUID est l identite du destructeur, en decimal. Absent pour un BOT, qui n en a pas : `bot` le dit.
	XUID string `json:"xuid,omitempty"`
	// Bot est vrai quand l index designe une entree de roster declaree par BOT_METADATA.
	Bot bool `json:"bot,omitempty"`
}

// VehicleDestructionCoverage dit ce que le bloc a vu, resolu et refuse de dire. Le denominateur est
// `Destroyed` ; les causes du destructeur (`ByHuman`, `ByBot` et les trois refus) en sont une
// partition, comme celles de la cause (`Cause`, `CauseUnresolved`, `CauseAbsent`).
type VehicleDestructionCoverage struct {
	// Destroyed : vies `destroyed` hors pieces montees. Pieces : pieces montees `destroyed`, qui ne
	// portent pas de bloc (la destruction est celle de leur porteur). Published : blocs poses.
	Destroyed int `json:"destroyed"`
	Pieces    int `json:"pieces"`
	Published int `json:"published"`
	// ByHuman / ByBot : destructeur nomme. KillerAbsent : le dead-state n a pas d index de joueur.
	// KillerOutOfRoster : l index n est celui d aucune entree du roster. KillerAmbiguous : plusieurs
	// entrees le portent et la presence ne tranche pas.
	ByHuman           int `json:"byHuman"`
	ByBot             int `json:"byBot"`
	KillerAbsent      int `json:"killerAbsent"`
	KillerOutOfRoster int `json:"killerOutOfRoster"`
	KillerAmbiguous   int `json:"killerAmbiguous"`
	// Cause : effet de degat connu, publie. CauseUnresolved : valeur lue, inconnue de `damagetag`
	// (non publiee). CauseAbsent : aucune valeur ecrite.
	Cause           int `json:"cause"`
	CauseUnresolved int `json:"causeUnresolved"`
	CauseAbsent     int `json:"causeAbsent"`
	// Occupant : destructions dont un episode nomme couvre l instant.
	Occupant int `json:"occupant"`
	// Killed : destructions qui portent au moins un occupant tue ; KilledOccupants leur nombre total
	// (la valeur du repli). ClockUnknown : destructions dont les occupants tues n ont pas pu etre
	// cherches, faute de calage du fil des morts sur l horloge du film.
	Killed          int `json:"killed"`
	KilledOccupants int `json:"killedOccupants"`
	ClockUnknown    int `json:"clockUnknown"`
}

const (
	// deadStateTagAbsent est la valeur que le moteur ecrit quand un champ `readOpt(1+32)` du
	// dead-state est absent (`types.DeadState.SrcTag0`).
	deadStateTagAbsent = uint32(0xFFFFFFFF)
	// vehicleDestructionAboardTolMS est la marge dans laquelle un episode qui prend fin AVANT l instant
	// de la destruction y est encore compte a bord : la sortie et la mort ne sont pas datees au meme
	// instant. Convertie en images a la cadence du document.
	vehicleDestructionAboardTolMS = 200
	// vehicleDestructionKilledWindowMS borne, de part et d autre de la destruction, la mort d un
	// occupant que le repli lui rapporte (horloge du fil des morts).
	vehicleDestructionKilledWindowMS = int64(500)
)

// destroyerFate dit ce que la resolution de l index du destructeur a rendu.
type destroyerFate int

const (
	destroyerNamed destroyerFate = iota
	destroyerAbsent
	destroyerOutOfRoster
	destroyerAmbiguous
)

// vehicleDestructionInputs porte ce que la pose lit hors du calque : le roster, le fil des morts, et
// son calage sur l horloge du film (nil quand il n est pas connu).
type vehicleDestructionInputs struct {
	roster []RosterEntry
	deaths []types.Death
	calage *int64
	fb     *fallback.Compteur
	// aboardTol est `vehicleDestructionAboardTolMS` en images du document, pose par la pose.
	aboardTol int
}

// attachVehicleDestructions pose `Destruction` sur les vies `destroyed` du document et leur
// couverture. APRES `attachVehicles` : les episodes sont ceux que le document publie.
func attachVehicleDestructions(ctx context.Context, doc *ReplayDocument, in vehicleDestructionInputs) {
	if doc.Coverage == nil || doc.Coverage.Vehicles == nil || !doc.Coverage.Vehicles.Scanned {
		return
	}
	in.aboardTol = framesOfMS(vehicleDestructionAboardTolMS, doc.FrameIntervalMS)
	var cov VehicleDestructionCoverage
	for i := range doc.Vehicles {
		tr := &doc.Vehicles[i]
		if tr.End != VehicleEndDestroyed || tr.mort == nil {
			continue
		}
		if tr.Part != "" {
			cov.Pieces++
			continue
		}
		cov.Destroyed++
		if d := in.destructionOf(ctx, *tr, &cov); d != nil {
			tr.Destruction = d
			cov.Published++
		}
	}
	if cov.Destroyed == 0 && cov.Pieces == 0 {
		return // aucune destruction : le document reste celui d avant, sans bloc de couverture
	}
	in.fb.DeclencheN(fallback.NomOccupantsTuesParProximiteDeMort, cov.KilledOccupants)
	doc.Coverage.Vehicles.Destructions = &cov
	logVehicleDestructions(ctx, cov)
}

// destructionOf assemble le bloc d une vie et compte ce qu il porte. Nil quand rien n est etabli.
func (in vehicleDestructionInputs) destructionOf(ctx context.Context, tr VehicleTrack, cov *VehicleDestructionCoverage) *VehicleDestruction {
	var d VehicleDestruction
	dead := tr.mort.Dead
	var fate destroyerFate
	d.By, fate = destroyerOf(in.roster, dead.EnumB, tr.endFrame())
	cov.countDestroyer(fate, d.By)
	if cause, known := damageCauseOf(dead.SrcTag0); known {
		d.Cause = &cause
	}
	cov.countCause(dead.SrcTag0, d.Cause != nil)
	aboard := ridesAboardAt(tr.Rides, tr.endFrame(), in.aboardTol)
	d.Occupant = destructionOccupant(aboard)
	if d.Occupant != "" {
		cov.Occupant++
	}
	d.Killed = in.killedAmong(ctx, aboard, tr.mort.TimestampUS, cov)
	if len(d.Killed) > 0 {
		cov.Killed++
		cov.KilledOccupants += len(d.Killed)
	}
	if d.By == nil && d.Cause == nil && d.Occupant == "" && len(d.Killed) == 0 {
		return nil
	}
	return &d
}

// countDestroyer range la resolution du destructeur dans la partition de la couverture.
func (c *VehicleDestructionCoverage) countDestroyer(fate destroyerFate, by *VehicleDestroyer) {
	switch {
	case fate == destroyerNamed && by.Bot:
		c.ByBot++
	case fate == destroyerNamed:
		c.ByHuman++
	case fate == destroyerAbsent:
		c.KillerAbsent++
	case fate == destroyerOutOfRoster:
		c.KillerOutOfRoster++
	default:
		c.KillerAmbiguous++
	}
}

// countCause range la valeur lue de `SrcTag0` dans la partition de la cause : publiee, lue mais inconnue
// de damagetag, ou absente.
func (c *VehicleDestructionCoverage) countCause(tag uint32, published bool) {
	switch {
	case published:
		c.Cause++
	case tag == deadStateTagAbsent:
		c.CauseAbsent++
	default:
		c.CauseUnresolved++
	}
}

// endFrame rend la frame de la fin ecrite : `TEnd`, toujours posee sur une vie `destroyed`.
func (tr VehicleTrack) endFrame() int {
	if tr.TEnd == nil {
		return tr.T1Max
	}
	return *tr.TEnd
}

// damageCauseOf rend l effet de degat que le dead-state ecrit, quand `damagetag` le connait.
func damageCauseOf(tag uint32) (uint32, bool) {
	if tag == deadStateTagAbsent || !damagetag.IsDamageEffect(tag) {
		return 0, false
	}
	return tag, true
}

// destroyerOf nomme le joueur dont l index est `idx` (`DeadState.EnumB`, -1 quand le film n en ecrit
// pas) a la frame donnee. Les entrees du roster qui portent cet index sont departagees par leur
// presence a cette frame ; un index porte par un humain et un bot, ou par deux humains, que la
// presence ne tranche pas, n est pas nomme.
func destroyerOf(roster []RosterEntry, idx int32, frame int) (*VehicleDestroyer, destroyerFate) {
	if idx < 0 {
		return nil, destroyerAbsent
	}
	var cands []RosterEntry
	for _, e := range roster {
		if e.FilmIndex == int(idx) {
			cands = append(cands, e)
		}
	}
	if len(cands) == 0 {
		return nil, destroyerOutOfRoster
	}
	if len(cands) > 1 {
		cands = presentAt(cands, frame)
	}
	switch {
	case len(cands) == 1 && cands[0].Bot:
		return &VehicleDestroyer{I: int(idx), Bot: true}, destroyerNamed
	case len(cands) == 1 && cands[0].XUID != "":
		return &VehicleDestroyer{I: int(idx), XUID: cands[0].XUID}, destroyerNamed
	case len(cands) > 1 && allBots(cands):
		return &VehicleDestroyer{I: int(idx), Bot: true}, destroyerNamed
	}
	return nil, destroyerAmbiguous
}

// presentAt garde les entrees dont une presence couvre la frame ; sans aucune, toutes (la presence
// est une borne d affichage, absente d un roster sans table de places).
func presentAt(cands []RosterEntry, frame int) []RosterEntry {
	var out []RosterEntry
	for _, e := range cands {
		if presenceCouvre(e.Presence, frame) {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		return cands
	}
	return out
}

func allBots(es []RosterEntry) bool {
	for _, e := range es {
		if !e.Bot {
			return false
		}
	}
	return true
}

// ridesAboardAt rend les episodes a bord a la frame `t`, le conducteur d abord : ceux qui la couvrent,
// et ceux qui ont pris fin au plus `tol` images avant.
func ridesAboardAt(rides []VehicleRide, t, tol int) []VehicleRide {
	var out []VehicleRide
	for _, r := range rides {
		if r.T0 <= t && r.T1 >= t-tol {
			out = append(out, r)
		}
	}
	slices.SortStableFunc(out, func(a, b VehicleRide) int {
		return cmp.Or(cmp.Compare(vehicleSeatRank(a.Seat), vehicleSeatRank(b.Seat)),
			cmp.Compare(b.T1, a.T1), cmp.Compare(a.Slot, b.Slot))
	})
	return out
}

// destructionOccupant rend le xuid du premier episode nomme de `aboard` (deja ordonne conducteur d abord).
func destructionOccupant(aboard []VehicleRide) string {
	for _, r := range aboard {
		if r.XUID != "" {
			return r.XUID
		}
	}
	return ""
}

// killedAmong est LE REPLI `repli_occupants_tues_par_proximite_de_mort` : ceux des occupants nommes
// de `aboard` dont le fil des morts porte la mort a moins de `vehicleDestructionKilledWindowMS` de
// l instant `deathUS` de la destruction (horloge du film). Sans calage du fil des morts sur l horloge
// du film, rien n est cherche et la destruction est comptee `ClockUnknown`.
func (in vehicleDestructionInputs) killedAmong(ctx context.Context, aboard []VehicleRide, deathUS uint64, cov *VehicleDestructionCoverage) []string {
	var xuids []string
	for _, r := range aboard {
		if r.XUID != "" && !slices.Contains(xuids, r.XUID) {
			xuids = append(xuids, r.XUID)
		}
	}
	if len(xuids) == 0 {
		return nil
	}
	if in.calage == nil {
		cov.ClockUnknown++
		return nil
	}
	at := int64(deathUS)/1000 - *in.calage
	var out []string
	for _, x := range xuids {
		n, err := strconv.ParseUint(x, 10, 64)
		if err != nil {
			slog.WarnContext(ctx, "rejeu : xuid d occupant illisible, aucune mort cherchee", "xuid", x, "err", err)
			continue
		}
		if diedWithin(in.deaths, n, at) {
			out = append(out, x)
		}
	}
	slices.Sort(out)
	return out
}

// diedWithin dit si le fil des morts porte une mort de `xuid` dans la fenetre autour de `atMS`.
func diedWithin(deaths []types.Death, xuid uint64, atMS int64) bool {
	for _, d := range deaths {
		if d.XUID == xuid && d.TimeMS >= atMS-vehicleDestructionKilledWindowMS && d.TimeMS <= atMS+vehicleDestructionKilledWindowMS {
			return true
		}
	}
	return false
}

// logVehicleDestructions dit ce que la pose a publie et refuse : un compteur muet est une erreur avalee.
func logVehicleDestructions(ctx context.Context, c VehicleDestructionCoverage) {
	if c.Destroyed == 0 && c.Pieces == 0 {
		return
	}
	slog.InfoContext(ctx, "rejeu : destructions de vehicule",
		"detruits", c.Destroyed, "pieces", c.Pieces, "publiees", c.Published,
		"parHumain", c.ByHuman, "parBot", c.ByBot, "destructeurAbsent", c.KillerAbsent,
		"destructeurHorsRoster", c.KillerOutOfRoster, "destructeurAmbigu", c.KillerAmbiguous,
		"cause", c.Cause, "causeNonResolue", c.CauseUnresolved, "causeAbsente", c.CauseAbsent,
		"avecOccupant", c.Occupant, "avecTues", c.Killed, "tues", c.KilledOccupants,
		"horlogeInconnue", c.ClockUnknown)
}
