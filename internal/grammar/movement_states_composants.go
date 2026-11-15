package grammar

// movement_states_composants.go — LES ETIQUETTES DE REGISTRE DES COMPOSANTS D ETAT DE MOUVEMENT ET LA FENTE DU SPRINT.
// Deplacement pur depuis `movement_states.go` (fichier au-dela du seuil de 500 lignes) : aucune ligne de logique ne change.

// Les etiquettes de registre des trois composants, avec leurs deux orthographes : les films
// portent l une OU l autre (avec ou sans `-component`), comme toute la famille.
// sprintAbilitySlotRaw est la valeur BRUTE d `i57` qui designe la fente du sprint.
//
// DEUX, ET PAS UN : le flux lit `R(2)` et l ecrivain pose `bloc+3 = valeur - 1`
// (`FUN_142f268c4`), donc la fente `1` — celle de `'sasp'`, nommee par `FUN_14319d1ec` — se lit
// `2` dans le flux. Le decalage vit ICI, en un seul point, plutot que dans chaque lecteur.
const sprintAbilitySlotRaw = 2

// abilityComponentName / abilityComponentAlt : l etiquette de registre d `i57`, deux orthographes.
const (
	abilityComponentName = "biped-spartan-ability-component"
	abilityComponentAlt  = "biped-spartan-ability"
)

const (
	crouchComponentName    = "unit-crouch-component"
	crouchComponentNameAlt = "unit-crouch"
	slideComponentName     = "biped-slide-component"
	slideComponentNameAlt  = "biped-slide"
	mobilityComponentName  = "biped-mobility-action-component"
	mobilityComponentAlt   = "biped-mobility-action"
)

// velocityComponentName est l etiquette de registre d `i1` du bipede, la seule que le deserialiseur
// de la vitesse de translation lit et publie ([EtatVitesse], `dispatch_object.go`).
const velocityComponentName = "object-translational-velocity-dynamic-precision-component"
