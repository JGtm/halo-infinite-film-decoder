package fallback

// registre_killsource_carte.go — LE REPLI DE LA RESOLUTION DE CARTE DU COLLECTEUR.
//
// L'entree repond a la question QUELLE CARTE decrit ce match : les identites candidates sont
// essayees dans l'ordre et la premiere qui resout gagne. Son site, `killcollector/map_identity.go`,
// est partage par les DEUX passes du collecteur (les positions et les distances de touche).
//
// LA DEGRADATION DES DISTANCES DE TOUCHE N'A PLUS DE SITE DANS LA BRIQUE. Une carte hors catalogue,
// ou des positions de bipedes illisibles, privent les touches de leur distance (elles restent
// comptees) : ce refus est decide et compte par l'hote (trois compteurs expvar, un par cause).
// L'entree reste au registre pour qu'un `Declenche` de l'hote se rattache a un nom connu.

var registreKillsourceCarte = []Repli{
	// `repli_carte_absente_largeurs_par_defaut` (pose au lot 3.4.1) EST RETIRE LE 2026-09-27 : son
	// critere de retrait est tenu — « 0 appel de production sans `Options.Carte` ; la CLI et les
	// instruments la resolvent eux aussi, ou disent pourquoi ils ne le peuvent pas ». Regle
	// utilisateur : « le flux du film est la seule source fiable. Pas de repli. » `killsource.Decode`
	// rend `ErrCarteAbsente` sans carte, le collecteur met le film de cote (outcome et compteur
	// dedies), la cuisson refusait deja une carte hors catalogue, `cmd/killsource` exige `-carte`,
	// et les instruments de recherche le DECLARENT (`Options.RechercheSansCarte`, interdit en
	// production par `archlint/killsource_recherche_sans_carte_test.go`).
	{
		Nom:       "repli_carte_premier_nom_resolu",
		Fait:      "quelle carte du catalogue decrit CE match",
		Mecanisme: "les identites candidates sont essayees dans l'ordre et la PREMIERE qui resout gagne, sans arbitrage",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		// SITE UNIQUE DEPUIS LE LOT 1.9.4 (2026-09-15), ET C'EST UN GAIN. La resolution vivait
		// dans `positions.go` et la passe des TOUCHES en avait une seconde, par signature de
		// largeurs d'axe. Les deux passes partagent desormais cette fonction : une seule regle
		// pour une seule question, donc un seul site a convertir le jour ou l'arbitrage existera.
		CompleteeParLHote: true,
		DatePose:          dateAudit0E,
		// POURQUOI LE LOT 1.9.4 NE LE RETIRE PAS, ALORS QU'IL EN ETAIT LA CIBLE.
		//
		// La cible ecrite au lot 1.9.0 disait « le nom de carte est resolu une fois et passe en
		// override ». Il l'est desormais — et cela ne retire pas ce repli-ci, qui porte une AUTRE
		// question : quand la base rend PLUSIEURS noms candidats, lequel decide ? Verifie sur
		// pieces (`platform/duckdb/replay_map_repo.go`, `MapKeysForMatch`) : l'ordre n'est pas
		// arbitraire, il est documente « du plus fiable au moins fiable » — le nom d'asset
		// canonique d'abord (il survit a un `map_name` reduit a un UUID), le libelle brut du
		// registre ensuite (le catalogue de modules est indexe en anglais, la cascade de langues
		// de `asset_translations` peut rendre un libelle traduit). Retirer le repli demanderait
		// d'ARBITRER entre ces deux sources, c'est-a-dire de trancher laquelle ment quand elles
		// divergent — une question de qualite du REGISTRE DES MATCHS, pas du decodeur de film,
		// et hors du perimetre d'un lot de la famille 1.9 (regle 7 : la decouverte se consigne,
		// elle ne se traite pas).
		CibleRetrait:    "un arbitrage du registre des matchs entre `asset_translations` et `match_registry.map_name`, ou un profil par carte qui rend l identite sans la base ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 match du parc ou deux noms candidats resolvent DES ENTREES DIFFERENTES du catalogue — mesure a faire avant tout arbitrage",
		CompteurBranche: true,
	},
}
