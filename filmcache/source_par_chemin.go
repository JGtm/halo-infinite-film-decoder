package filmcache

// source_par_chemin.go : une [Source] construite hors de la disposition du cache. Les outils qui
// recoivent un manifeste par chemin (`--manifest <fichier>`) et un repertoire de chunks
// quelconque n'ont pas de paire (racine, short8) : en inventer une serait recopier la
// disposition du cache. Le format lu est celui d'[Open] : une seule forme de manifeste.

import (
	"encoding/json"
	"fmt"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// ParseManifest rend l'index d'un manifeste (format d'[Open]) sans ouvrir de chemin de cache.
// Une structure inattendue est une erreur, jamais un manifeste vide : un manifeste present mais
// illisible n'est pas un film absent du cache. Les tailles `size_bytes` ne sont pas rendues ici :
// pour garder le controle de troncature, passer par [OpenManifest].
func ParseManifest(raw []byte) ([]types.ChunkMeta, error) {
	var mf manifestJSON
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, fmt.Errorf("filmcache: manifeste de film invalide : %w", err)
	}
	return mf.source("", "", "").chunks, nil
}

// NewSource construit une [Source] dont les chunks se lisent dans `chunkDir`. `chunks` donne
// l'index (`Meta()[i]` decrit le chunk d'indice `i` et porte son numero de fichier en
// [types.ChunkMeta.Index]). Aucune taille n'est connue : [Source.Chunk] lit sans controle de
// troncature et [Source.TailleDeclaree] rend 0. Pour un manifeste qui porte `size_bytes`,
// preferer [OpenManifest].
func NewSource(chunks []types.ChunkMeta, chunkDir string) *Source {
	return &Source{chunks: chunks, tailles: make([]int64, len(chunks)), direct: chunkDir}
}

// OpenManifest construit la [Source] d'un manifeste donne par ses octets, dont les chunks se
// lisent dans `chunkDir`. Contrairement a [NewSource], les tailles `size_bytes` du manifeste sont
// retenues : un fichier d'une autre taille rend [ErrChunkTronque], comme pour [Open]. Une
// structure invalide est une erreur.
func OpenManifest(raw []byte, chunkDir string) (*Source, error) {
	var mf manifestJSON
	if err := json.Unmarshal(raw, &mf); err != nil {
		return nil, fmt.Errorf("filmcache: manifeste de film invalide : %w", err)
	}
	return mf.source("", "", chunkDir), nil
}
