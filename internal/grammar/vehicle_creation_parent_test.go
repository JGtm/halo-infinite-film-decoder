package grammar

import "testing"

// TestParentDeCreationSansLecture — sans i10 au masque, ou avec un i10 lu DETACHE, le record de creation ne
// nomme aucun parent : l'appelant n'a aucune lecture, jamais un parent zero.
func TestParentDeCreationSansLecture(t *testing.T) {
	arch := Archetype{Components: []string{"object-position", compObjectParentState}, Levels: []uint32{0, 3}}
	prof := ProfilDeBalayageParDefaut()
	pay := make([]byte, 16) // que des zeros : porte « attache » fermee, queue vide
	if _, ok := readCreationParent(pay, 0, []int{0}, arch, prof); ok {
		t.Error("masque sans i10 : une lecture est rendue")
	}
	if _, ok := readCreationParent(pay, 0, []int{1}, arch, prof); ok {
		t.Error("i10 detache : une lecture est rendue")
	}
}
