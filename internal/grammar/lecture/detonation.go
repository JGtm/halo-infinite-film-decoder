package lecture

// MessageDeDetonation est un message `projectile_detonate` (genre 5, lecteur `FUN_1408096f8`,
// [chargeDetonation]) ou `projectile_impact_effect` (genre 6, lecteur `FUN_1410f03b4`,
// [chargeImpact]) de la vue A : sa référence de projectile et sa position de détonation, quand
// la charge les a lues (grammar/vue_a_detonations.go).
//
//   - Réf. 0 (domaine 5) : la vie de l'entité projectile qui détone — LA MÊME TABLE que l'entité
//     de `ScanProjectiles` (ti=41). Absente seulement si la porte de la référence est à 0.
//   - Position ([DetonationPositionLue]) : lue par `FUN_14076e524` au niveau 0xf (détonation) ou
//     0xc (impact), déquantifiée aux bornes de la table installée sur le lecteur
//     (`br.vueA.positions`) — la table DÉFAUT du build si la porte est posée, celle de la région
//     jouée sinon ([dequantDetonationPosition]). Absente si la lecture est en pleine précision
//     (`brute`) ou si la porte ouvre un index que la table ne sait pas lire.
//
// Vingt-quatre octets : sa taille est gelée par `tailles_test.go`.
type MessageDeDetonation struct {
	// Debut est le premier bit du message : le bit de continuation qui l'annonce.
	Debut uint32
	// X, Y, Z sont la position de détonation déquantifiée, repère monde ; valides seulement si
	// [DetonationPositionLue].
	X, Y, Z float32
	// Ref0 est l'index brut de la référence 0 (domaine 5) ; Gen0 sa génération (2 bits).
	Ref0 uint16
	Gen0 uint8
	// Genre est 5 ou 6.
	Genre uint8
	// Drapeaux : [DetonationRef0], [DetonationPositionLue].
	Drapeaux uint8
}

// Les drapeaux d'un [MessageDeDetonation].
const (
	// DetonationRef0 : la porte de la référence 0 est ouverte.
	DetonationRef0 uint8 = 1 << iota
	// DetonationPositionLue : la position a été lue ET déquantifiée.
	DetonationPositionLue
)
