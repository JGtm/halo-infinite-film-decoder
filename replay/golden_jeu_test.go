package replay

// golden_jeu_test.go — LES MESSAGES DE JEU DES HUIT BUILDS, FIGES A COTE DE LEURS ENTREES.
//
// Les messages de jeu et les scripts du film voyagent dans le COMPLEMENT de la section 1 du fichier de faits,
// pas dans le blob des entrees (même forme que golden_medailles_test.go) : un fichier par build,
// `jeu_<short8>.bin.gz`, encodé par le codec de production ([encodeJeu]), posé sur les entrées relues
// ([chargerGoldenBuild], [loadGoldenInputs]). Ils ne portent aucun xuid (des index de joueur du film) : la
// table des identités fictives n'a rien à y réécrire ; les xuids du golden d'assemblage sont ceux du roster.
//
// REGENERATION (jamais d'édition à la main), depuis le film, UN FILM PAR PROCESSUS :
//
//	REPLAY_FILM_CACHE=<repo>/data/cache/film_chunks \
//	  go test ./replay/ -run 'TestGoldenJeuRegenerate/<short8>' -update
//
// Elle échoue TOUJOURS en nommant ce qu'elle a écrit ; on relance sans les drapeaux pour vérifier.

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// jeuPath : le fichier des messages de jeu d'un build.
func jeuPath(short8 string) string {
	return filepath.Join(goldenDir, "jeu_"+short8+".bin.gz")
}

// chargerJeuDuBuild pose sur `g` les messages de jeu figés d'un build. ABSENT = ECHEC : les huit builds les
// portent, et un build qui les perdrait publierait un calque absent.
func chargerJeuDuBuild(t *testing.T, short8 string, g *FilmFacts) {
	t.Helper()
	raw, err := os.ReadFile(jeuPath(short8)) //nolint:gosec // chemin construit depuis la table des builds
	if err != nil {
		t.Fatalf("messages de jeu du build %s absents (%v) — régénérer avec %s=<cache> -run "+
			"TestGoldenJeuRegenerate -update", short8, err, miniFilmCacheEnv)
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("messages de jeu %s : gzip illisible : %v", short8, err)
	}
	defer func() { _ = zr.Close() }()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(zr); err != nil {
		t.Fatalf("messages de jeu %s : décompression : %v", short8, err)
	}
	r := &greader{b: buf.Bytes()}
	decodeJeu(r, g)
	if r.err != nil || r.off != len(r.b) || !g.GameEventsRead {
		t.Fatalf("messages de jeu %s : relecture invalide (err %v, %d/%d octets lus, lus %v)", short8, r.err,
			r.off, len(r.b), g.GameEventsRead)
	}
}

// ecrireJeuDuBuild écrit le fichier des messages de jeu d'un build, par le codec de production, et rend sa taille.
func ecrireJeuDuBuild(t *testing.T, short8 string, g *FilmFacts) int {
	t.Helper()
	w := &gwriter{}
	encodeJeu(w, g)
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
	if err := os.WriteFile(jeuPath(short8), out.Bytes(), 0o600); err != nil {
		t.Fatalf("écriture %s : %v", jeuPath(short8), err)
	}
	return out.Len()
}

// TestGoldenJeuRegenerate : LA SEULE PORTE D'ECRITURE des messages de jeu figés (cf. l'en-tête).
func TestGoldenJeuRegenerate(t *testing.T) {
	cache := os.Getenv(miniFilmCacheEnv)
	switch {
	case !*updateGolden:
		t.Skip("régénération des messages de jeu des builds : passer -update (et " + miniFilmCacheEnv + ")")
	case cache == "":
		t.Skip("régénération des messages de jeu des builds : " + miniFilmCacheEnv + " non défini")
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
			n := ecrireJeuDuBuild(t, b.Short8, g)
			t.Fatalf("REECRIT %s (%d messages, %d octets) — relancer sans -update pour vérifier",
				jeuPath(b.Short8), len(g.GameEvents), n)
		})
	}
}

// renderMedaillesEtScores publie, dans le golden d'assemblage, les deux calques datés du film : les médailles
// puis le score par action.
func renderMedaillesEtScores(p func(string, ...any), doc ReplayDocument) {
	renderMedailles(p, doc)
	renderScoreActions(p, doc)
}

// renderScoreActions publie le calque du score par action : sa couverture, le compte par réponse, la somme
// par joueur, la première et la dernière action.
func renderScoreActions(p func(string, ...any), doc ReplayDocument) {
	p("## SCORE PAR ACTION — réponses du jeu (genre 82), posées par (TS - origine) / pas ; absentes si les messages n'ont pas été lus")
	c := doc.Coverage.ScoreActions
	if c == nil {
		p("couverture absente : messages de jeu non lus")
		return
	}
	p("groupe %q (absent : %v) · trames %d (lues %d, arrêtées %d, vue A illisible : %v) · complet : %v", c.Group, c.GroupMissing, c.Frames,
		c.ClosedFrames, c.StoppedFrames, c.ViewAUnreadable, c.Complete)
	p("%d message(s) : %d à points · %d sans rapport avec le score · %d nommé(s) sans points · %d hors de l'arbre · %d non confirmé(s) · %d copie(s) retirée(s)",
		c.Messages, c.ScoreMessages, c.Unrelated, c.NamedWithoutPoints, c.OutsideTree, c.UnconfirmedRank, c.Copies)
	p("%d (message, destinataire) à points · %d publié(s) · %d sans humain au roster · %d absent(s) à l'instant · %d sans origine · "+
		"%d avant la frame 0 · %d après la fin · %d sans cible résolue",
		c.Scored, c.Published, c.NoRecipient, c.NotPresent, c.NoOrigin, c.BeforeOrigin, c.AfterEnd, c.NoTarget)
	parNom := map[string]int{}
	parJoueur := map[string]int{}
	for _, a := range doc.ScoreActions {
		parNom[fmt.Sprintf("%s (%d)", a.Name, a.Points)]++
		parJoueur[a.XUID] += a.Points
	}
	for _, k := range slices.Sorted(maps.Keys(parNom)) {
		p("  %s x%d", k, parNom[k])
	}
	for _, k := range slices.Sorted(maps.Keys(parJoueur)) {
		p("  somme %s = %d", k, parJoueur[k])
	}
	if n := len(doc.ScoreActions); n > 0 {
		f, l := doc.ScoreActions[0], doc.ScoreActions[n-1]
		p("première : t=%d xuid=%s %s %d cible=%q · dernière : t=%d xuid=%s %s %d cible=%q", f.T, f.XUID, f.Name, f.Points,
			f.Target, l.T, l.XUID, l.Name, l.Points, l.Target)
	}
	p("")
}
