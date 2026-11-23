package replay

// document_eliminations.go — LES ÉLIMINATIONS DE CHAQUE JOUEUR, AVEC LEUR MARQUE DE TIR À LA TÊTE ET DE FRAG PARFAIT.
//
// CE QUE C'EST. Une ligne par élimination du tableau des scores : le tueur, l'image où le compteur d'éliminations
// monte, la victime quand le film la donne, et deux marques que le jeu écrit lui-même dans ses compteurs. Rien n'est
// estimé : la marque d'un tir à la tête est la montée du compteur de tirs à la tête du tableau, la marque d'un frag
// parfait est la médaille `Perfect`.
//
// LA SOURCE DE CHAQUE CHAMP.
//   - le tueur et l'image : les pas de `scoreTimeline.players[].kills.total`. Le compteur couvre toutes les versions du
//     jeu, et chaque unité d'un pas est une élimination ; un pas de n unités donne n lignes à la même image ;
//   - la victime : l'action de score `killPersonalScore` du même joueur, à une image près (`scoreActions[]`). Le jeu ne
//     l'écrit pas pour toutes les éliminations (absente ou partielle sur les anciennes versions) : `victim` est alors
//     absent, jamais deviné ;
//   - le tueur BOT : un bot n'a pas d'identifiant et le tableau ne nomme pas sa ligne ; son élimination se lit sur l'action de score
//     `killPersonalScore` qu'il reçoit (posée à sa place, cf. document_score_actions.go), une ligne par action, à l'image de
//     l'action. Sans compteur de tirs à la tête ni médaille rattachée : `headshot` et `perfect` sont absents ;
//   - `headshot` : la montée du compteur `stats.headshots` du même joueur dans la MÊME IMAGE que la montée du compteur
//     d'éliminations. Le tableau écrit les deux compteurs dans un même enregistrement ; l'identité de l'enregistrement
//     n'est pas portée par les courbes publiées, l'image est donc la clé ;
//   - `perfect` : la médaille `Perfect` du même joueur, à deux images au plus d'un pas d'éliminations.
//
// TROIS ÉTATS, PAS DEUX. `headshot` et `perfect` sont des pointeurs : vrai, faux, ou ABSENT (inconnu). Un inconnu ne se
// lit jamais comme un faux.
//
// QUAND `headshot` VAUT FAUX. Le joueur a un compteur de tirs à la tête publié, et ce compteur ne monte pas à l'image
// du pas. Il vaut VRAI quand le compteur monte autant que le pas d'éliminations. Il est INCONNU :
//   - sans compteur de tirs à la tête pour ce joueur : une courbe absente ne prouve pas un compteur resté à zéro (elle
//     est aussi celle d'un compteur dont toutes les émissions ont été écartées) ;
//   - quand le pas compte plus d'éliminations que de montées (on ne sait pas lesquelles sont à la tête) ou moins
//     (montée impossible) ;
//   - quand la garde des compteurs a écarté des émissions du compteur de tirs à la tête dans ce film
//     (`coverage.score.statsGuarded`) : une émission retirée peut cacher la montée d'une élimination, le faux n'est
//     plus prouvé. Une montée conservée reste une preuve du vrai.
//
// QUAND `perfect` VAUT VRAI. Une médaille `Perfect` du joueur se rattache à l'unique pas d'éliminations du joueur situé à
// deux images au plus ; le pas se marque quand il porte autant de médailles que d'éliminations. Une médaille sans pas
// candidat, ou avec plusieurs, ne marque rien ; un pas qui porte un autre nombre de médailles que d'éliminations ne
// marque rien. `perfect` ne vaut JAMAIS faux : l'absence d'une médaille ne prouve pas que le film les liste toutes.
//
// CLÉ DE JOINTURE POUR UN CONSOMMATEUR qui a ses propres éliminations : le tueur (`xuid`), la victime (`victim`) et
// l'image `t`, à une image près ; sans victime, le tueur et l'image.

// Elimination est UNE élimination d'un joueur.
type Elimination struct {
	// T est l'index de frame où le compteur d'éliminations du joueur monte, sur le même axe que Point.T.
	T int `json:"t"`
	// XUID est le tueur humain, en décimal — même forme que Track.XUID. Absent quand le tueur est un bot : Place le désigne.
	XUID string `json:"xuid,omitempty"`
	// Place est l'index de joueur du film du tueur quand c'est un bot, qui n'a pas d'identifiant : le même espace que
	// `roster[].filmIndex` et que `reloads[].place`. Absent pour un tueur humain ; XUID et Place s'excluent.
	Place *int `json:"place,omitempty"`
	// Victim est la victime humaine, en décimal. Absente quand le film ne la donne pas pour cette élimination, ou quand
	// la victime est un bot (VictimPlace).
	Victim string `json:"victim,omitempty"`
	// VictimPlace est l'index de joueur du film de la victime quand c'est un bot (même espace que Place) ; Victim et
	// VictimPlace s'excluent.
	VictimPlace *int `json:"victimPlace,omitempty"`
	// Headshot : l'élimination est un tir à la tête (vrai), n'en est pas un (faux), ou on ne sait pas (absent). Cf. l'en-tête.
	Headshot *bool `json:"headshot,omitempty"`
	// Perfect : l'élimination est un frag parfait (vrai) ou on ne sait pas (absent) ; jamais faux. Cf. l'en-tête.
	Perfect *bool `json:"perfect,omitempty"`
}

// EliminationCoverage dit ce que le calque des éliminations a posé, et ce qu'il n'a pas su marquer.
//
// Invariants : `eliminations` = `headYes` + `headNo` + `headUnknown` = `perfectYes` + `perfectUnknown` ;
// `headUnknown` = `headNoCounter` + `headAmbiguous` + `headGuarded`.
type EliminationCoverage struct {
	// Eliminations est le nombre de lignes publiées ; WithVictim, celles dont la victime est donnée.
	Eliminations int `json:"eliminations"`
	WithVictim   int `json:"withVictim"`
	// BotEliminations est la part de `eliminations` dont le tueur est un bot (lignes à `place`, sans xuid) : lues sur les
	// actions de score du bot, jamais sur le tableau, qui ne nomme pas ses lignes. VictimPlaces, la part de `withVictim` dont
	// la victime est un bot (`victimPlace`). Absents à zéro.
	BotEliminations int `json:"botEliminations,omitempty"`
	VictimPlaces    int `json:"victimPlaces,omitempty"`
	// HeadYes, HeadNo, HeadUnknown : les trois états de `headshot`.
	HeadYes     int `json:"headYes"`
	HeadNo      int `json:"headNo"`
	HeadUnknown int `json:"headUnknown"`
	// HeadNoCounter compte les inconnus d'un joueur sans compteur de tirs à la tête ; HeadAmbiguous, ceux d'un pas dont le
	// nombre de montées ne vaut pas le nombre d'éliminations ; HeadGuarded, ceux qui seraient faux sans les émissions
	// que la garde des compteurs a écartées.
	HeadNoCounter int `json:"headNoCounter"`
	HeadAmbiguous int `json:"headAmbiguous"`
	HeadGuarded   int `json:"headGuarded"`
	// HeadOrphanRises compte les unités de montée du compteur de tirs à la tête qu'aucune élimination ne porte : montée à
	// une image sans pas d'éliminations, ou au-delà des éliminations du pas.
	HeadOrphanRises int `json:"headOrphanRises"`
	// PerfectYes et PerfectUnknown : les deux états publiés de `perfect`.
	PerfectYes     int `json:"perfectYes"`
	PerfectUnknown int `json:"perfectUnknown"`
	// PerfectAmbiguous compte les médailles `Perfect` qui n'ont marqué aucune élimination parce que plusieurs pas les
	// réclament, ou que leur pas porte un autre nombre de médailles que d'éliminations ; PerfectOrphans, celles sans pas
	// à deux images (ou d'un joueur sans éliminations publiées).
	PerfectAmbiguous int `json:"perfectAmbiguous"`
	PerfectOrphans   int `json:"perfectOrphans"`
}
