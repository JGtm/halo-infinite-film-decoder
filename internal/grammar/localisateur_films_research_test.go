//go:build research

package grammar

// localisateur_films_research_test.go — LE PRE-TEST DE TETE DU LOCALISATEUR, CONFRONTE A DE VRAIS
// FILMS : sur CHAQUE trame delta, la localisation de production rend la position de la forme sans
// pre-test ([marchLocateStrictReference], [marchLocateFallbackReference]), aucune position acceptee
// par la reference n est ecartee, et les compteurs de l observateur sont ceux de la reference.
//
// Les films ne sont pas dans le depot. L operateur donne leur correspondance (la meme que celle de
// `cmd/film-bench` : nom, dossier des morceaux, manifeste, carte), le catalogue des bornes de
// carte, et les noms retenus :
//
//	LOCALISATEUR_CORRESPONDANCE=<correspondance.tsv> LOCALISATEUR_CATALOGUE=<map_quant_bounds.json> \
//	  LOCALISATEUR_FILMS=film06,film09 \
//	  go test -tags=research -count=1 -timeout 60m -run '^TestLePretestDeTeteSurDeVraisFilms$' ./internal/grammar/
//
// Le monde et le cadre sont ceux de la marche des trames sous la generation stricte, profil par
// defaut de la carte : le test juge la decision du localisateur sur de vrais payloads et de vrais
// mondes, pas le profil calibre de la cuisson.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ligneDeCorrespondance : une ligne `nom dossier manifeste carte` ; un chemin relatif se lit depuis
// le dossier du fichier.
type ligneDeCorrespondance struct{ dossier, manifeste, carte string }

func lireLaCorrespondance(t *testing.T, chemin string) map[string]ligneDeCorrespondance {
	t.Helper()
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin donne par l operateur
	if err != nil {
		t.Fatalf("correspondance illisible : %v", err)
	}
	base := filepath.Dir(chemin)
	resoudre := func(p string) string {
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		return filepath.Join(base, p)
	}
	out := map[string]ligneDeCorrespondance{}
	for _, l := range strings.Split(string(brut), "\n") {
		l = strings.TrimRight(l, "\r")
		cols := strings.Split(l, "\t")
		if strings.TrimSpace(l) == "" || strings.HasPrefix(l, "#") || len(cols) != 4 || cols[0] == "nom" {
			continue
		}
		out[cols[0]] = ligneDeCorrespondance{resoudre(cols[1]), resoudre(cols[2]), strings.TrimSpace(cols[3])}
	}
	return out
}

// ouvrirUnFilm charge un film et le contexte de sa carte, sous la generation stricte.
func ouvrirUnFilm(t *testing.T, l ligneDeCorrespondance, cat *profile.MapQuantCatalog) *FilmContext {
	t.Helper()
	brut, err := os.ReadFile(l.manifeste) //nolint:gosec // chemin donne par l operateur
	if err != nil {
		t.Fatalf("manifeste illisible : %v", err)
	}
	var mf struct {
		Chunks []struct {
			Index     int `json:"index"`
			ChunkType int `json:"chunk_type"`
			StartMS   int `json:"start_ms"`
		} `json:"chunks"`
	}
	if err := json.Unmarshal(brut, &mf); err != nil {
		t.Fatalf("manifeste invalide : %v", err)
	}
	meta := make([]types.ChunkMeta, len(mf.Chunks))
	for i, c := range mf.Chunks {
		meta[i] = types.ChunkMeta{Index: c.Index, ChunkType: c.ChunkType, StartMS: c.StartMS}
	}
	film, err := source.LoadDir(l.dossier, meta)
	if err != nil {
		t.Fatalf("film illisible : %v", err)
	}
	entree, err := cat.Lookup(l.carte)
	if err != nil {
		found := false
		for _, e := range cat.Maps {
			if e.Module == l.carte {
				entree, found = e, true
				break
			}
		}
		if !found {
			t.Fatalf("carte absente du catalogue")
		}
	}
	fc := NewFilmContextForMap(film, &entree, nil)
	bal := fc.ProfilDeBalayage()
	bal.Grammaire.GenerationStricte = true
	fc.PoserProfilDeBalayage(bal)
	return fc
}

func TestLePretestDeTeteSurDeVraisFilms(t *testing.T) {
	corr, catalogue, noms := os.Getenv("LOCALISATEUR_CORRESPONDANCE"), os.Getenv("LOCALISATEUR_CATALOGUE"),
		os.Getenv("LOCALISATEUR_FILMS")
	if corr == "" || catalogue == "" || noms == "" {
		t.Skip("LOCALISATEUR_CORRESPONDANCE, LOCALISATEUR_CATALOGUE et LOCALISATEUR_FILMS sont requis")
	}
	cat, err := profile.LoadMapQuantCatalog(catalogue)
	if err != nil {
		t.Fatalf("catalogue illisible : %v", err)
	}
	lignes := lireLaCorrespondance(t, corr)
	for _, nom := range strings.Split(noms, ",") {
		l, ok := lignes[strings.TrimSpace(nom)]
		if !ok {
			t.Fatalf("film %q absent de la correspondance", nom)
		}
		t.Run(nom, func(t *testing.T) { comparerUnFilm(t, ouvrirUnFilm(t, l, cat)) })
	}
}

func comparerUnFilm(t *testing.T, fc *FilmContext) {
	t.Helper()
	obsNouvelle, obsReference := NouvelleObservation(), NouvelleObservation()
	m, err := fc.nouveauMarcheurDesTrames(nil)
	if err != nil {
		t.Fatalf("marche des trames : %v", err)
	}
	var total bilanDeComparaison
	var ordresLibres, ordresStricts int
	m.parcourir(func(tr *trameLue) bool {
		pay := tr.paquet.Payload
		cfgNouvelle := m.cfg
		cfgNouvelle.Obs = obsNouvelle
		cfgReference := m.cfg
		cfgReference.Obs = obsReference

		// Ordres de production : la nouvelle forme contre la composition des references.
		sRef := marchLocateStrictReference(pay, m.monde, cfgReference)
		for _, ordre := range []OrdreDeLocalisation{SignatureStricte, SignaturePuisLargeurLibre} {
			attendu, attenduLibre := sRef, false
			if ordre == SignaturePuisLargeurLibre {
				ok := false
				if sRef >= 0 {
					rec, _, essai := TryDeltaAt(pay, sRef, m.monde, cfgReference)
					ok = essai && m.monde.GenerationMatches(rec.ID, cfgReference.Profil.Grammaire.GenerationStricte)
				}
				if !ok {
					attendu = marchLocateFallbackReference(pay, m.monde, cfgReference)
					attenduLibre = attendu >= 0
				}
			}
			s, libre := LocaliserBoucleDeRecords(pay, m.monde, cfgNouvelle, ordre)
			if s != attendu || libre != attenduLibre {
				t.Fatalf("ordre %d : (%d, %v), reference (%d, %v)", ordre, s, libre, attendu, attenduLibre)
			}
			if ordre == SignaturePuisLargeurLibre && attendu >= 0 {
				ordresLibres++
			} else if attendu >= 0 {
				ordresStricts++
			}
		}
		// Chaque position acceptee par une reference passe le pre-test.
		cfgSansObs := m.cfg
		cfgSansObs.Obs = nil
		total.ajouter(comparerAuxReferences(t, pay, m.monde, cfgSansObs))
		if !memesCompteurs(obsNouvelle, obsReference) {
			t.Fatalf("les compteurs de l observateur different apres la trame %d", total.payloads)
		}
		return true
	})
	t.Logf("%d trames comparees ; %d positions acceptees par les references ; %d localisations strictes "+
		"et %d localisations libres rendues par les references ; %d decisions d ordre non negatives "+
		"(%d strictes, %d du second ordre) ; %d leurres stricts, %d leurres libres ecartes ou rejetes",
		total.payloads, total.acceptees, total.strictes, total.libres, ordresStricts+ordresLibres,
		ordresStricts, ordresLibres, total.leursStricts, total.leursLibres)
	if total.payloads == 0 || total.acceptees == 0 {
		t.Fatalf("aucune acceptation exercee : %+v", total)
	}
}

// memesCompteurs dit que deux observations portent les memes compteurs, a une exception pres : les
// deux histogrammes d index de position ([Observation.IndexAbsolus], [Observation.IndexSansLigne]),
// que chaque essai de decodage alimente. Ils n ont aucun lecteur hors des instruments de recherche et
// ne sont publies nulle part ; un essai ecarte ne les alimente pas.
func memesCompteurs(a, b *Observation) bool {
	ca, cb := *a, *b
	ca.IndexAbsolus, cb.IndexAbsolus = nil, nil
	ca.IndexSansLigne, cb.IndexSansLigne = nil, nil
	return reflect.DeepEqual(ca, cb)
}
