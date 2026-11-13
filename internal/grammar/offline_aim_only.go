package grammar

// offline_aim_only.go — LA VISEE D'UN BIPEDE QUI NE REPLIQUE PLUS SA POSITION.
//
// # LE POINT AVEUGLE QUE CE FICHIER OUVRE (lot V11, 2026-09-03)
//
// `ScanBipedRecords` n'accepte un record que s'il porte un `i0` ABSOLU de la region attendue
// ET si le premier index de son masque vaut 0 (`ascendingFromZero`). C'est une exigence du
// DETECTEUR, pas du format : le masque d'un record delta n'a aucune obligation de declarer
// `i0`. Un record qui ne replique que la visee lui est donc STRUCTURELLEMENT invisible.
//
// Or c'est exactement ce que fait un occupant de vehicule. Le modele V1a.4 dit qu'il cesse de
// repliquer sa POSITION monde — c'est la primitive du « trou » sur laquelle repose toute
// l'attribution des episodes d'occupation. Il ne dit rien de sa VISEE, et la mesure du lot V11
// tranche : le flux `i21 unit-desired-aiming-vector` CONTINUE pendant le trou.
//
// # CE QUE LA MESURE ETABLIT (V11_ORIENTATION_TOURELLE_2026-09-03.md)
//
//	PRESENCE    17 838 / 4 052 / 17 911 lectures sur `0d76e8f1` / `fccc61cd` / `4898d586`,
//	            soit 173,2 / 39,0 / 172,2 par slot bipede — contre 0,5 / 0,2 / 0,1 par slot sur
//	            une bande FANTOME de meme cardinalite (slots jamais vus porter un archetype).
//	            Rapport signal / plancher de bruit : x347 a x1722.
//	JUSTESSE    appariee a la lecture `i21` AVEC i0 du meme slot a moins de 200 ms, l'ecart
//	            median de cap vaut 0,4 deg (R 0,975 / 0,983 / 0,989). TEMOIN par melange
//	            deterministe : 86,5 / 87,0 deg, R 0,047 / 0,041.
//	UTILITE     sur les episodes d'occupation ATTESTES par la sortie, 24 / 25 portent au moins
//	            une visee a bord, a 5 a 46 lectures par seconde, la ou le meme episode ne porte
//	            0 ou 1 lecture `i21` AVEC position. C'est la visee du CONDUCTEUR, de
//	            l'ARTILLEUR et du PASSAGER — chacun sur son propre slot bipede.
//
// # AUCUNE GRAMMAIRE N'EST REECRITE ICI
//
// L'en-tete est celle de `matchBipedHeaderRaw` (meme prefixe, meme slot 13 bits, meme filtre de generation,
// meme couple de zeros, meme compteur de masque), a la SEULE exception documentee : le premier
// index du masque n'est pas contraint a zero. Les composants qui precedent `i21` sont consommes
// par leurs detenteurs existants (`readVelocityComponent`, `readForwardComponent`,
// `readAngularVelocityComponent`, `readBodyVitalityComponent`, `readShieldVitalityComponent`)
// et `i21` lui-meme par `readAimingVectorComponent` — le meme code, valide en production, qui
// lit la visee dans un record porteur de position.

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// BipedAim est une lecture de VISEE issue d'un record qui ne porte AUCUNE position.
//
// Elle ne remplace pas `BipedPosition` : elle la COMPLETE la ou celle-ci ne peut rien dire,
// c'est-a-dire pendant qu'un joueur est a bord d'un vehicule (ou, plus generalement, chaque
// fois que le moteur ne re-emet pas la position dans le meme record que la visee).
type BipedAim struct {
	// Slot est l'identifiant bas (13 bits) de l'entite — meme convention que BipedPosition.
	Slot uint32
	// Chunk / PacketIndex localisent la lecture dans le film.
	Chunk, PacketIndex int
	// TimestampUS est l'horodatage du paquet porteur (horloge du film), donc la MEME horloge
	// que BipedPosition, types.VehicleEvent et FireEvent.
	TimestampUS uint64
	// YawRaw / PitchRaw sont les deux scalaires bruts d'`i21` : R(12) puis R(11).
	YawRaw, PitchRaw uint32
}

// viseeLue est une visee et le premier bit de l en-tete de son record dans son payload : la place
// que la regle de la passe derriere la marche juge, et son rang dans le flux.
type viseeLue struct {
	BipedAim
	bit int
}

// indexDeLaVisee est l index du composant de visee (`unit-desired-aiming-vector`) dans l archetype
// bipede ; composantsAvantLaVisee, le masque des index que la passe de la visee seule traverse avant
// lui ([aimOnlyCursorToI21]).
const (
	indexDeLaVisee         = 21
	composantsAvantLaVisee = 1<<1 | 1<<2 | 1<<3 | 1<<4 | 1<<5
)

// AimHeadingDeg rend le cap de visee en degres, avec la convention MESUREE et deja publiee de
// `BipedPosition.AimHeadingDeg` — le meme calcul, un seul detenteur.
func (a BipedAim) AimHeadingDeg() float32 { return aimHeadingDegFromRaw(a.YawRaw) }

// AimPitchDeg rend l'elevation de visee en degres, avec la convention MESUREE et deja publiee
// de `BipedPosition.AimPitchDeg` (reserve comprise) — le meme calcul, un seul detenteur.
func (a BipedAim) AimPitchDeg() float32 { return aimPitchDegFromRaw(a.PitchRaw) }

// ScanFilmBipedAimOnly est l'ENVELOPPE D2, HORS PRODUCTION : elle charge le film puis appelle
// [ScanBipedAimOnly]. La cuisson passe un contexte deja ouvert (une seule decompression).
func ScanFilmBipedAimOnly(dir string) ([]BipedAim, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanBipedAimOnly(contexteDeBobine(film))
}

// ScanBipedAimOnly decode, sur tous les chunks d'un film DEJA CHARGE, les lectures de visee des
// records bipedes qui NE PORTENT PAS de position. La bande de slots est celle des images-cles
// `ti=35`, exactement comme [ScanBipedPositions] — et c'est la MEME, relevee une seule fois par
// le contexte du film (lot 2 de PLAN_CUISSON_PERF).
//
// LA MARCHE D ABORD : les visees des records que la marche des trames a lus
// ([canalDesLecturesBipedes.noterLaVisee]), puis celles que la passe bit a bit rend derriere elle, la
// ou la fermeture de la trame ne prouve pas et pour un slot que la marche n y a pas lu
// ([rendParLAncrage]). Sans registre, la passe seule, sur chaque payload en entier.
func ScanBipedAimOnly(fc *FilmContext) ([]BipedAim, error) {
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return nil, ErrNoFilmChunk
	}
	band := fc.BipedSlots()
	if band.Count() == 0 {
		return nil, fmt.Errorf("aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	// Lot J5.2 : les generations vivantes du film, pas la seule 1 ; DATEES a l instant du paquet
	// (lot R2-bis) : la visee d un corps n est pas lue avant son record de creation.
	var out []viseeLue
	var trames map[paquetDuFlux]trameDuCanal
	fin := payloadEntier
	if lu, err := fc.lecturesBipedes(); err == nil {
		out, trames, fin = slices.Clone(lu.visees), lu.trames, finSelonLesTrames(lu.trames)
	}
	marche := len(out)
	read := 0
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		read++
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			gens := fc.GenerationsVivantesA(pk.TimestampUS)
			pay := pk.Payload(data)
			t, vu := trames[paquetDuFlux{c, pk.Index}]
			for _, a := range scanBipedAimRecordsJusqua(pay, band, gens, fc.ContexteDeLecture(),
				fin(c, pk.Index, len(pay)*8)) {
				if !rendParLAncrage(t, vu, a.Slot, BipedTypeIndex, a.bit) {
					continue
				}
				a.Chunk, a.PacketIndex, a.TimestampUS = c, pk.Index, pk.TimestampUS
				out = append(out, a)
			}
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	if marche > 0 && marche < len(out) {
		rangerLesViseesDansLeFlux(out, nums)
	}
	return sansLeurBit(out), nil
}

// sansLeurBit rend les visees de `v`, dans leur ordre.
func sansLeurBit(v []viseeLue) []BipedAim {
	if len(v) == 0 {
		return nil
	}
	out := make([]BipedAim, len(v))
	for i := range v {
		out[i] = v[i].BipedAim
	}
	return out
}

// rangerLesViseesDansLeFlux trie les visees dans l ordre du flux : rang du chunk, rang du paquet, bit
// de l en-tete.
func rangerLesViseesDansLeFlux(v []viseeLue, chunks []int) {
	rang := make(map[int]int, len(chunks))
	for i, c := range chunks {
		rang[c] = i
	}
	slices.SortStableFunc(v, func(a, b viseeLue) int {
		return cmp.Or(cmp.Compare(rang[a.Chunk], rang[b.Chunk]), cmp.Compare(a.PacketIndex, b.PacketIndex),
			cmp.Compare(a.bit, b.bit))
	})
}

// ScanBipedAimRecords balaie un payload de paquet delta bit a bit et renvoie les visees des
// records bipedes SANS position. PUR (aucune I/O) : c'est le coeur testable. Les champs
// Chunk/PacketIndex/TimestampUS sont laisses a zero (remplis par l'appelant). `gens` est le filtre de
// generation du handle (lot J5.2) ; nil : tous les slots au repli nomme (generation 1).
func ScanBipedAimRecords(payload []byte, slots SlotBand, gens *GenerationsVivantes, ctx ContexteDeLecture) []BipedAim {
	return sansLeurBit(scanBipedAimRecordsJusqua(payload, slots, gens, ctx, len(payload)*8))
}

// scanBipedAimRecordsJusqua est [ScanBipedAimRecords] arrete au bit `fin` : aucun en-tete n est cherche
// a ce bit ni au-dela.
func scanBipedAimRecordsJusqua(payload []byte, slots SlotBand, gens *GenerationsVivantes, ctx ContexteDeLecture,
	fin int) []viseeLue {
	total := len(payload) * 8
	var out []viseeLue
	// UN SEUL lecteur de bits pour tout le payload : les composants de vitalite traverses avant
	// `i21` le repositionnent par `SetBitPos`, la ou ils en allouaient un chacun PAR CANDIDAT.
	br := LecteurSur(payload)
	br.PoserContexte(ctx)
	for p := 0; p+bipedHeaderBits <= total && p < fin; {
		at, slot, ok := matchAimOnlyRecord(br, payload, p, slots, gens)
		if !ok {
			p++
			continue
		}
		var d componentDirs
		readAimingVectorComponent(payload, at, total, &d)
		if !d.HasYaw {
			p++
			continue
		}
		out = append(out, viseeLue{BipedAim: BipedAim{Slot: slot, YawRaw: d.YawRaw, PitchRaw: d.PitchRaw}, bit: p})
		p = at + 1 + aimYawBits + aimPitchBits // pas de re-scan chevauchant
	}
	return out
}

// matchAimOnlyRecord teste l'en-tete d'un record delta bipede dont le masque NE COMMENCE PAS
// par `i0`, consomme les composants qui precedent `i21`, et rend le bit ou commence `i21`.
//
// La seule difference avec `matchBipedHeaderRaw` est l'ABSENCE de la contrainte « premier index
// du masque = 0 » ; tout le reste (prefixe, slot, generation vivante, couple de zeros, compteur) est
// identique, et c'est ce qui rend le plancher de faux positifs mesurable par une bande fantome.
func matchAimOnlyRecord(br *Lecteur, pay []byte, p int, slots SlotBand, gens *GenerationsVivantes) (int, uint32, bool) {
	total := len(pay) * 8
	if uint32(source.BitsStricts(pay, p, 1)) != 1 {
		return 0, 0, false
	}
	h := LireHandleDelta(pay, p)
	if !slots.Has(h.Slot) {
		return 0, 0, false
	}
	if !gens.Accepte(h) { // generation vivante du slot, ou repli nomme (generations_vivantes.go)
		return 0, 0, false
	}
	slot := h.Slot
	if uint32(source.BitsStricts(pay, p+16, 2)) != 0 { // 14e bit d'id + selecteur de baseline
		return 0, 0, false
	}
	mc := int(uint32(source.BitsStricts(pay, p+18, 3)))
	if mc < 1 || mc > bipedMaxMaskCnt {
		return 0, 0, false
	}
	if p+bipedHeaderBits+bipedIndexBits*mc > total {
		return 0, 0, false
	}
	idx, ok := ascendingMask(pay, p+bipedHeaderBits, mc)
	if !ok || idx[0] == 0 {
		return 0, 0, false // un masque qui declare i0 releve de ScanBipedRecords
	}
	at, ok := aimOnlyCursorToI21(br, pay, p+bipedHeaderBits+bipedIndexBits*mc, total, idx)
	if !ok {
		return 0, 0, false
	}
	return at, slot, true
}

// ascendingMask valide une liste d'index de composants STRICTEMENT croissants, sans exiger que
// le premier vaille zero (c'est `ascendingFromZero` moins cette seule contrainte).
func ascendingMask(pay []byte, at, count int) ([]int, bool) {
	out := make([]int, 0, count)
	prev := -1
	for k := range count {
		idx := int(uint32(source.BitsStricts(pay, at+bipedIndexBits*k, bipedIndexBits)))
		if idx <= prev {
			return nil, false
		}
		prev = idx
		out = append(out, idx)
	}
	return out, true
}

// aimOnlyCursorToI21 consomme les composants qui PRECEDENT `i21` dans le masque et rend le bit
// ou `i21` commence. Chaque composant est consomme par SON detenteur existant : aucune largeur
// n'est reecrite ici. Un index non modelise avant `i21` arrete la lecture (le curseur ne serait
// plus fiable) — c'est la meme regle que `scanRecordDirs`.
func aimOnlyCursorToI21(br *Lecteur, pay []byte, at, total int, idx []int) (int, bool) {
	var d componentDirs
	var v componentVitals
	for _, id := range idx {
		if id == indexDeLaVisee {
			return at, true
		}
		var ok bool
		switch id {
		case 1:
			at, ok = readVelocityComponent(pay, at, total, &d)
		case 2:
			at, ok = readForwardComponent(pay, at, total, &d)
		case 3:
			at, ok = readAngularVelocityComponent(pay, at, total)
		case 4:
			at, ok = readBodyVitalityComponent(br, at, total, &v)
		case 5:
			at, ok = readShieldVitalityComponent(br, at, total, &v)
		default:
			return at, false
		}
		if !ok {
			return at, false
		}
	}
	return at, false // i21 absent du masque
}
