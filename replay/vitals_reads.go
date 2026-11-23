package replay

// vitals_reads.go : LA SANTE ET LE BOUCLIER DE CHAQUE JOUEUR, PAR VIE, A CHAQUE CHANGEMENT (`vitals[]`).
//
// CE QUE C EST. Le film ecrit la sante (composant i4) et le bouclier (i5) du bipede a chaque trame ou l un d eux
// change, sans attendre une image-cle : une ligne dit, pour une vie, l etat COMPLET (`sh` et `hp`) a l instant `ms`.
// Entre deux lignes, l etat est tenu. Les points de trajectoire (`tracks[].points[].sh` et `.hp`) en publient une
// partie, au premier echantillon de chaque case de frame et seulement quand le record de position les porte ;
// `damage[]` publie ce que chaque coup retire. Ce calque est la serie entiere dont ils sont des extraits.
//
// UNITES, CELLES DES POINTS DE TRAJECTOIRE ET DE `damage[]` POUR QU UN LECTEUR LES RECOUPE.
//   - `sh` est la fraction du bouclier plein : 1 = plein, au dela le surbouclier, jusqu a 4. Un point de trajectoire
//     l ecrete a 1 ; sous 1 les deux valeurs sont les memes (arrondies au millieme). Un bouclier en points vaut
//     `sh` fois damageeffects.BouclierMaximal.
//   - `hp` est la fraction de sante de 0 a 1, comme `points[].hp` ; en points : `hp` fois damageeffects.CorpsMaximal.
//   - `damage[]` publie des dixiemes de point retires au bouclier et au corps : la difference de deux lignes
//     consecutives d une vie, en points, retrouve leur somme a un cran pres (un cran de bouclier = 1/64).
//
// UNE VIE, PAS UN SIEGE. Les lignes se rangent par vie (le slot et sa generation) : un slot recycle repart pleine.
// Une vie nait pleine (`sh` 1, `hp` 1) et AUCUNE ligne ne la marque ; la premiere ligne est le premier changement.
// Le lecteur joint une ligne a sa piste par le slot et le temps, comme les autres calques par slot ; une ligne peut
// preceder le `startFrame` de sa piste (la creation du corps precede parfois d une seconde son premier point).
//
// LA MORT. La sante a zero est ecrite apres la fin de la piste du joueur (environ 38 ms), et un coup qui vide le
// bouclier et la sante a la fois n ecrit aucune sante intermediaire. Ce calque ne coupe donc pas une vie a la fin de sa
// piste : sa premiere ligne a `hp` 0 est publiee a son instant, marquee `dead`, et c est la derniere ligne de la vie.
// Une vie est morte quand sa sante est a zero, la definition du jeu : aucun seuil. La derniere sante vivante est la
// ligne qui precede, ou la valeur tenue de la vie.
//
// LA RAMPE. La recharge du bouclier est ecrite cran par cran a chaque trame ; la regle est dans vitals_ramp.go.
//
// BORNES. Rien avant l origine de l axe de frames ni apres sa fin, rien dont le slot n a aucune piste publiee, rien
// d une vie apres sa mort. L ordre est celui de l horloge du film puis le slot et la generation : il ne depend pas du
// nombre de fils.

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// Vital est l etat vital d une vie a l instant d un changement.
type Vital struct {
	// T est l index de frame, sur le meme axe que Point.T ; Ms l instant a la milliseconde depuis l origine de l axe.
	T  int `json:"t"`
	Ms int `json:"ms"`
	// Slot designe la Track concernee, donc une vie.
	Slot uint32 `json:"slot"`
	// Sh est le bouclier (1 = plein, jusqu a 4), Hp la sante (de 0 a 1), tous deux au millieme. Toujours presents : une
	// ligne dit l etat complet, et un bouclier ou une sante a zero est une valeur, pas une absence.
	Sh float32 `json:"sh"`
	Hp float32 `json:"hp"`
	// Dead marque la premiere ligne de la vie dont la sante est a zero : la mort, avec sa derniere sante.
	Dead bool `json:"dead,omitempty"`
}

// VitalCoverage dit ce que le calque a lu, publie et ecarte.
//
// L IDENTITE : read = unchanged + beforeOrigin + afterEnd + noTrack + afterDeath + ramp + published. `births` est hors de
// l identite : une naissance n est pas une lecture de valeur.
type VitalCoverage struct {
	// Read : les lectures de sante ou de bouclier du canal. Births : les naissances de vie lues.
	Read   int `json:"read"`
	Births int `json:"births,omitempty"`
	// Unchanged : lectures dont l etat egale celui que la vie avait deja (la recharge repete la valeur a chaque trame).
	Unchanged int `json:"unchanged,omitempty"`
	// BeforeOrigin et AfterEnd : lectures hors de l axe de frames du match. NoTrack : lectures dont le slot n a aucune piste
	// publiee. AfterDeath : lectures d une vie posterieures a la ligne de sa mort.
	BeforeOrigin int `json:"beforeOrigin,omitempty"`
	AfterEnd     int `json:"afterEnd,omitempty"`
	NoTrack      int `json:"noTrack,omitempty"`
	AfterDeath   int `json:"afterDeath,omitempty"`
	// Ramp : changements ecartes parce qu une rampe de recharge les porte (la ligne avant et la ligne apres suffisent a un
	// lecteur qui interpole, a un cran pres).
	Ramp int `json:"ramp,omitempty"`
	// Published : les lignes publiees. Deaths : parmi elles, les lignes `dead`.
	Published int `json:"published"`
	Deaths    int `json:"deaths,omitempty"`
}

// VitalsInput est ce que la marche des trames donne au calque.
type VitalsInput struct {
	Reads []VitalRead
	Stats VitalStats
}

// ligneVitale est une ligne retenue avec son instant exact, pour la regle de rampe.
type ligneVitale struct {
	v   Vital
	us  uint64
	vie cleDeVie
	// protegee : la ligne ne s ecarte jamais (cf. vitals_ramp.go).
	protegee bool
}

// retenirLesVitalites range les lectures sur l axe de frames et par vie : rien hors de l axe, rien sans piste, rien apres la
// mort de la vie. Elle rend les lignes par vie, dans l ordre de l horloge.
func retenirLesVitalites(in VitalsInput, publies map[uint32]bool, axe axeDeFrames, cov *VitalCoverage) map[cleDeVie][]ligneVitale {
	lectures := slices.Clone(in.Reads)
	slices.SortStableFunc(lectures, func(a, b VitalRead) int {
		return cmp.Or(cmp.Compare(a.TimestampUS, b.TimestampUS), cmp.Compare(a.Slot, b.Slot), cmp.Compare(a.Gen, b.Gen))
	})
	morts := map[cleDeVie]uint64{}
	vies := map[cleDeVie][]ligneVitale{}
	for _, r := range lectures {
		k := cleDeVie{r.Slot, r.Gen}
		if r.Birth {
			delete(morts, k)
			continue
		}
		t, refus := frameDeLAction(r.TimestampUS, axe)
		mort, estMorte := morts[k]
		switch {
		case refus == refusAvantOrigine || refus == refusSansOrigine:
			cov.BeforeOrigin++
		case refus == refusApresFin:
			cov.AfterEnd++
		case !publies[r.Slot]:
			cov.NoTrack++
		case estMorte && r.TimestampUS > mort:
			cov.AfterDeath++
		default:
			l := ligneVitale{us: r.TimestampUS, vie: k, v: Vital{T: t, Ms: axe.millisecondes(r.TimestampUS), Slot: r.Slot,
				Sh: *fractionForJSON(r.Shield), Hp: *fractionForJSON(r.Health)}}
			if r.Health <= 0 && !estMorte {
				l.v.Dead, l.protegee = true, true
				morts[k] = r.TimestampUS
			}
			vies[k] = append(vies[k], l)
		}
	}
	return vies
}

// buildVitals met les changements de sante et de bouclier sur l axe de frames. `degats` sont les messages de degat du film :
// la regle de rampe ne touche ni l instant d un coup ni la valeur juste avant. `avecRampe` faux publie tous les changements
// (la mesure du poids de la regle).
func buildVitals(in VitalsInput, degats []grammar.DegatLu, publies map[uint32]bool, axe axeDeFrames, avecRampe bool) ([]Vital, VitalCoverage) {
	cov := VitalCoverage{Read: in.Stats.Read, Births: in.Stats.Births, Unchanged: in.Stats.Unchanged}
	vies := retenirLesVitalites(in, publies, axe, &cov)
	coups := instantsDeCoups(degats)
	var out []ligneVitale
	for k, lignes := range vies {
		proteger(lignes, coups[k])
		if avecRampe {
			var ecartees int
			lignes, ecartees = ecarterLesRampes(lignes)
			cov.Ramp += ecartees
		}
		out = append(out, lignes...)
	}
	slices.SortFunc(out, func(a, b ligneVitale) int {
		return cmp.Or(cmp.Compare(a.us, b.us), cmp.Compare(a.v.Slot, b.v.Slot), cmp.Compare(a.vie.gen, b.vie.gen))
	})
	pub := make([]Vital, len(out))
	for i, l := range out {
		pub[i] = l.v
		cov.Deaths += unSiVrai(l.v.Dead)
	}
	cov.Published = len(pub)
	return pub, cov
}

// poserLesVitalites publie le calque et sa couverture, quand la marche des trames a ete jouee (une lecture absente
// n est pas « aucun changement »).
func (a *assemblage) poserLesVitalites() {
	if !a.opt.Vitals.Stats.Scanned || a.doc.Coverage == nil {
		return
	}
	vitals, cov := buildVitals(a.opt.Vitals, a.opt.Damages.Messages, publishedSlots(a.doc.Tracks),
		axeDeFrames{a.origin, a.step, a.doc.FrameCount}, true)
	a.doc.Vitals = vitals
	a.doc.Coverage.Vitals = &cov
	slog.InfoContext(a.ctx, "rejeu : vitalites", "lues", cov.Read, "naissances", cov.Births, "inchangees", cov.Unchanged,
		"avantOrigine", cov.BeforeOrigin, "apresFin", cov.AfterEnd, "sansPiste", cov.NoTrack, "apresMort", cov.AfterDeath,
		"rampes", cov.Ramp, "publiees", cov.Published, "morts", cov.Deaths)
}
