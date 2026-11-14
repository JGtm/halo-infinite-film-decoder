package grammar

import (
	"slices"
	"testing"
)

// chunkAvecNoms écrit un chunk 0 synthétique : les `noms`, chacun précédé de deux octets de bourrage, puis
// décalés de [decalageDesNomsDeScript] bits vers la droite, comme le jeu les écrit dans le flux.
func chunkAvecNoms(noms ...string) []byte {
	var flux []byte
	for _, n := range noms {
		flux = append(flux, 0x00, 0x01)
		flux = append(flux, n...)
	}
	flux = append(flux, 0x00, 0x00)
	chunk := make([]byte, len(flux)+1)
	for i, b := range flux {
		chunk[i] |= b >> decalageDesNomsDeScript
		chunk[i+1] |= b << (8 - decalageDesNomsDeScript)
	}
	return chunk
}

// TestLesNomsDeScriptSeLisentDansLeChunk0 : le nom de fichier de chaque chemin qui finit par `.lua`, en
// minuscules, trié, sans doublon ; un nom sans séparateur de chemin, une autre extension et un chunk sans
// nom n'en rendent aucun.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne pas exiger de séparateur ; garder le chemin entier ; ne pas
// mettre en minuscules ; lire sans décalage.
func TestLesNomsDeScriptSeLisentDansLeChunk0(t *testing.T) {
	cas := []struct {
		nom     string
		noms    []string
		attendu []string
	}{
		{"un mode", []string{`tags\multiplayer\lua\SlayerEvents.lua`}, []string{"slayerevents.lua"}},
		{"deux fichiers, les deux séparateurs, un doublon", []string{
			`a\ArenaFlagEvents.lua`, `b/flageventstubs.lua`, `c\arenaflagevents.lua`},
			[]string{"arenaflagevents.lua", "flageventstubs.lua"}},
		{"chiffres, tiret bas et espace", []string{`x\evenement_2 b.lua`}, []string{"evenement_2 b.lua"}},
		{"sans séparateur de chemin", []string{`slayerevents.lua`}, nil},
		{"autre extension", []string{`a\slayerevents.luac`, `a\notes.txt`}, nil},
		{"aucun nom", nil, nil},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			got := nomsDeScriptDuFlux(fluxDesNomsDeScript(chunkAvecNoms(c.noms...)))
			if !slices.Equal(got, c.attendu) {
				t.Errorf("noms %q, attendu %q", got, c.attendu)
			}
		})
	}
	if got := fluxDesNomsDeScript([]byte{1}); got != nil {
		t.Errorf("flux d'un chunk d'un octet : %v, attendu aucun", got)
	}
}
