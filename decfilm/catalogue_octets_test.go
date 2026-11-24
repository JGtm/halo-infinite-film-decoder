package decfilm_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/decfilm"
)

// TestLoadMapQuantCatalogBytesLitUnCatalogue : un catalogue embarqué se lit depuis ses octets,
// avec le même contrôle de schéma que la lecture par chemin.
func TestLoadMapQuantCatalogBytesLitUnCatalogue(t *testing.T) {
	blob := fmt.Sprintf(`{"schemaVersion":%d,"source":"test","maps":{}}`, decfilm.MapQuantSchemaVersion)
	cat, err := decfilm.LoadMapQuantCatalogBytes("embarque", []byte(blob))
	if err != nil {
		t.Fatalf("catalogue valide refusé : %v", err)
	}
	if cat.SchemaVersion != decfilm.MapQuantSchemaVersion || cat.Source != "test" {
		t.Errorf("catalogue mal relu : %+v", cat)
	}
}

// TestLoadMapQuantCatalogBytesRefuseUneAutreVersion : une version de schéma différente est une
// erreur qui nomme l'origine, jamais un catalogue vide ; un JSON invalide aussi.
// MUTATIONS — le contrôle de version retiré, l'origine absente du message : ROUGES.
func TestLoadMapQuantCatalogBytesRefuseUneAutreVersion(t *testing.T) {
	blob := fmt.Sprintf(`{"schemaVersion":%d,"maps":{}}`, decfilm.MapQuantSchemaVersion+1)
	if cat, err := decfilm.LoadMapQuantCatalogBytes("origine-x", []byte(blob)); err == nil || cat != nil ||
		!strings.Contains(err.Error(), "origine-x") {
		t.Errorf("autre version : (%v, %v), attendu une erreur nommant l'origine", cat, err)
	}
	if cat, err := decfilm.LoadMapQuantCatalogBytes("origine-y", []byte("{pas du json")); err == nil || cat != nil ||
		!strings.Contains(err.Error(), "origine-y") {
		t.Errorf("JSON invalide : (%v, %v), attendu une erreur nommant l'origine", cat, err)
	}
}
