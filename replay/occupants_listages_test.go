package replay

// occupants_listages_test.go — LA PRESENCE D UN HUMAIN, DATEE PAR LES PAQUETS DE TYPE 8.
//
//	L-EXACT     un humain liste de la frame 5 a la veille de la frame 40 est present de 5 a 39, sans
//	            prolongation d affichage : le listage date l arrivee et le depart ;
//	L-VIES      une vie hors de toute plage ouvre ou prolonge la presence (un corps qui bouge prouve
//	            la presence de son occupant) ;
//	L-RETOUR    un humain qui part puis revient porte deux intervalles, et deux plages qui se touchent
//	            (changement de siege) n en font qu un ;
//	L-SANS      sans plage, la presence vient des entites et des vies, comme avant ;
//	L-BOT       un bot n est jamais date par les plages de type 8.

import (
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

func entreesAvecListages(scan grammar.PlayerEntityScan, ls ...types.RosterListing) entreesDesOccupants {
	in := entreesDeTest(scan)
	in.listages = listagesParXUID(ls)
	return in
}

func TestListagesDatentLaPresenceDUnHumain(t *testing.T) {
	// Les frames valent 100 ms : 500 ms -> frame 5, 4 000 ms -> frame 40.
	in := entreesAvecListages(grammar.PlayerEntityScan{},
		types.RosterListing{XUID: 100, Slot: 3, FromUS: 500_000, ToUS: 4_000_000})
	occ := lierLesOccupants([]RosterEntry{entree(3, "100")}, []Track{vieDe("100", 10, 20)}, in)
	p := occ.parEntree[0].presence
	if len(p) != 1 || p[0] != (intervalleDePresence{de: 5, a: 39, aMax: 39}) || !occ.parEntree[0].lue {
		t.Fatalf("presence %+v : attendu [5, 39] exacte et lue", p)
	}
}

func TestListagesUneVieHorsPlageProlongeLaPresence(t *testing.T) {
	in := entreesAvecListages(grammar.PlayerEntityScan{},
		types.RosterListing{XUID: 100, Slot: 3, FromUS: 500_000, ToUS: 4_000_000})
	occ := lierLesOccupants([]RosterEntry{entree(3, "100")}, []Track{vieDe("100", 35, 60)}, in)
	p := occ.parEntree[0].presence
	if len(p) != 1 || p[0] != (intervalleDePresence{de: 5, a: 60, aMax: 60}) {
		t.Fatalf("presence %+v : la vie qui touche la plage la prolonge jusqu a sa fin", p)
	}
}

func TestListagesRetourEtChangementDeSiege(t *testing.T) {
	in := entreesAvecListages(grammar.PlayerEntityScan{},
		types.RosterListing{XUID: 100, Slot: 3, FromUS: 0, ToUS: 1_000_000},
		types.RosterListing{XUID: 100, Slot: 6, FromUS: 1_000_000, ToUS: 2_000_000},
		types.RosterListing{XUID: 100, Slot: 6, FromUS: 5_000_000})
	occ := lierLesOccupants([]RosterEntry{entree(3, "100")}, nil, in)
	p := occ.parEntree[0].presence
	if len(p) != 2 || p[0] != (intervalleDePresence{de: 0, a: 19, aMax: 19}) ||
		p[1].de != 50 || p[1].a != siegeFrames-1 {
		t.Fatalf("presence %+v : attendu [0, 19] puis [50, fin] (les deux premieres plages se touchent)", p)
	}
}

func TestListagesAbsentsLaPresenceVientDesEntites(t *testing.T) {
	scan := scanDeTest([]int{10, 50, 90}, grammar.PlayerEntity{Slot: 7, Index: 4, Team: 0, FirstKF: 1,
		LastKF: 1, Seen: 1})
	in := entreesAvecListages(scan, types.RosterListing{XUID: 999, Slot: 4, FromUS: 0})
	occ := lierLesOccupants([]RosterEntry{entree(4, "100")}, nil, in)
	p := occ.parEntree[0].presence
	if len(p) != 1 || p[0] != (intervalleDePresence{de: 50, a: 50, aMax: 89}) {
		t.Fatalf("presence %+v : sans plage pour ce xuid, l entite date la presence comme avant", p)
	}
}

func TestListagesNeDatentPasUnBot(t *testing.T) {
	in := entreesAvecListages(grammar.PlayerEntityScan{}, types.RosterListing{XUID: 0, Slot: 8, FromUS: 0})
	bot := RosterEntry{FilmIndex: 8, Name: "343 X [bot]", Bot: true, XUID: "0"}
	if got := presencesListees(bot, in); got != nil {
		t.Fatalf("presences %+v : un bot n est jamais date par le type 8", got)
	}
}

// TestEntiteDeLIndexSuitLeListage (L-ENTITE) : un humain liste sous un index pendant une plage n est lie qu a
// l entite de cet index lue pendant cette plage ; l entite lue apres son depart (l occupant suivant du siege) ne
// lui est pas attribuee, meme quand l une de ses vies tombe dans sa fenetre. Une lecture sans siege ne dit rien.
func TestEntiteDeLIndexSuitLeListage(t *testing.T) {
	// Images-cles aux frames 10, 50, 90 (100 ms) : 1 s, 5 s, 9 s.
	scan := scanDeTest([]int{10, 50, 90},
		grammar.PlayerEntity{Slot: 7, Index: 1, Team: 0, FirstKF: 0, LastKF: 0, Seen: 1},
		grammar.PlayerEntity{Slot: 8, Index: 1, Team: 1, FirstKF: 2, LastKF: 2, Seen: 1})
	roster := []RosterEntry{entree(1, "100")}
	tracks := []Track{vieDe("100", 5, 12), vieDe("100", 85, 95)}
	in := entreesAvecListages(scan, types.RosterListing{XUID: 100, Slot: 1, FromUS: 0, ToUS: 3_000_000})
	occ := lierLesOccupants(roster, tracks, in)
	if got := occ.parEntree[0].entites; len(got) != 1 || got[0] != 0 {
		t.Fatalf("entites liees %v : seule l entite lue pendant la plage est a l humain", got)
	}
	if eq := occ.parEntree[0].equipe; eq == nil || *eq != 0 {
		t.Fatalf("equipe %v : celle de son entite", deref(eq))
	}
	sans := entreesAvecListages(scan, types.RosterListing{XUID: 100, Slot: -1, FromUS: 0, ToUS: 3_000_000})
	occ = lierLesOccupants(roster, tracks, sans)
	if len(occ.parEntree[0].entites) != 2 {
		t.Fatalf("entites liees %v : sans siege lu, ses vies designent ses entites comme avant", occ.parEntree[0].entites)
	}
}

// TestListagesParIndexDeLEntree (L-INDEX) : un humain qui revient sous un autre siege a une entree par index ; chacune
// ne prend que les plages de son index, et un index que aucune plage ne porte garde la lecture par les entites.
func TestListagesParIndexDeLEntree(t *testing.T) {
	in := entreesAvecListages(grammar.PlayerEntityScan{},
		types.RosterListing{XUID: 100, Slot: 3, FromUS: 0, ToUS: 1_000_000},
		types.RosterListing{XUID: 100, Slot: 9, FromUS: 5_000_000})
	in.multiples = xuidsSousPlusieursIndex([]RosterEntry{entree(3, "100"), entree(9, "100")})
	if got := presencesListees(entree(3, "100"), in); len(got) != 1 || got[0].a != 9 {
		t.Errorf("entree d'index 3 : %+v, attendu l'intervalle de sa plage", got)
	}
	if got := presencesListees(entree(9, "100"), in); len(got) != 1 || got[0].de != 50 {
		t.Errorf("entree d'index 9 : %+v, attendu l'intervalle de sa plage", got)
	}
	if got := presencesListees(entree(5, "100"), in); got != nil {
		t.Errorf("entree d'index 5 : %+v, aucune plage ne la porte", got)
	}
}
