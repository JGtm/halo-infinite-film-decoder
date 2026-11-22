package lecture

// MessageDeTir est un message `action_weapon_fire` (genre 36, lecteur `FUN_14080c1f8`) de la vue A : les
// champs de sa tête, ceux que lit le record de tir de tête d'une trame, mais à n'importe quel rang de la liste.
type MessageDeTir struct {
	// Rang est le rang du message dans [VueA.Genres] ; zéro pour le message de tête.
	Rang uint16
	// Unite est l'index brut de la référence 0 (l'unité qui tire), valable quand [TirUnite] ; Gen sa
	// génération (2 bits).
	Unite uint16
	Gen   uint8
	// Numero est le numéro de tir du joueur, modulo 256.
	Numero uint8
	// Tireur est l'indice de joueur (cinq bits), ou [RefAbsente] quand la garde du champ est fermée.
	Tireur int8
	// Drapeaux : [TirUnite], [TirCourt], [TirBloc].
	Drapeaux uint8
	// Arme est l'identifiant global 64 bits de l'arme.
	Arme uint64
}

// Les drapeaux d'un [MessageDeTir].
const (
	// TirUnite : la porte de la référence 0 est ouverte.
	TirUnite uint8 = 1 << iota
	// TirCourt et TirBloc sont les deux drapeaux de tête du message.
	TirCourt
	TirBloc
)
