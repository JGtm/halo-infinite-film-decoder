package grammar

import "github.com/JGtm/halo-infinite-film-decoder/internal/profile"

// walkDeltaBipedRecords est [walkDeltaBipedRecordsJusqua] sur chaque payload en entier : l ancrage
// sans marche, la reference des tests.
func walkDeltaBipedRecords(
	fc *FilmContext, chunks []int, slots SlotBand, lay profile.I0Layout, visit func(deltaBipedRecord),
) {
	walkDeltaBipedRecordsJusqua(fc, chunks, slots, lay, payloadEntier, visit)
}
