package decfilm

import (
	"encoding/json"
	"fmt"
)

// LoadMapQuantCatalogBytes lit le catalogue des bornes de carte depuis ses octets, pour un
// catalogue embarqué dans le binaire de l'hôte. `origine` nomme la provenance dans les messages
// d'erreur (un chemin, « embarqué »...). Même contrôle de version de schéma que
// [LoadMapQuantCatalog] : un JSON invalide ou une version autre que [MapQuantSchemaVersion] est
// une erreur, jamais un catalogue vide.
//
// Le lecteur est ici et non dans la couche `profile` : y toucher, même sans changer une
// largeur, change l'empreinte de cette couche et impose une révision de profil.
func LoadMapQuantCatalogBytes(origine string, octets []byte) (*MapQuantCatalog, error) {
	var c MapQuantCatalog
	if err := json.Unmarshal(octets, &c); err != nil {
		return nil, fmt.Errorf("catalogue de bornes invalide (%s) : %w", origine, err)
	}
	if c.SchemaVersion != MapQuantSchemaVersion {
		return nil, fmt.Errorf("catalogue de bornes en version %d, attendu %d (%s)",
			c.SchemaVersion, MapQuantSchemaVersion, origine)
	}
	return &c, nil
}
