package replay

// golden_degats_test.go — LES MESSAGES `damage_aftermath` DES HUIT BUILDS, FIGES A COTE DE LEURS ENTREES.
//
// Les messages de degat du film voyagent dans le COMPLEMENT de la section 1 du fichier de faits,
// pas dans le blob des entrees (même forme que golden_medailles_test.go) : un fichier par build,
// `degats_<short8>.bin.gz`, encodé par le codec de production ([encodeDegats]), posé sur les entrées relues
// ([chargerGoldenBuild], [loadGoldenInputs]). Ils ne portent aucun xuid (des index de joueur du film) : la
// table des identités fictives n'a rien à y réécrire ; les xuids du golden d'assemblage sont ceux du roster.
//
// REGENERATION (jamais d'édition à la main), depuis le film, UN FILM PAR PROCESSUS :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./replay/ -run 'TestGoldenDegatsRegenerate/<short8>' -update
//
// Elle échoue TOUJOURS en nommant ce qu'elle a écrit ; on relance sans les drapeaux pour vérifier.

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// degatsPath : le fichier des messages de degat d'un build.
func degatsPath(short8 string) string {
	return filepath.Join(goldenDir, "degats_"+short8+".bin.gz")
}

// chargerDegatsDuBuild pose sur `g` les messages de degat figés d'un build. ABSENT = ECHEC : les huit builds les
// portent, et un build qui les perdrait publierait un calque absent.
func chargerDegatsDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(degatsPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("messages de degat du build %s absents (%v) — régénérer avec %s=<cache> -run "+
			"TestGoldenDegatsRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("messages de degat %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("messages de degat %s : décompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	decodeDegats(r, g)
	if r.err != nil || r.off != len(r.b) || !g.DamagesRead {
		t.Fatalf("messages de degat %s : relecture invalide (err %v, %d/%d octets lus, lus %v)", short8, r.err,
			r.off, len(r.b), g.DamagesRead)
	}
}

// ecrireDegatsDuBuild écrit le fichier des messages de degat d'un build, par le codec de production, et rend sa taille.
func ecrireDegatsDuBuild(t *testing.T, short8 string, g *FilmFacts) int {
	t.Helper()
	w := &gwriter{}
	encodeDegats(w, g)
	var out bytes.Buffer
	zw, err := gzip.NewWriterLevel(&out, gzip.BestCompression)
	if err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if _, err := zw.Write(w.b); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip : %v", err)
	}
	if err := os.WriteFile(degatsPath(short8), out.Bytes(), 0o600); err != nil {
		t.Fatalf("écriture %s : %v", degatsPath(short8), err)
	}
	return out.Len()
}

// TestGoldenDegatsRegenerate : LA SEULE PORTE D'ECRITURE des messages de degat figés (cf. l'en-tête).
func TestGoldenDegatsRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("régénération des messages de degat des builds : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("régénération des messages de degat des builds : " + miniFilmCacheEnv + " non défini")
	}
	for _, b := range goldenBuilds() {
		t.Run(b.Short8, func(t *testing.T) {
			entry, err := b.mapQuant(t)
			if err != nil {
				t.Fatalf("carte %q : %v", b.Map, err)
			}
			g, err := decodeFilmInputsForEntry(b.Short8, filepath.Join(cache, b.Short8), entry)
			if err != nil {
				t.Fatalf("décodage de %s : %v", b.Short8, err)
			}
			n := ecrireDegatsDuBuild(t, b.Short8, g)
			t.Fatalf("REECRIT %s (%d messages, %d octets) — relancer sans -update pour vérifier",
				degatsPath(b.Short8), len(g.Damages), n)
		})
	}
}

// renderTouches publie dans le golden d'assemblage les touches et les dégâts en points (estimation) et le bilan de tirs par
// joueur : la couverture, le bilan de chaque joueur, la première et la dernière touche, la première et la dernière ligne de dégât.
func renderTouches(p func(string, ...any), doc ReplayDocument) {
	p("## TOUCHES ET DEGATS EN POINTS (ESTIMATION) — effets de dégât d'arme, une touche par (auteur, unité, trame) ; absents si les messages de dégât n'ont pas été lus")
	c := doc.Coverage.Hits
	if c == nil {
		p("couverture absente : messages de dégât non lus")
		return
	}
	p("%d message(s) · %d d'arme · %d touche(s) · %d posée(s) · %d sans auteur · %d sans victime · %d avant la frame 0 · %d après la fin · %d joueur(s) dont %d suffisant(s) (couverture >= %.1f, doublons < %.1f)",
		c.Messages, c.WeaponMessages, c.Hits, c.Published, c.NoShooter, c.NoVictim, c.BeforeOrigin, c.AfterEnd, c.Sufficient, c.Players, c.MinCoverage, c.MaxDuplicates)
	p("%d ligne(s) de dégât dont %d mesurée(s) · %d message(s) sans piste · %d touche(s) et %d dégât(s) sur soi · %d trame(s) arrêtée(s) et %d de vitalités non lues sur %d", c.Damages, c.DamagesMeasured, c.DamagesOffUnit, c.SelfHits, c.SelfDamages, c.StoppedFrames, c.VitalsUnread, c.Frames)
	for _, s := range doc.ShotStats {
		p("  %s : %d tir(s) dont %d écrit(s) · %d message(s) · %d touche(s) · couverture %.2f · doublons %.2f · infligé %.1f · subi %.1f · suffisant %v",
			s.XUID, s.Shots, s.ShotEvents, s.Messages, s.Hits, s.Coverage, s.Duplicates, s.DamageDealt, s.DamageTaken, s.Sufficient)
	}
	if n := len(doc.Hits); n > 0 {
		f, l := doc.Hits[0], doc.Hits[n-1]
		p("première : t=%d %s -> %s %s w=%s · dernière : t=%d %s -> %s %s w=%s", f.T, f.Shooter, f.Victim, f.Effect, f.Weapon, l.T, l.Shooter, l.Victim, l.Effect, l.Weapon)
	}
	// LES LIGNES A ZERO POINT NE SE LISENT PAS ICI : le rendu garde le sens d avant leur publication (premiere et derniere ligne A POINTS).
	var avecPoints []Damage
	for _, d := range doc.Damage {
		if d.Shield != 0 || d.Body != 0 {
			avecPoints = append(avecPoints, d)
		}
	}
	if n := len(avecPoints); n > 0 {
		f, l := avecPoints[0], avecPoints[n-1]
		p("dégât, première : t=%d %s -> %s %s bouclier %.1f corps %.1f · dernière : t=%d %s -> %s %s bouclier %.1f corps %.1f",
			f.T, f.Attacker, f.Victim, f.Effect, f.Shield, f.Body, l.T, l.Attacker, l.Victim, l.Effect, l.Shield, l.Body)
	}
	p("")
}
