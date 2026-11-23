package replay

import (
	"encoding/json"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// collageNeColle est la lecture d un projectile ne colle (sans vol) de type 0x580b8831, cree par l unite du slot 600, colle au joueur du slot 561.
func collageNeColle(ts uint64) types.ProjectileAttachRead {
	l := collageDeTest(ts)
	l.Life.Slot, l.Created = 1700, true
	l.BornUS, l.HasBirth = ts, true
	l.Tag, l.HasTag = 0x580b8831, true
	l.Creator, l.HasCreator = types.LifeKey{Slot: 600, Gen: 1}, true
	return l
}

// TestUnCollageSansVolSePubliePartoutUneSeuleFois : la lecture qu aucun vol ne porte est publiee dans `attachments[]` avec son parent resolu,
// son type et son createur resolu en piste ; la lecture posee sur un vol n y est pas ; la couverture compte chacune dans une seule classe.
// MUTATIONS : la lecture posee aussi publiee, la resolution du createur retiree -> ROUGE.
func TestUnCollageSansVolSePubliePartoutUneSeuleFois(t *testing.T) {
	lectures := []types.ProjectileAttachRead{collageDeTest(2_230_000), collageNeColle(2_400_000)}
	a := assemblageDeCollage([]types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}, lectures)
	a.doc.Tracks = append(a.doc.Tracks, Track{Slot: 600, StartFrame: 0, EndFrame: 40, XUID: "2"})
	a.poserLesProjectilesColles()

	if len(a.doc.Attachments) != 1 {
		t.Fatalf("%d collage(s) publie(s) dans attachments[], attendu 1 : %+v", len(a.doc.Attachments), a.doc.Attachments)
	}
	want := Attachment{T: 14, ParentSlot: 561, Target: "player", Victim: "1", Node: 22, Pose: [3]uint32{32214, 33936, 33041},
		Tag: "580b8831", By: 600, Attacker: "2", Born: true}
	if got := a.doc.Attachments[0]; got != want {
		t.Errorf("collage publie %+v, attendu %+v", got, want)
	}
	if a.doc.Projectiles[a.projParPiste[0]].Attach == nil {
		t.Error("le collage porte par un vol a disparu de projectiles[].attach")
	}
	c := *a.projCov.Attach
	if c.Read != 2 || c.OnFlight != 1 || c.Listed != 1 || c.OutOfMatch != 0 || c.Player != 2 || c.Tagged != 1 || c.Authored != 1 {
		t.Errorf("couverture %+v", c)
	}
}

// TestUneCreationPosterieureAuVolEstUneAutreVie : la cle d un vol se reutilise ; une lecture dont la creation (lue) suit le debut du vol est celle
// d une autre vie, publiee a part, meme si elle tombe avant la detonation du vol.
func TestUneCreationPosterieureAuVolEstUneAutreVie(t *testing.T) {
	autreVie := collageDeTest(2_230_000)
	autreVie.BornUS, autreVie.HasBirth = 2_100_000, true
	a := assemblageDeCollage([]types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}, []types.ProjectileAttachRead{autreVie})
	a.poserLesProjectilesColles()
	if a.doc.Projectiles[a.projParPiste[0]].Attach != nil || len(a.doc.Attachments) != 1 {
		t.Errorf("la lecture d une autre vie ne doit pas se poser sur le vol : %+v, %+v", a.doc.Projectiles[a.projParPiste[0]].Attach, a.doc.Attachments)
	}
	vraieVie := collageDeTest(2_230_000)
	vraieVie.BornUS, vraieVie.HasBirth = 2_000_000, true
	a = assemblageDeCollage([]types.ProjectileTrack{volDeProjectile(1133, 1, 2_000_000, 10)}, []types.ProjectileAttachRead{vraieVie})
	a.poserLesProjectilesColles()
	if a.doc.Projectiles[a.projParPiste[0]].Attach == nil || len(a.doc.Attachments) != 0 {
		t.Errorf("la lecture de la vie du vol doit se poser sur lui : %+v", a.doc.Attachments)
	}
}

// TestLeCreateurDUnCollageSeResoutParLesPistes : un createur sans piste a la frame est compte a part, un collage dont la creation ne nomme pas
// de createur aussi ; un parent bot est une cible sans victime ; un vehicule est une cible sans victime.
func TestLeCreateurDUnCollageSeResoutParLesPistes(t *testing.T) {
	sansPiste := collageNeColle(2_400_000)
	sansPiste.Creator.Slot = 900
	sansCreateur := collageNeColle(2_500_000)
	sansCreateur.HasCreator, sansCreateur.Creator = false, types.LifeKey{}
	sansCreateur.Life.Slot = 1701
	surBot := collageNeColle(2_600_000)
	surBot.Life.Slot = 1702
	surBot.Parent.Slot = 600
	surVehicule := collageNeColle(2_700_000)
	surVehicule.Life.Slot = 1703
	surVehicule.Parent = types.LifeKey{Slot: 700, Gen: 2}
	a := assemblageDeCollage(nil, []types.ProjectileAttachRead{sansPiste, sansCreateur, surBot, surVehicule})
	a.projCov = &ProjectileCoverage{}
	a.doc.Tracks = append(a.doc.Tracks, Track{Slot: 600, StartFrame: 0, EndFrame: 40}) // un bot : une piste sans xuid
	a.poserLesProjectilesColles()

	at := a.doc.Attachments
	if len(at) != 4 {
		t.Fatalf("%d collages publies, attendu 4", len(at))
	}
	if at[0].By != 0 || at[0].Attacker != "" || at[1].By != 0 {
		t.Errorf("un createur sans piste ou absent ne doit pas se resoudre : %+v %+v", at[0], at[1])
	}
	if at[2].Target != "player" || at[2].Victim != "" || at[2].By != 600 || at[2].Attacker != "" {
		t.Errorf("un bot est un joueur sans xuid, createur et cible : %+v", at[2])
	}
	if at[3].Target != "vehicle" || at[3].Victim != "" {
		t.Errorf("un vehicule est une cible sans victime : %+v", at[3])
	}
	if c := *a.projCov.Attach; c.AuthorNoTrack != 1 || c.AuthorNone != 1 || c.Authored != 2 || c.Listed != 4 ||
		c.Player != 3 || c.Vehicle != 1 {
		t.Errorf("couverture %+v", c)
	}
}

// TestUnCollageSansVolSePublieEnJSON : la forme publiee de `attachments[]` ; les champs sans valeur sont absents.
func TestUnCollageSansVolSePublieEnJSON(t *testing.T) {
	plein, err := json.Marshal(Attachment{T: 14, ParentSlot: 561, Target: "player", Victim: "1", Node: 22, Pose: [3]uint32{1, 2, 65535},
		Tag: "580b8831", By: 600, Attacker: "2", Born: true})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"t":14,"parentSlot":561,"target":"player","victim":"1","node":22,"pose":[1,2,65535],"tag":"580b8831","by":600,"attacker":"2","born":true}`
	if string(plein) != want {
		t.Errorf("attachment = %s, attendu %s", plein, want)
	}
	nu, _ := json.Marshal(Attachment{T: 3, ParentSlot: 900, Node: 5})
	if string(nu) != `{"t":3,"parentSlot":900,"node":5,"pose":[0,0,0]}` {
		t.Errorf("un collage sans cible, type ni createur : %s", nu)
	}
}
