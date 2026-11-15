package grammar

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// entreeSynthetique fabrique un paquet d'image-cle portant UN record du tableau des scores dont le composant d'entree
// vaut `valeur` et `genre`, a `debut` bits du paquet, dans l'etat `etat`.
func entreeSynthetique(slot int32, valeur uint64, genre uint64, etat lecture.Etat, bits uint32) (*lecture.Paquet, *lecture.Record) {
	w := &bitWriter{}
	w.bits(0, 16)
	w.bits(valeur, 32)
	w.bits(genre, 8)
	w.bits(0, 16)
	p := &lecture.Paquet{
		TS:      4_000_000,
		Payload: w.buf,
		Comps:   []lecture.Composant{{Index: 0, Etat: lecture.EtatDelimite, Debut: 0, Bits: 16}, {Index: 1, Etat: etat, Debut: 16, Bits: bits}},
	}
	p.Records = []lecture.Record{{TI: statborgTypeIndex, Comps: [2]uint32{0, 2}}}
	p.Records[0].Vie.Slot = uint32(slot)
	return p, &p.Records[0]
}

// TestLeCanalDesEntreesLitLeJoueurLEquipeEtLeVacant : le genre 0 donne l'identifiant moteur du joueur (seize bits de
// poids fort), le genre 1 le numero de l'equipe (seize bits de poids faible), un genre inconnu est une entite vacante,
// et une occurrence non traversee ou trop courte n'est pas lue.
func TestLeCanalDesEntreesLitLeJoueurLEquipeEtLeVacant(t *testing.T) {
	arch := Archetype{Components: []string{"statborg-current-round-value-stat-component", statborgEntryComponent}}
	cas := []struct {
		nom           string
		valeur, genre uint64
		etat          lecture.Etat
		bits          uint32
		want          types.StatborgEntry
		lu, vacant    bool
		unreached     bool
	}{
		{"joueur", 13 << 16, 0, lecture.EtatInterprete, 40, types.StatborgEntry{Slot: 9, TS: 4_000_000, Kind: 0, ID: 13}, true, false, false},
		{"joueur, poids faible ignore", 13<<16 | 0xabcd, 0, lecture.EtatInterprete, 40, types.StatborgEntry{Slot: 9, TS: 4_000_000, Kind: 0, ID: 13}, true, false, false},
		{"equipe", 0xffff0001, 1, lecture.EtatInterprete, 40, types.StatborgEntry{Slot: 9, TS: 4_000_000, Kind: 1, ID: 1}, true, false, false},
		{"genre inconnu", 7 << 16, 2, lecture.EtatInterprete, 40, types.StatborgEntry{}, false, true, false},
		{"non traversee", 13 << 16, 0, lecture.EtatInfranchissable, 0, types.StatborgEntry{}, false, false, true},
		{"trop courte", 13 << 16, 0, lecture.EtatInterprete, 39, types.StatborgEntry{}, false, false, true},
	}
	for _, c := range cas {
		p, _ := entreeSynthetique(9, c.valeur, c.genre, c.etat, c.bits)
		canal := &canalDesEntreesDuStatborg{arch: arch}
		canal.ImageCle(p, nil)
		switch {
		case c.lu && (len(canal.lus) != 1 || canal.lus[0] != c.want || canal.st.Read != 1):
			t.Errorf("%s : %+v, lu %d, attendu %+v", c.nom, canal.lus, canal.st.Read, c.want)
		case c.vacant && (len(canal.lus) != 0 || canal.st.Vacant != 1):
			t.Errorf("%s : %+v, vacant %d", c.nom, canal.lus, canal.st.Vacant)
		case c.unreached && (len(canal.lus) != 0 || canal.st.Unreached != 1):
			t.Errorf("%s : %+v, non atteint %d", c.nom, canal.lus, canal.st.Unreached)
		}
		if canal.st.Records != 1 {
			t.Errorf("%s : %d record(s) compte(s), attendu 1", c.nom, canal.st.Records)
		}
	}
}

// TestLeCanalDesEntreesIgnoreLesAutresArchetypes : un record d'un autre archetype n'est ni lu ni compte.
func TestLeCanalDesEntreesIgnoreLesAutresArchetypes(t *testing.T) {
	p, _ := entreeSynthetique(9, 13<<16, 0, lecture.EtatInterprete, 40)
	p.Records[0].TI = managedPlayerTypeIndex
	canal := &canalDesEntreesDuStatborg{arch: Archetype{Components: []string{"a", statborgEntryComponent}}}
	canal.ImageCle(p, nil)
	if canal.st.Records != 0 || len(canal.lus) != 0 {
		t.Errorf("record d'un autre archetype pris en compte : %+v", canal.st)
	}
}

// TestLeCanalDesEntreesLitLesIssuesDesManches : le masque des manches jouees leve le bit des champs de deux bits non nuls
// des issues, d'une entite a l'autre il s'unit, et un composant non traverse ne leve rien.
func TestLeCanalDesEntreesLitLesIssuesDesManches(t *testing.T) {
	arch := Archetype{Components: []string{"v", statborgEntryComponent, statborgOutcomesComponent}}
	paquet := func(issues map[int]uint64, etat lecture.Etat) *lecture.Paquet {
		w := &bitWriter{}
		w.bits(0, 8)
		for m := range 32 {
			w.bits(issues[m], 2)
		}
		w.bits(0, 8)
		p := &lecture.Paquet{Payload: w.buf, Comps: []lecture.Composant{{Index: 2, Etat: etat, Debut: 8, Bits: 64}}}
		p.Records = []lecture.Record{{TI: statborgTypeIndex, Comps: [2]uint32{0, 1}}}
		return p
	}
	canal := &canalDesEntreesDuStatborg{arch: arch}
	canal.ImageCle(paquet(map[int]uint64{0: 3, 2: 1}, lecture.EtatInterprete), nil)
	if canal.st.DecidedRounds != 0b101 {
		t.Errorf("masque = %b, attendu 101", canal.st.DecidedRounds)
	}
	canal.ImageCle(paquet(map[int]uint64{1: 2, 31: 1}, lecture.EtatInterprete), nil)
	if canal.st.DecidedRounds != 1<<31|0b111 {
		t.Errorf("masque uni = %b", canal.st.DecidedRounds)
	}
	vide := &canalDesEntreesDuStatborg{arch: arch}
	vide.ImageCle(paquet(map[int]uint64{0: 3}, lecture.EtatInfranchissable), nil)
	if vide.st.DecidedRounds != 0 {
		t.Errorf("composant non traverse : masque %b", vide.st.DecidedRounds)
	}
}
