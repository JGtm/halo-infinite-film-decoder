package replay

// SchemaVersion est incrémenté quand la forme de ReplayDocument change d'une façon que le
// client web doit gérer. L'ajout de champs OPTIONNELS (omitempty) ne casse pas le client
// et n'incrémente pas la version ; seul un changement cassant le fait.
//
// EXCEPTION ASSUMÉE AU 2026-09-08 (v49, lot M1b, décision utilisateur ferme) : le champ ajouté
// (`coverage.bridge.deathOffsetMs`) est bien optionnel — le web sait déjà distinguer « connu »
// de « absent » sur le seul pointeur, sans regarder cette version. Le bump n'est donc pas
// exigé par la RÈGLE ci-dessus ; il est posé quand même pour que `backfill-replay` recuise
// tout le parc < 49 en une passe (plutôt que de laisser un artefact ancien répondre « calage
// inconnu » indéfiniment faute de recuisson déclenchée).
//
// LA CHRONIQUE FAIT FOI, ET ELLE EST LA SEULE : `document_chronicle.go` (les notes par version qui
// la recopiaient ici ont ete retirees le 2026-09-14, lot 1.0, revue R1, constat R1-4).
//
// CE QUE LE GARDE-RAIL TIENT, EXACTEMENT (revue R2, constat R2-1) : `document_shape_test.go`
// exige une entree de chronique pour la version COURANTE, au moment ou la forme se refige. Il ne
// balaie PAS les versions passees — c'est `replaybuild/artifact_schema_history_test.go` qui les
// rejoue, sur la liste que la chronique DECLARE. Une version cuite dont l'entree manque echappe
// donc aux deux : le retrait de ces notes-ci a fait disparaitre la seule description de la v51,
// restauree a la chronique le meme jour. Une entree de chronique se pose DANS LE COMMIT qui
// monte la version, jamais apres.
const SchemaVersion = 126
