package filmcache_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/filmcache"
)

const manifesteAvecTailles = `{"chunks":[
  {"index":0,"chunk_type":2,"start_ms":0,"size_bytes":3},
  {"index":4,"chunk_type":3,"start_ms":15000,"size_bytes":2}
]}`

// dossierDeChunks pose des chunks dans un repertoire quelconque, hors disposition du cache.
func dossierDeChunks(t *testing.T, chunks map[int]string) string {
	t.Helper()
	dir := t.TempDir()
	for i, body := range chunks {
		if err := os.WriteFile(filepath.Join(dir, chunkFileName(i)), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestParseManifestRendLIndex : l'index se lit, un JSON invalide est une erreur.
func TestParseManifestRendLIndex(t *testing.T) {
	metas, err := filmcache.ParseManifest([]byte(manifesteAvecTailles))
	if err != nil || len(metas) != 2 {
		t.Fatalf("ParseManifest = %v, %v", metas, err)
	}
	if metas[1].Index != 4 || metas[1].ChunkType != 3 || metas[1].StartMS != 15000 {
		t.Errorf("entree mal relue : %+v", metas[1])
	}
	if _, err := filmcache.ParseManifest([]byte("{pas du json")); err == nil {
		t.Error("un manifeste invalide doit rendre une erreur")
	}
}

// TestNewSourceLitSansTailleEtSansPanique : une source par chemin lit ses chunks dans le
// repertoire donne, sans taille connue. Le defaut vise : des tailles nulles que Chunk indexait.
// MUTATIONS — tailles laissees a nil (panique d'indice), disposition du cache utilisee a la
// place du repertoire : ROUGES.
func TestNewSourceLitSansTailleEtSansPanique(t *testing.T) {
	dir := dossierDeChunks(t, map[int]string{0: "AAA", 4: "BB"})
	metas, err := filmcache.ParseManifest([]byte(manifesteAvecTailles))
	if err != nil {
		t.Fatal(err)
	}
	src := filmcache.NewSource(metas, dir)
	if src.NumChunks() != 2 {
		t.Fatalf("NumChunks = %d", src.NumChunks())
	}
	raw, err := src.Chunk(1)
	if err != nil || string(raw) != "BB" {
		t.Errorf("Chunk(1) = %q, %v ; attendu \"BB\" (numero de fichier 4)", raw, err)
	}
	if src.TailleDeclaree(0) != 0 {
		t.Errorf("TailleDeclaree = %d, attendu 0 (inconnue)", src.TailleDeclaree(0))
	}
	// Un fichier de taille quelconque passe : aucun controle sans taille.
	if err := os.WriteFile(filepath.Join(dir, chunkFileName(0)), []byte("AAAAAAAA"), 0o600); err != nil {
		t.Fatal(err)
	}
	if raw, err := src.Chunk(0); err != nil || len(raw) != 8 {
		t.Errorf("sans taille, Chunk(0) = %q, %v", raw, err)
	}
	if _, err := src.Chunk(2); err == nil {
		t.Error("Chunk hors bornes doit rendre une erreur")
	}
}

// TestOpenManifestRemplitLesTailles : avec `size_bytes`, la source par chemin garde le controle de
// troncature d'[Open], et une taille absente reste lisible.
// MUTATIONS — tailles non remplies (aucune troncature detectee), taille lue a la mauvaise
// position : ROUGES.
func TestOpenManifestRemplitLesTailles(t *testing.T) {
	dir := dossierDeChunks(t, map[int]string{0: "AAA", 4: "BB"})
	src, err := filmcache.OpenManifest([]byte(manifesteAvecTailles), dir)
	if err != nil {
		t.Fatal(err)
	}
	if src.TailleDeclaree(0) != 3 || src.TailleDeclaree(1) != 2 {
		t.Errorf("tailles = %d, %d ; attendu 3, 2", src.TailleDeclaree(0), src.TailleDeclaree(1))
	}
	if raw, err := src.Chunk(0); err != nil || string(raw) != "AAA" {
		t.Errorf("Chunk(0) = %q, %v", raw, err)
	}
	if err := os.WriteFile(filepath.Join(dir, chunkFileName(4)), []byte("B"), 0o600); err != nil {
		t.Fatal(err)
	}
	var tronque *filmcache.ErrChunkTronque
	if _, err := src.Chunk(1); !errors.As(err, &tronque) || tronque.Attendu != 2 || tronque.Lu != 1 {
		t.Errorf("chunk tronque : err = %v, attendu ErrChunkTronque{Attendu:2, Lu:1}", err)
	}

	// Manifeste historique sans taille : lisible sans controle.
	sans, err := filmcache.OpenManifest([]byte(manifesteDeuxChunks), dossierDeChunks(t, map[int]string{0: "x", 1: "yy"}))
	if err != nil {
		t.Fatal(err)
	}
	if raw, err := sans.Chunk(1); err != nil || string(raw) != "yy" {
		t.Errorf("sans taille, Chunk(1) = %q, %v", raw, err)
	}
	if _, err := filmcache.OpenManifest([]byte("{pas du json"), dir); err == nil {
		t.Error("un manifeste invalide doit rendre une erreur")
	}
}
