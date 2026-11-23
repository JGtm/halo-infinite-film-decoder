package replay

import "testing"

// zoneAvecProprietaires construit un etat de zone dont les intervalles sont tenus par `camps` aux
// images `debuts` (un intervalle par camp, chacun finissant a l'image suivante).
func zoneAvecProprietaires(ref, rang int, debuts, camps []int) ZoneState {
	r := rang
	z := ZoneState{ZoneRef: ref, LetterRank: &r}
	for i, d := range debuts {
		c := camps[i]
		z.Spans = append(z.Spans, ZoneSpan{T0: d, T1: d + 5, Owner: &c})
	}
	return z
}

func marqueur(lettre int, instants, camps []int) ZoneLetterMarker {
	m := ZoneLetterMarker{Letter: lettre}
	for i, t := range instants {
		m.Changes = append(m.Changes, ZoneLetterChange{TimeMS: t, Team: camps[i]})
	}
	return m
}

// Trois zones aux historiques distincts ; les marqueurs repliquent les memes changements avec un
// decalage d'horloge de 40 s et quelques centaines de ms de bruit.
func troisZones() []ZoneState {
	return []ZoneState{
		zoneAvecProprietaires(0, 2, []int{100, 500, 900}, []int{0, 1, 0}),
		zoneAvecProprietaires(1, 0, []int{200, 600}, []int{1, 0}),
		zoneAvecProprietaires(2, 1, []int{300, 700, 1100, 1300}, []int{0, 1, 0, 1}),
	}
}

func TestZoneLettersControlAccord(t *testing.T) {
	const frame, off = 100, 40000
	ms := []ZoneLetterMarker{
		marqueur(2, []int{10000 + off + 300, 50000 + off - 200, 90000 + off}, []int{0, 1, 0}),
		marqueur(0, []int{20000 + off, 60000 + off + 400}, []int{1, 0}),
		marqueur(1, []int{30000 + off, 70000 + off, 110000 + off - 100, 130000 + off}, []int{0, 1, 0, 1}),
	}
	got := zoneLettersControlOf(ms, troisZones(), frame)
	if got.Read != 3 || got.Agreed != 3 || got.Unmatched != 0 {
		t.Fatalf("accord attendu 3/3 sans rattachement manque, obtenu %+v", got)
	}
}

func TestZoneLettersControlDesaccordSeCompte(t *testing.T) {
	const frame, off = 100, 40000
	// Le marqueur de la zone de rang 0 annonce la lettre B : desaccord ; les deux autres s'accordent.
	// Deux lettres distinctes sont lues (B, C), deux accords, et le rang publie n'est jamais modifie.
	zones := troisZones()
	ms := []ZoneLetterMarker{
		marqueur(1, []int{20000 + off, 60000 + off}, []int{1, 0}),
		marqueur(2, []int{10000 + off, 50000 + off, 90000 + off}, []int{0, 1, 0}),
		marqueur(1, []int{30000 + off, 70000 + off, 110000 + off, 130000 + off}, []int{0, 1, 0, 1}),
	}
	got := zoneLettersControlOf(ms, zones, frame)
	if got.Read != 2 || got.Agreed != 2 || got.Unmatched != 0 {
		t.Fatalf("attendu 2 lettres lues, 2 accords sur 3 marqueurs, obtenu %+v", got)
	}
	if *zones[1].LetterRank != 0 {
		t.Fatalf("le controle ne doit jamais modifier le rang publie")
	}
	// Les trois lettres decalees d'un cran : lues, jamais accordees.
	ms[0].Letter, ms[1].Letter, ms[2].Letter = 1, 0, 2
	got = zoneLettersControlOf(ms, zones, frame)
	if got.Agreed != 0 || got.Read != 3 || got.Unmatched != 0 {
		t.Fatalf("lettres fausses : lues mais non accordees, obtenu %+v", got)
	}
}

func TestZoneLettersControlBijectionExigee(t *testing.T) {
	const frame, off = 100, 40000
	// Deux marqueurs rejouent la meme zone : aucun des deux n'est rattache.
	ms := []ZoneLetterMarker{
		marqueur(0, []int{20000 + off, 60000 + off}, []int{1, 0}),
		marqueur(1, []int{20000 + off, 60000 + off}, []int{1, 0}),
	}
	got := zoneLettersControlOf(ms, troisZones(), frame)
	if got.Read != 2 || got.Agreed != 0 || got.Unmatched != 2 {
		t.Fatalf("une zone reclamee deux fois ne rattache personne, obtenu %+v", got)
	}
}

func TestZoneLettersControlEgaliteEtSilence(t *testing.T) {
	const frame = 100
	// Deux zones au meme historique : la meilleure n'est pas unique.
	z := []ZoneState{
		zoneAvecProprietaires(0, 0, []int{100, 500}, []int{0, 1}),
		zoneAvecProprietaires(1, 1, []int{100, 500}, []int{0, 1}),
	}
	m := []ZoneLetterMarker{marqueur(0, []int{10000, 50000}, []int{0, 1})}
	if got := zoneLettersControlOf(m, z, frame); got.Agreed != 0 || got.Unmatched != 1 || got.Read != 1 {
		t.Fatalf("egalite : lue mais non rattachee, obtenu %+v", got)
	}
	// Marqueur sans lettre, sans changement, sans zone : rien n'est compte, rien ne plante.
	if got := zoneLettersControlOf([]ZoneLetterMarker{{Letter: -1}}, z, frame); got != (zoneLettersControl{}) {
		t.Fatalf("marqueur sans lettre : attendu zero, obtenu %+v", got)
	}
	if got := zoneLettersControlOf(m, nil, frame); got.Read != 1 || got.Unmatched != 1 {
		t.Fatalf("aucune zone : lue, non rattachee, obtenu %+v", got)
	}
	if got := zoneLettersControlOf([]ZoneLetterMarker{marqueur(0, nil, nil)}, z, frame); got.Read != 1 || got.Unmatched != 1 {
		t.Fatalf("marqueur sans changement : lu, non rattache, obtenu %+v", got)
	}
}

func TestZoneLettersControlIndependantDeLOrdre(t *testing.T) {
	const frame, off = 100, -25000
	ms := []ZoneLetterMarker{
		marqueur(2, []int{10000 + off, 50000 + off, 90000 + off}, []int{0, 1, 0}),
		marqueur(0, []int{20000 + off, 60000 + off}, []int{1, 0}),
		marqueur(1, []int{30000 + off, 70000 + off, 110000 + off, 130000 + off}, []int{0, 1, 0, 1}),
	}
	ref := zoneLettersControlOf(ms, troisZones(), frame)
	rev := []ZoneLetterMarker{ms[2], ms[1], ms[0]}
	if got := zoneLettersControlOf(rev, troisZones(), frame); got != ref {
		t.Fatalf("l'ordre des marqueurs change le resultat : %+v contre %+v", got, ref)
	}
	if ref.Agreed != 3 {
		t.Fatalf("decalage negatif : accord attendu 3/3, obtenu %+v", ref)
	}
}

func TestFloorDiv(t *testing.T) {
	for _, c := range [][3]int{{7, 2, 3}, {-7, 2, -4}, {-4, 2, -2}, {0, 5, 0}, {-1, 1500, -1}} {
		if got := floorDiv(c[0], c[1]); got != c[2] {
			t.Errorf("floorDiv(%d,%d) = %d, attendu %d", c[0], c[1], got, c[2])
		}
	}
}
