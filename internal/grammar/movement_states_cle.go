package grammar

// movementStateKey deduplique une lecture : le chemin d inference de la marche ([decodeInferLoop])
// re-parcourt un record quand une chaine de transitoires le demande, et le deserialiseur
// re-publie alors la MEME transition. Compter deux fois la meme gonflerait les denominateurs
// sans qu aucun compteur ne le dise.
type movementStateKey struct {
	slot uint32
	kind string
	ts   uint64
}
