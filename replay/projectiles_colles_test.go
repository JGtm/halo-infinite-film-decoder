package replay

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// volDeProjectile est un vol court de la vie (slot, generation), a partir de `t0` (microsecondes), un echantillon par 20 ms, qui
// se termine en (16,9 + decalage, -15, 4,8).
func volDeProjectile(slot, gen uint32, t0 uint64, n int, decalageX ...float32) types.ProjectileTrack {
	tr := types.ProjectileTrack{Slot: slot, Gen: gen}
	for i := range n {
		tr.Pts = append(tr.Pts, types.ProjectileSample{TimestampUS: t0 + uint64(i)*20_000,
			X: 16 + float32(i)*0.1 + append(decalageX, 0)[0], Y: -15, Z: 4.8})
	}
	return tr
}

// collageDeTest est la lecture d un projectile (slot 1133, generation 1) colle au parent (slot 561, generation 1), noeud 22, aux
// trois valeurs d un record capture.
func collageDeTest(ts uint64) types.ProjectileAttachRead {
	return types.ProjectileAttachRead{TimestampUS: ts, Life: types.LifeKey{Slot: 1133, Gen: 1},
		Parent: types.LifeKey{Slot: 561, Gen: 1}, Node: 22, Pose: [3]uint32{32214, 33936, 33041}}
}

const (
	origineDuCollage = 1_000_000
	pasDuCollage     = 100_000
)

// assemblageDeCollage prepare l assemblage d une passe de collage : les vols donnes publies par buildProjectiles, un joueur au
// slot 561 de la frame 0 a la 40, un vehicule au slot 700 (generation 2) de la frame 0 a la 30, et un axe de 40 frames.
func assemblageDeCollage(vols []types.ProjectileTrack, lectures []types.ProjectileAttachRead,
	detonations ...types.ProjectileDetonation) *assemblage {
	detos := append(detonations, types.ProjectileDetonation{TimestampUS: 3_200_000, HasPosition: true, X: 16.9, Y: -15, Z: 4.8})
	publies, parPiste, _ := buildProjectiles(vols, origineDuCollage, pasDuCollage, detos, nil)
	a := &assemblage{origin: origineDuCollage, step: pasDuCollage, projParPiste: parPiste,
		projCov: &ProjectileCoverage{Tracks: len(vols), Published: len(publies)}}
	a.opt.Projectiles = vols
	a.opt.ProjectileAttaches = ProjectileAttachInput{Reads: lectures, Stats: types.ProjectileAttachStats{Scanned: true,
		Records: len(lectures) + 3, Read: len(lectures), Created: 1, WideLocations: 2}}
	a.doc.Projectiles = publies
	a.doc.FrameCount = 40
	a.doc.Tracks = []Track{{Slot: 561, StartFrame: 0, EndFrame: 40, XUID: "1"}}
	a.doc.Vehicles = []VehicleTrack{{Slot: 700, Gen: 2, T0: 0, T1: 25, T1Max: 30}}
	return a
}

// TestUnVolQuiSeColleSurUnJoueurPublieSonCollage : le projectile publie dont la cle est celle de la lecture porte `attach` : la
// frame de la lecture, le slot du parent, le noeud et la pose BRUTE ; l autre vol n en porte pas. La couverture compte la lecture
// posee sur un joueur.
func TestUnVolQuiSeColleSurUnJoueurPublieSonCollage(t *testing.T) {
	vols := []types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10), volDeProjectile(1140, 1, 2_050_000, 10)}
	a := assemblageDeCollage(vols, []types.ProjectileAttachRead{collageDeTest(2_230_000)})
	a.poserLesProjectilesColles()

	p := a.doc.Projectiles[a.projParPiste[0]]
	want := ProjectileAttach{FromFrame: 12, ParentSlot: 561, Node: 22, Pose: [3]uint32{32214, 33936, 33041}}
	if p.Attach == nil || *p.Attach != want {
		t.Fatalf("attache %+v, attendu %+v", p.Attach, want)
	}
	if p.Attach.FromFrame > p.DetoT {
		t.Errorf("le collage (%d) suit la detonation (%d)", p.Attach.FromFrame, p.DetoT)
	}
	if autre := a.doc.Projectiles[a.projParPiste[1]]; autre.Attach != nil {
		t.Errorf("un vol qui ne colle pas porte un collage : %+v", *autre.Attach)
	}
	if c := a.projCov.Attach; c == nil || c.OnFlight != 1 || c.Player != 1 || c.Read != 1 || c.Listed != 0 || c.OutOfMatch != 0 {
		t.Errorf("couverture %+v", c)
	}
}

// TestLeCollageNEstPublieQueDansLaFenetreDeVieDuVol : une lecture avant le vol, apres sa detonation, ou d une autre vie de la meme
// cle ne se pose sur aucun projectile et se compte « sans vol ». MUTATIONS : la borne de l echantillon ou celle de la
// detonation retiree, la generation ignoree -> ROUGE.
func TestLeCollageNEstPublieQueDansLaFenetreDeVieDuVol(t *testing.T) {
	cas := map[string]types.ProjectileAttachRead{
		"avant le vol":  collageDeTest(1_900_000),
		"apres la fin":  collageDeTest(3_900_000),
		"une autre vie": func() types.ProjectileAttachRead { l := collageDeTest(2_230_000); l.Life.Gen = 2; return l }(),
		"un autre slot": func() types.ProjectileAttachRead { l := collageDeTest(2_230_000); l.Life.Slot = 1134; return l }(),
		"a la creation (0)": func() types.ProjectileAttachRead {
			l := collageDeTest(2_230_000)
			l.Life.Slot = 1500
			l.Created = true
			return l
		}(),
	}
	for nom, l := range cas {
		a := assemblageDeCollage([]types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}, []types.ProjectileAttachRead{l})
		a.poserLesProjectilesColles()
		if got := a.doc.Projectiles[a.projParPiste[0]].Attach; got != nil {
			t.Errorf("%s : collage publie %+v", nom, *got)
		}
		if c := a.projCov.Attach; c.Listed != 1 || c.OnFlight != 0 {
			t.Errorf("%s : couverture %+v, attendu une lecture sans vol", nom, *c)
		}
	}
}

// TestUneCleReutiliseDonneSonCollageAChaqueVol : la generation d un projectile tient sur deux bits, donc une cle revient. Deux
// vols de la meme cle prennent chacun la lecture de leur vie, et une lecture ne se pose qu une fois ; la premiere lecture d une
// vie est celle qui se pose.
func TestUneCleReutiliseDonneSonCollageAChaqueVol(t *testing.T) {
	vols := []types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10), volDeProjectile(1133, 1, 3_000_000, 10, 30)}
	deuxieme := collageDeTest(3_230_000)
	deuxieme.Node = 26
	autre := collageDeTest(2_330_000) // seconde lecture de la premiere vie : seule la premiere se pose
	autre.Node = 30
	a := assemblageDeCollage(vols, []types.ProjectileAttachRead{collageDeTest(2_230_000), autre, deuxieme},
		types.ProjectileDetonation{TimestampUS: 4_200_000, HasPosition: true, X: 46.9, Y: -15, Z: 4.8})
	a.doc.FrameCount = 60
	a.poserLesProjectilesColles()

	premier, second := a.doc.Projectiles[a.projParPiste[0]], a.doc.Projectiles[a.projParPiste[1]]
	if premier.Attach == nil || premier.Attach.Node != 22 || premier.Attach.FromFrame != 12 {
		t.Errorf("premier vol : %+v, attendu le noeud 22 a la frame 12", premier.Attach)
	}
	if second.Attach == nil || second.Attach.Node != 26 || second.Attach.FromFrame != 22 {
		t.Errorf("second vol : %+v, attendu le noeud 26 a la frame 22", second.Attach)
	}
	if c := a.projCov.Attach; c.OnFlight != 2 || c.Listed != 1 {
		t.Errorf("couverture %+v, attendu deux posees et une sans vol", *c)
	}
}

// TestLaCibleDuCollageEstUnJoueurUnVehiculeOuRien : le parent se resout par les pistes deja publiees, a la frame du collage :
// un joueur dont la piste tient le slot, un vehicule de meme generation encore la, sinon rien ; chaque lecture posee compte
// dans une seule cible.
func TestLaCibleDuCollageEstUnJoueurUnVehiculeOuRien(t *testing.T) {
	cas := []struct {
		nom    string
		parent types.LifeKey
		ts     uint64
		cible  cibleDuCollage
	}{
		{"joueur", types.LifeKey{Slot: 561, Gen: 1}, 2_230_000, cibleJoueur},
		{"joueur apres la fin de sa piste", types.LifeKey{Slot: 561, Gen: 1}, 2_230_000 + 50*pasDuCollage, cibleNonResolue},
		{"vehicule", types.LifeKey{Slot: 700, Gen: 2}, 2_230_000, cibleVehicule},
		{"vehicule d une autre generation", types.LifeKey{Slot: 700, Gen: 1}, 2_230_000, cibleNonResolue},
		{"vehicule apres son dernier instant", types.LifeKey{Slot: 700, Gen: 2}, 2_230_000 + 25*pasDuCollage, cibleNonResolue},
		{"objet", types.LifeKey{Slot: 900, Gen: 1}, 2_230_000, cibleNonResolue},
		{"slot sous la base des unites", types.LifeKey{Slot: 3, Gen: 1}, 2_230_000, cibleNonResolue},
	}
	a := assemblageDeCollage(nil, nil)
	for _, c := range cas {
		frame := int((c.ts - origineDuCollage) / pasDuCollage)
		if got := cibleDuParent(c.parent, frame, a.doc.Tracks, a.doc.Vehicles); got != c.cible {
			t.Errorf("%s : cible %d, attendue %d", c.nom, got, c.cible)
		}
	}
	vols := []types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}
	for i, parent := range []types.LifeKey{{Slot: 561, Gen: 1}, {Slot: 700, Gen: 2}, {Slot: 900, Gen: 1}} {
		l := collageDeTest(2_230_000)
		l.Parent = parent
		a := assemblageDeCollage(vols, []types.ProjectileAttachRead{l})
		a.poserLesProjectilesColles()
		c := a.projCov.Attach
		if c.Player+c.Vehicle+c.Unresolved != c.OnFlight || c.OnFlight != 1 || [3]int{c.Player, c.Vehicle, c.Unresolved}[i] != 1 {
			t.Errorf("parent %v : couverture %+v", parent, *c)
		}
	}
}

// TestLaCouvertureDesCollagesClasseChaqueLecture : une lecture est hors du match, sans vol publie ou posee, jamais deux a la
// fois ; les comptes du canal passent tels quels.
func TestLaCouvertureDesCollagesClasseChaqueLecture(t *testing.T) {
	avant := collageDeTest(500_000)   // avant l origine de l axe
	apres := collageDeTest(9_000_000) // apres la derniere frame
	creation := collageDeTest(2_230_000)
	creation.Life.Slot, creation.Created = 1600, true
	lectures := []types.ProjectileAttachRead{avant, creation, collageDeTest(2_230_000), apres}
	a := assemblageDeCollage([]types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}, lectures)
	a.poserLesProjectilesColles()
	c := *a.projCov.Attach
	if c.OutOfMatch != 2 || c.Listed != 1 || c.OnFlight != 1 || c.OutOfMatch+c.Listed+c.OnFlight != c.Read {
		t.Errorf("couverture %+v", c)
	}
	if c.Records != 7 || c.Created != 1 || c.WideLocations != 2 {
		t.Errorf("les comptes du canal ne passent pas tels quels : %+v", c)
	}
}

// TestSansLectureDesCollagesLeCalqueRestePasPose : marche non jouee, archetype sans i10 ou aucun collage lu : ni collage ni couverture.
func TestSansLectureDesCollagesLeCalqueRestePasPose(t *testing.T) {
	vols := []types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}
	lectures := []types.ProjectileAttachRead{collageDeTest(2_230_000)}
	cas := map[string]func(a *assemblage){
		"marche non jouee": func(a *assemblage) { a.opt.ProjectileAttaches.Stats.Scanned = false },
		"i10 absent":       func(a *assemblage) { a.opt.ProjectileAttaches.Stats.Absent = true },
	}
	for nom, f := range cas {
		a := assemblageDeCollage(vols, lectures)
		f(a)
		a.poserLesProjectilesColles()
		if a.projCov != nil && a.projCov.Attach != nil {
			t.Errorf("%s : couverture posee %+v", nom, *a.projCov.Attach)
		}
		if got := a.doc.Projectiles[a.projParPiste[0]].Attach; got != nil {
			t.Errorf("%s : collage publie %+v", nom, *got)
		}
	}
}

// TestLeCollageSePublieEnJSONAvecSaPoseBrute : la forme publiee est `attach {fromFrame, parentSlot, node, pose}` ; la pose est
// faite de trois entiers bruts ; sans collage, le champ est absent.
func TestLeCollageSePublieEnJSONAvecSaPoseBrute(t *testing.T) {
	avec, err := json.Marshal(Projectile{Attach: &ProjectileAttach{FromFrame: 12, ParentSlot: 561, Node: 22, Pose: [3]uint32{32214, 33936, 65535}}})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(avec, &m); err != nil {
		t.Fatal(err)
	}
	if got, want := string(m["attach"]), `{"fromFrame":12,"parentSlot":561,"node":22,"pose":[32214,33936,65535]}`; got != want {
		t.Errorf("attach = %s, attendu %s", got, want)
	}
	sans, _ := json.Marshal(Projectile{})
	var vide map[string]json.RawMessage
	if err := json.Unmarshal(sans, &vide); err != nil {
		t.Fatal(err)
	}
	if _, ok := vide["attach"]; ok {
		t.Errorf("un projectile sans collage publie `attach` : %s", sans)
	}
}

// TestSansPisteDeProjectileLesCollagesSontPublieEtComptes : la lecture des trajectoires est vide ou en echec alors que des collages sont
// lus : chacun est publie dans `attachments[]`, compte dans la couverture, et `noTracks` dit qu aucun vol n a pu le porter. Les
// repetitions du canal sont publiees dans la couverture. MUTATIONS : la sortie anticipee sans piste rendue, `noTracks` non pose -> ROUGE.
func TestSansPisteDeProjectileLesCollagesSontPublieEtComptes(t *testing.T) {
	a := assemblageDeCollage(nil, []types.ProjectileAttachRead{collageDeTest(2_230_000), collageDeTest(2_500_000)})
	a.projCov = nil
	a.opt.ProjectileAttaches.Stats.Repeats = 3
	a.poserLesProjectilesColles()
	if a.projCov == nil || a.projCov.Attach == nil {
		t.Fatal("collages lus et aucune couverture : silence")
	}
	c := a.projCov.Attach
	if c.Read != 2 || c.Listed != 2 || c.NoTracks != 2 || c.OnFlight != 0 || c.Repeats != 3 || len(a.doc.Attachments) != 2 {
		t.Errorf("couverture %+v, %d collages publies ; attendu 2 lus, 2 listes, 2 sans piste, 3 repetitions", *c, len(a.doc.Attachments))
	}
}

// TestSansPisteLaCouvertureDesCollagesAtteintLeDocumentPublie : PAR LE VRAI ASSEMBLAGE, sur le JSON du document. Des collages lus et
// aucune piste de projectile : `attachments[]` sort ET `coverage.projectiles.attach` sort avec `noTracks` et `repeats` justes (le bloc de
// couverture cree apres la recopie de `composerLaCouverture` se perdait). MUTATION : la pose sur le document retiree -> ROUGE.
func TestSansPisteLaCouvertureDesCollagesAtteintLeDocumentPublie(t *testing.T) {
	c := collageDeTest(10_000_000)
	doc := BuildFromPositions(context.Background(), "m", "halo_infinite", positionsPourVersion(), nil, Options{
		FrameIntervalMS: 100,
		ProjectileAttaches: ProjectileAttachInput{Reads: []types.ProjectileAttachRead{c, c},
			Stats: types.ProjectileAttachStats{Scanned: true, Records: 5, Read: 2, Repeats: 3}},
	})
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var pub struct {
		Attachments []json.RawMessage `json:"attachments"`
		Coverage    struct {
			Projectiles *struct {
				Tracks int `json:"tracks"`
				Attach *struct {
					Read     int `json:"read"`
					Listed   int `json:"listed"`
					NoTracks int `json:"noTracks"`
					Repeats  int `json:"repeats"`
				} `json:"attach"`
			} `json:"projectiles"`
		} `json:"coverage"`
	}
	if err := json.Unmarshal(b, &pub); err != nil {
		t.Fatal(err)
	}
	p := pub.Coverage.Projectiles
	if p == nil || p.Attach == nil {
		t.Fatalf("collages lus sans piste : `coverage.projectiles.attach` absent du document publie (%d attachments)", len(pub.Attachments))
	}
	if len(pub.Attachments) != 2 || p.Tracks != 0 || p.Attach.Read != 2 || p.Attach.Listed != 2 || p.Attach.NoTracks != 2 || p.Attach.Repeats != 3 {
		t.Errorf("document publie : %d attachments, couverture %+v", len(pub.Attachments), *p.Attach)
	}
}
