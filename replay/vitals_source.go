package replay

// vitals_source.go : CE QUE LA MARCHE DES TRAMES TRANSPORTE DE LA SANTE ET DU BOUCLIER, et que les faits portent.
//
// Le canal des vitalites ([grammar.LectureVitale]) rend une lecture par trame et par bipede : le jeu reecrit le
// composant de bouclier a chaque cran de recharge et tant qu il est actif, et 92 % des lectures repetent la valeur
// precedente. Ce fichier ne garde que les CHANGEMENTS, par vie (le slot et sa generation), sous la forme d un etat
// complet : une lecture partielle est completee de la valeur de l autre composant, comme le fait [registreDesViesDe].
// Le calcul des points de degat ne passe pas par ici : il lit toutes les lectures, avant la reduction.

import "github.com/JGtm/halo-infinite-film-decoder/internal/grammar"

// VitalRead est l etat vital d une vie a un instant ou il CHANGE, ou sa naissance.
type VitalRead struct {
	// TimestampUS est l horloge du film du paquet porteur, sur celle de BipedPosition.TimestampUS.
	TimestampUS uint64
	// Slot et Gen designent la vie : la paire, comme [types.LifeKey]. Un slot recycle porte une generation neuve.
	Slot uint32
	Gen  uint8
	// Birth : la vie commence ici, pleine ; Health et Shield valent alors 1.
	Birth bool
	// Health est la sante de 0 a 1 (la zone negative du film, dont le sens n est pas etabli, se replie sur 0) ;
	// Shield est le bouclier de 0 a 4, 1 etant le bouclier plein.
	Health, Shield float32
}

// VitalStats sont les denominateurs du calque des vitalites.
type VitalStats struct {
	// Scanned : la marche des trames a ete jouee ; faux = rien n a ete lu, ce qui n est pas « aucun changement ».
	Scanned bool
	// Read : les lectures de sante ou de bouclier du canal, naissances exclues. Births : les naissances lues.
	Read, Births int
	// Unchanged : les lectures dont l etat egale celui que la vie avait deja, ecartees avant d etre transportees.
	Unchanged int
}

// etatVitalPlein est l etat d une vie a sa naissance.
var etatVitalPlein = [2]float32{1, 1}

// reduireLesVitalites ne garde des lectures du canal que les naissances et les changements d etat par vie, dans
// l ordre du flux. Une vie sans naissance lue part pleine, comme dans [registreDesViesDe].
func reduireLesVitalites(lectures []grammar.LectureVitale) ([]VitalRead, VitalStats) {
	st := VitalStats{Scanned: true}
	etats := map[cleDeVie][2]float32{}
	var out []VitalRead
	for _, l := range lectures {
		k := cleDeVie{l.Slot, l.Gen}
		if l.Creation {
			st.Births++
			etats[k] = etatVitalPlein
			out = append(out, VitalRead{TimestampUS: l.TS, Slot: l.Slot, Gen: l.Gen, Birth: true, Health: 1, Shield: 1})
			continue
		}
		st.Read++
		avant, connu := etats[k]
		if !connu {
			avant = etatVitalPlein
		}
		apres := avant
		if l.ASante {
			apres[0] = grammar.HealthFraction(l.Sante)
		}
		if l.ABouclier {
			apres[1] = l.Bouclier
		}
		if apres == avant {
			st.Unchanged++
			continue
		}
		etats[k] = apres
		out = append(out, VitalRead{TimestampUS: l.TS, Slot: l.Slot, Gen: l.Gen, Health: apres[0], Shield: apres[1]})
	}
	return out, st
}
