package replay

// identity_registry_resolution.go — LES RESOLUTIONS D UN INDEX DE PARTICIPANT, rassemblees pour la lecture directe
// des vies (sorties de `identity_registry_creation.go`).

import "github.com/JGtm/halo-infinite-film-decoder/types"

// resolutionDesIndex rassemble les resolutions d un index de participant du film : la table effective (plus l
// index des chunks hors table `chunks`), les bots declares, l entite et la declaration des index partages, les
// sieges lus dans la population de la session et le joueur aux commandes.
func resolutionDesIndex(in IdentityInput, chunks map[int]uint64) resolutionDIndex {
	res := resolutionDIndex{versXUID: indexToXUIDOf(in.PlayerIndices.ByXUID), versBot: indexDesBotsDeclares(in.Bots),
		parEntite: lireEntitesDesIndexPartages(in.Entities, in.Bots, in.PlayerIndices),
		parChunks: map[int]bool{},
		sieges:    nouveauxSiegesListes(in.Presences),
		controles: map[uint32][]types.UnitControlRead{},
		connus:    map[uint64]bool{}}
	res.parDeclaration = lireLesDeclarations(in.Bots, in.PlayerIndices, in.Entities, res.sieges)
	for j, x := range chunks {
		if _, tenu := res.versXUID[j]; !tenu {
			res.versXUID[j], res.parChunks[j] = x, true
		}
	}
	for _, x := range res.versXUID {
		res.connus[x] = true
	}
	if in.Presences.ControlStats.Scanned {
		for _, c := range in.Presences.Controls {
			res.controles[c.Slot] = append(res.controles[c.Slot], c)
		}
	}
	return res
}

// resolutionDIndex porte les resolutions d un index de participant. Une structure plutot que des
// parametres de plus : le depot borne a cinq, et elles vont toujours ensemble.
type resolutionDIndex struct {
	versXUID map[int]uint64
	versBot  map[int]bool
	// parEntite : les index que plusieurs occupants se relaient, lus par entite (lot M2.3).
	parEntite entitesDesIndexPartages
	// parDeclaration : les index que plusieurs occupants se relaient, lus par les declarations de leurs bots
	// (identity_registry_declarations.go).
	parDeclaration lectureParDeclaration
	// parChunks : les index de `versXUID` que seule la lecture des chunks attribue (index des chunks
	// hors table, identity_registry_index_des_chunks.go).
	parChunks map[int]bool
	// sieges : les plages de listage par siege ; connus : les xuids de la table d index ; controles : les lectures
	// d `unit-control` par slot (identity_registry_listages.go).
	sieges    siegesListes
	connus    map[uint64]bool
	controles map[uint32][]types.UnitControlRead
}
