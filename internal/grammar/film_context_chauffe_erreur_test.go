package grammar

import (
	"errors"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
)

// TestChaufferRendLesErreursDuDecoupageEtDuRegistre : sur un film sans chunk de registre, la chauffe va
// jusqu'au bout et rend l'erreur que les lecteurs retrouvent dans le memo ; un contexte nil n'en rend aucune.
func TestChaufferRendLesErreursDuDecoupageEtDuRegistre(t *testing.T) {
	fc := NewFilmContext(&source.Film{})
	err := fc.Chauffer()
	if !errors.Is(err, ErrNoRegistryChunk) {
		t.Fatalf("erreur de chauffe = %v, attendu %v", err, ErrNoRegistryChunk)
	}
	if _, errLue := fc.Registry(); !errors.Is(errLue, ErrNoRegistryChunk) {
		t.Errorf("le lecteur ne retrouve pas l'erreur memorisee : %v", errLue)
	}
	if err := (*FilmContext)(nil).Chauffer(); err != nil {
		t.Errorf("contexte nil : %v", err)
	}
}
