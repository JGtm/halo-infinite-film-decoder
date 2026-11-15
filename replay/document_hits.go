package replay

// document_hits.go — LES TOUCHES ET LE BILAN DE TIRS PAR JOUEUR, LUS DANS LE FILM : UNE ESTIMATION.
//
// CE QUE C'EST. Une touche est un message `damage_aftermath` (genre 0 de la vue A) dont l'effet de dégât est un
// effet d'arme ([damageeffects.EstUneTouche]), compté une fois par (auteur, unité blessée, trame) : une rafale de
// plomb d'un fusil à pompe qui touche la même unité dans la même trame est une touche. Les projectiles sont
// compris : leur effet est un effet d'arme comme un autre. L'arme est celle du dernier tir de l'auteur dans les
// deux secondes qui précèdent (`w`) ; l'effet de dégât, lui, est toujours publié (`effect`).
//
// C'EST UNE ESTIMATION, ET LE DOCUMENT LE DIT : `coverage.hits.estimated` vaut toujours vrai. Le film n'écrit pas
// le nombre de touches que l'API compte (`shots_hit`) ; il écrit des dégâts, et la mesure les compare à elle
// (chronique v125, annexe du seuil). Un joueur dont les données ne suffisent pas porte `sufficient` faux : le
// consommateur n'affiche alors que « données insuffisantes », jamais le chiffre.
//
// LE BILAN PAR JOUEUR (`shotStats[]`). Le jeu numérote les tirs de chaque joueur modulo 256 à CHAQUE tir, écrit ou
// non : la somme des sauts de ce numéro est le nombre de tirs ([ShotStat.Shots]), alors que les événements de tir
// du film n'en écrivent qu'une part ([ShotStat.ShotEvents]). Trois critères, mesurés, disent si les données d'un
// joueur suffisent :
//   - la couverture, `shotEvents / shots`, au moins [minCouvertureDesTirs] : sous elle, l'arme et les tirs continus
//     manquent trop pour que les touches se comparent aux tirs ;
//   - les doublons, `messages / hits`, sous [maxDoublonsDeTouches] : un joueur dont les touches s'écrivent en plusieurs messages par
//     trame (fusil à pompe, rafale de plusieurs effets) se compte moins bien ;
//   - la lecture : la vue A ne s'arrête sur un message non lu, et les records de vie des bipèdes ne restent illisibles, que sur au plus
//     [maxPartDeTramesArretees] des trames du film (ensemble) ; au-delà, les
//     messages de dégât perdus derrière l'arrêt rendent les touches incomparables à celles de l'API.
//
// Les deux rapports et les seuils sont publiés avec le bilan : le consommateur peut les afficher, la décision ne lui
// appartient pas. Un film dont les messages de dégât ne se lisent pas du tout n'a aucune touche : tous ses
// joueurs sont insuffisants.
//
// LES TIRS SONT CEUX DU RESTE DE L'ASSEMBLAGE ([assemblage.tirsDuCombat]) : sur un film dont l'index de tireur n'est pas la place
// (`coverage.seats.tirsIndexNonPlace`), aucun tir ne s'attribue par l'index : ni tir écrit, ni numérotation, ni arme, et aucun joueur n'est suffisant.
// La numérotation se suit par joueur résolu, pas par index : un remplaçant sur le même index a la sienne.
//
// L'IDENTITÉ. L'auteur est l'index de joueur que le message écrit, résolu à la frame du message parmi les humains
// du roster présents (cf. [xuidALaFrame]) ; la victime est l'unité blessée, dont la piste (slot = index d'unité
// + 512) porte le xuid à cette frame. UNE TOUCHE SUR SOI (la victime est l'auteur : une arme à zone) n'est pas une touche : elle
// est comptée à part (`coverage.hits.selfHits`), comme le dégât qu'un joueur se fait à lui-même (`selfDamages`) n'est pas du dégât
// infligé. Une touche dont l'un des deux ne se résout pas est comptée au bilan de
// l'auteur et à la couverture, pas publiée.

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/damageeffects"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Seuils de suffisance des données d'un joueur, mesurés (annexe du seuil, chronique v125).
const (
	minCouvertureDesTirs    = 0.7
	maxDoublonsDeTouches    = 1.2
	maxPartDeTramesArretees = 0.005
)

// fenetreDeLArmeUS est la durée, en microsecondes, avant une touche dans laquelle le dernier tir de l'auteur
// nomme l'arme.
const fenetreDeLArmeUS = 2_000_000

// decalageDesSlotsDUnite est l'écart entre l'index d'unité d'un message de dégât et le slot de sa piste.
const decalageDesSlotsDUnite = 512

// Hit est UNE touche estimée : quand, qui tire, qui est touché, par quel effet et avec quelle arme.
type Hit struct {
	// T est l'index de frame, sur le même axe que Point.T.
	T int `json:"t"`
	// Shooter et Victim sont les joueurs, en décimal — même forme que Track.XUID.
	Shooter string `json:"shooter"`
	Victim  string `json:"victim"`
	// Effect est l'identifiant de l'effet de dégât (le tag `jpt!`), en hexadécimal sur huit chiffres.
	Effect string `json:"effect"`
	// Weapon est l identifiant global de l arme du dernier tir de l auteur dans les deux secondes qui précèdent,
	// même forme que Shot.Weapon (clé de metadata.weapon_labels.weapon_id). Absent quand l auteur n a pas tiré à ce moment.
	Weapon string `json:"w,omitempty"`
}

// ShotStat est le bilan de tirs d'un joueur sur tout le film, et le verdict de suffisance de ses données.
type ShotStat struct {
	XUID string `json:"xuid"`
	// Shots est le nombre de tirs estimé par la numérotation des tirs ; ShotEvents, celui des événements de tir
	// que le film écrit.
	Shots      int `json:"shots"`
	ShotEvents int `json:"shotEvents"`
	// Messages est le nombre de messages de dégât d'arme de l'auteur ; Hits, celui des touches (un par auteur,
	// unité blessée et trame).
	Messages int `json:"messages"`
	Hits     int `json:"hits"`
	// Coverage vaut ShotEvents / Shots, Duplicates vaut Messages / Hits ; zéro quand leur dénominateur l'est.
	// DamageDealt et DamageTaken sont les points de dégât infligés à d autres joueurs ou à des bots (hors soi-même) et subis, au dixième près
	// (cf. document_damage.go) ; ils couvrent TOUT le film, alors que `damage[]` ne porte que l instant tenant dans l axe ; Sufficient les couvre aussi.
	DamageDealt float64 `json:"damageDealt"`
	DamageTaken float64 `json:"damageTaken"`
	Coverage    float64 `json:"coverage"`
	Duplicates  float64 `json:"duplicates"`
	// Sufficient : les données du joueur suffisent (cf. l'en-tête du fichier). Faux, le consommateur n'affiche
	// aucun chiffre de tir ni de touche.
	Sufficient bool `json:"sufficient"`
}

// HitInput est ce que l'assemblage reçoit des dégâts : les messages `damage_aftermath` lus dans le film et le
// témoin qui dit qu'ils l'ont été. Faux, les calques ne sont pas produits.
type HitInput struct {
	Read     bool
	Messages []grammar.DegatLu
	Stats    grammar.StatsDegats
	// Points : ce que chaque message retire a son unite blessee, dans le meme ordre ([PointsDeDegat]) ; plus court que Messages, les
	// messages au-dela ne retirent rien.
	Points []PointsDeDegat
}

// HitCoverage dit ce que le calque a lu, publié et écarté.
type HitCoverage struct {
	// Estimated vaut toujours vrai : les touches sont une estimation.
	Estimated bool `json:"estimated"`
	// MinCoverage et MaxDuplicates sont les seuils de suffisance appliqués aux joueurs.
	MinCoverage   float64 `json:"minCoverage"`
	MaxDuplicates float64 `json:"maxDuplicates"`
	// Frames, StoppedFrames et MaxStoppedShare : les trames delta du film, celles dont la lecture de la vue A s arrete sur un message non lu (les
	// messages de degat derriere l arret sont perdus), et la part d arret au-dela de laquelle aucun joueur n a de donnees suffisantes.
	Frames        int `json:"frames"`
	StoppedFrames int `json:"stoppedFrames"`
	// VitalsUnread : les trames dont les records de vie des bipedes restent illisibles sans que la vue A s y arrete ; elles comptent avec les trames
	// arretees dans la part de lecture (leurs points de degat se chiffrent sur une vie incomplete).
	VitalsUnread    int     `json:"vitalsUnread"`
	MaxStoppedShare float64 `json:"maxStoppedShare"`
	// Messages est le nombre de messages de dégât lus ; WeaponMessages, ceux dont l'effet est un effet d'arme et
	// qui nomment un auteur ; Hits, les touches qu'ils font (un par auteur, unité blessée et trame).
	Messages       int `json:"messages"`
	WeaponMessages int `json:"weaponMessages"`
	Hits           int `json:"hits"`
	// Published est le nombre de touches posées sur l'axe ; les quatre suivants, celles qui ne le sont pas :
	// auteur sans humain présent au roster, victime sans piste à cette frame, avant la frame 0, après la dernière.
	Published    int `json:"published"`
	NoShooter    int `json:"noShooter"`
	NoVictim     int `json:"noVictim"`
	BeforeOrigin int `json:"beforeOrigin"`
	AfterEnd     int `json:"afterEnd"`
	// Sufficient est le nombre de joueurs dont les données suffisent, sur Players joueurs au bilan.
	Sufficient int `json:"sufficient"`
	// Damages est le nombre de lignes de `damage[]` posées, DamagesMeasured celles dont les points viennent de la vie lue autour du coup, DamagesOffUnit les
	// messages dont l unite blessee n a pas de piste a cette frame (un vehicule, un objet) : ni publies ni comptes.
	Damages         int `json:"damages"`
	DamagesMeasured int `json:"damagesMeasured"`
	DamagesOffUnit  int `json:"damagesOffUnit"`
	// SelfHits est le nombre de touches d une arme sur son propre tireur (une arme a zone), SelfDamages celui des lignes de degat qu un joueur se fait a
	// lui-meme : ni les unes ni les autres ne sont des touches ni du degat inflige, elles se comptent a part.
	SelfHits    int `json:"selfHits"`
	SelfDamages int `json:"selfDamages"`
	Players     int `json:"players"`
}

// armesDesTirs range les tirs de chaque auteur par instant, pour retrouver l arme du dernier avant une touche.
type armesDesTirs map[int][]grammar.FireEvent

func rangerLesTirs(fire []grammar.FireEvent) armesDesTirs {
	out := armesDesTirs{}
	for _, e := range fire {
		if e.HasShooter {
			out[e.FilmIndex] = append(out[e.FilmIndex], e)
		}
	}
	for _, s := range out {
		slices.SortStableFunc(s, func(a, b grammar.FireEvent) int { return cmp.Compare(a.TimestampUS, b.TimestampUS) })
	}
	return out
}

// derniereArme rend l'arme du dernier tir de l'auteur `joueur` entre `ts - fenetreDeLArmeUS` et `ts`, ou "".
func (a armesDesTirs) derniereArme(joueur int, ts uint64) string {
	s := a[joueur]
	i, _ := slices.BinarySearchFunc(s, ts, func(e grammar.FireEvent, ts uint64) int { return cmp.Compare(e.TimestampUS, ts+1) })
	if i == 0 || ts-s[i-1].TimestampUS > fenetreDeLArmeUS {
		return ""
	}
	return formatWeaponID(s[i-1].WeaponID)
}

// resolveurDeTouches résout les index de joueur et les unités d'un film en xuid, à une frame.
type resolveurDeTouches struct {
	humains map[int][]RosterEntry
	tracks  []Track
	axe     axeDeFrames
}

// frameBornee rend la frame de l'instant `ts`, ramenée dans l'axe pour la RESOLUTION d'un joueur (jamais pour la
// pose d'une touche) : un instant avant l'origine résout à la frame 0, après la fin à la dernière.
func (r resolveurDeTouches) frameBornee(ts uint64) int {
	t, refus := frameDeLAction(ts, r.axe)
	switch refus {
	case refusAvantOrigine:
		return 0
	case refusApresFin:
		return max(r.axe.frames-1, 0)
	}
	return t
}

// victime rend le xuid du joueur dont la piste est l'unité `unite` à la frame `t` (vide pour un bot : une piste sans xuid),
// et si une piste tient cette unité à cette frame.
func (r resolveurDeTouches) victime(unite uint16, t int) (xuid string, piste bool) {
	for _, tr := range r.tracks {
		if tr.Slot == uint32(unite)+decalageDesSlotsDUnite && t >= tr.StartFrame && (tr.EndFrame == 0 || t <= tr.EndFrame+1) {
			if tr.XUID != "" {
				return tr.XUID, true
			}
			piste = true
		}
	}
	return "", piste
}

// bilanDesTirs compte, par joueur résolu, les tirs écrits, la numérotation des tirs et les messages et touches.
type bilanDesTirs map[string]*ShotStat

func (b bilanDesTirs) de(xuid string) *ShotStat {
	if b[xuid] == nil {
		b[xuid] = &ShotStat{XUID: xuid}
	}
	return b[xuid]
}

// compterLesTirs ajoute au bilan les tirs écrits et la numérotation des tirs de chaque joueur.
func compterLesTirs(fire []grammar.FireEvent, r resolveurDeTouches, bilan bilanDesTirs) {
	// LA NUMEROTATION EST CELLE D UN JOUEUR, pas celle d un index : deux humains successifs sur le meme index (un remplacant) ont chacun la leur,
	// et le saut du premier numero du remplacant ne se compte pas contre le dernier du partant.
	derniers := map[string]int{}
	tries := slices.Clone(fire)
	slices.SortStableFunc(tries, func(a, b grammar.FireEvent) int { return cmp.Compare(a.TimestampUS, b.TimestampUS) })
	for _, e := range tries {
		if !e.HasShooter {
			continue
		}
		xuid, res := xuidALaFrame(r.humains, e.FilmIndex, r.frameBornee(e.TimestampUS))
		if res != destinataireResolu {
			continue
		}
		saut := 1
		if dernier, vu := derniers[xuid]; vu {
			saut = (int(e.FireNumber) - dernier + 256) % 256
		}
		derniers[xuid] = int(e.FireNumber)
		s := bilan.de(xuid)
		s.ShotEvents++
		s.Shots += saut
	}
}

// cleDeTouche identifie une touche : l'auteur, l'unité blessée, la trame.
type cleDeTouche struct {
	joueur int
	unite  uint16
	ts     uint64
}

// buildHits pose les touches du film sur l'axe de frames et dresse le bilan de tirs de chaque joueur. Les touches
// rendues sont triées par frame, auteur, victime et effet ; le bilan, par xuid.
func buildHits(in HitInput, fire []grammar.FireEvent, roster []RosterEntry, tracks []Track, axe axeDeFrames) ([]Hit, []Damage, []ShotStat, HitCoverage) {
	cov := HitCoverage{Estimated: true, MinCoverage: minCouvertureDesTirs, MaxDuplicates: maxDoublonsDeTouches, MaxStoppedShare: maxPartDeTramesArretees,
		Messages: len(in.Messages), Frames: in.Stats.Trames, StoppedFrames: in.Stats.Arretees, VitalsUnread: in.Stats.VitalesNonLues}
	r := resolveurDeTouches{humains: humainsParIndex(roster), tracks: tracks, axe: axe}
	bilan := bilanDesTirs{}
	compterLesTirs(fire, r, bilan)
	armes := rangerLesTirs(fire)
	msgs := messagesParInstant(in)
	vues := map[cleDeTouche]bool{}
	var hits []Hit
	for _, m := range msgs {
		d := m.d
		if d.Joueur < 0 || !d.ATag || !d.AVictime || !damageeffects.EstUneTouche(d.Tag) {
			continue
		}
		cov.WeaponMessages++
		t := r.frameBornee(d.TS)
		tireur, res := xuidALaFrame(r.humains, int(d.Joueur), t)
		cle := cleDeTouche{int(d.Joueur), d.Victime, d.TS}
		if victime, _ := r.victime(d.Victime, t); res == destinataireResolu && victime == tireur {
			if !vues[cle] {
				vues[cle] = true
				cov.SelfHits++
			}
			continue
		}
		if res == destinataireResolu {
			bilan.de(tireur).Messages++
		}
		if vues[cle] {
			continue
		}
		vues[cle] = true
		cov.Hits++
		if res == destinataireResolu {
			bilan.de(tireur).Hits++
		}
		if h, ok := poserUneTouche(d, tireur, res, r, armes, &cov); ok {
			hits = append(hits, h)
		}
	}
	cov.Published = len(hits)
	slices.SortStableFunc(hits, func(a, b Hit) int {
		return cmp.Or(cmp.Compare(a.T, b.T), cmp.Compare(a.Shooter, b.Shooter), cmp.Compare(a.Victim, b.Victim), cmp.Compare(a.Effect, b.Effect))
	})
	lecturePleine := cov.Frames > 0 && float64(cov.StoppedFrames+cov.VitalsUnread) <= maxPartDeTramesArretees*float64(cov.Frames)
	damages := poserLesDegats(msgs, r, armes, bilan, &cov)
	stats := arreterLesBilans(bilan, lecturePleine, &cov)
	return hits, damages, stats, cov
}

// poserUneTouche pose la touche du message `d` sur l'axe, ou compte pourquoi elle n'y est pas.
func poserUneTouche(d grammar.DegatLu, tireur string, res issueDuDestinataire, r resolveurDeTouches, armes armesDesTirs, cov *HitCoverage) (Hit, bool) {
	t, refus := frameDeLAction(d.TS, r.axe)
	switch {
	case refus == refusAvantOrigine:
		cov.BeforeOrigin++
	case refus == refusApresFin:
		cov.AfterEnd++
	case refus == refusSansOrigine || res != destinataireResolu:
		cov.NoShooter++
	default:
		victime, _ := r.victime(d.Victime, t)
		if victime == "" {
			cov.NoVictim++
			return Hit{}, false
		}
		return Hit{T: t, Shooter: tireur, Victim: victime, Effect: formaterLEffet(d.Tag), Weapon: armes.derniereArme(int(d.Joueur), d.TS)}, true
	}
	return Hit{}, false
}

// arreterLesBilans calcule les deux rapports et le verdict de chaque joueur, trie le bilan par xuid et complète la
// couverture de leur compte.
func arreterLesBilans(bilan bilanDesTirs, lecturePleine bool, cov *HitCoverage) []ShotStat {
	stats := make([]ShotStat, 0, len(bilan))
	for _, s := range bilan {
		if s.Shots > 0 {
			s.Coverage = float64(s.ShotEvents) / float64(s.Shots)
		}
		s.DamageDealt, s.DamageTaken = math.Round(s.DamageDealt*10)/10, math.Round(s.DamageTaken*10)/10
		if s.Hits > 0 {
			s.Duplicates = float64(s.Messages) / float64(s.Hits)
		}
		s.Sufficient = lecturePleine && s.Shots > 0 && s.Hits > 0 && s.Coverage >= minCouvertureDesTirs && s.Duplicates < maxDoublonsDeTouches
		if s.Sufficient {
			cov.Sufficient++
		}
		stats = append(stats, *s)
	}
	slices.SortFunc(stats, func(a, b ShotStat) int { return cmp.Compare(a.XUID, b.XUID) })
	cov.Players = len(stats)
	return stats
}

// poserLesTouches publie les calques : seulement quand les messages de dégât ont été lus, et APRES le roster et ses
// présences, les pistes dont il tire les xuids, et la couverture qu'il complète.
func (a *assemblage) poserLesTouches() {
	if !a.opt.Damages.Read {
		return
	}
	hits, damages, stats, cov := buildHits(a.opt.Damages, a.tirsDuCombat(), a.doc.Roster, a.doc.Tracks, axeDeFrames{a.origin, a.step, a.doc.FrameCount})
	a.doc.Hits, a.doc.Damage, a.doc.ShotStats = hits, damages, stats
	a.doc.Coverage.Hits = &cov
}

// formaterLEffet écrit l'identifiant d'un effet de dégât : huit chiffres hexadécimaux.
func formaterLEffet(tag uint32) string { return fmt.Sprintf("%08x", tag) }
