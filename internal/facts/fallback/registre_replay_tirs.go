package fallback

// registre_replay_tirs.go — les replis du TIREUR et de LA VISEE D UN TIR publies au rejeu
// (`replay/tirs_par_place.go`, `replay/shots_visee.go`).
//
// LA LECTURE QUE CE REPLI REMPLACE QUAND ELLE MANQUE. La direction d un tir est la visee i21 de son
// tireur, la derniere lecture du slot au plus tard a l instant du tir, dans la meme vie. Le champ de
// visee du record de tir 36 ne sert qu a defaut : il n est ecrit que sous les deux drapeaux nuls qui
// le precedent, et la ou il l est il rend la visee i21 du meme paquet a moins d un degre.
var registreReplayTirs = []Repli{
	{
		Nom:  NomViseeDuTirLueDansLeRecord,
		Fait: "la direction d un tir publiee au rejeu (`Shot.H`, `Shot.P`)",
		Mecanisme: "le champ de visee du record de tir 36 (grammar.FireEvent.Aim, lu sous ses deux drapeaux " +
			"nuls) sert de direction quand le slot tireur n a aucune lecture i21, avec ou sans position, au " +
			"plus tard a l instant du tir et a moins de 5 s",
		// LA LECTURE QUI MANQUE est la visee i21 du tireur : un slot qui n en replique aucune dans la
		// vie du tir, ou un occupant de vehicule sur un film ou les visees sans position ne sont pas
		// lues (le balayage ne tourne que sur un film a vehicules).
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{Fichier: pkgReplay + "shots_visee.go", Ancre: "func (v *viseurDesTirs) viser("},
		},
		DatePose:     "2026-11-13",
		CibleRetrait: "sans objet pour l instant : le repli ne couvre que les tirs dont le tireur ne replique aucune visee i21 tenable",
		CritereRetrait: "coverage.shots.aim.record mesure a 0 sur le parc complet une fois recuit : alors le repli " +
			"n est plus jamais le chemin d un tir",
		CompteurBranche: true,
	},
	{
		Nom:  NomTireurParLeSiegeDeLaPlace,
		Fait: "le joueur qui tient la place de controle d un tir ou d une rafale (`FireEvent.FilmIndex`, `ContinuousFireBurst.PlayerIndex`)",
		Mecanisme: "le siege du roster (l entree unique dont un intervalle de presence tenu a la place " +
			"couvre l instant, `tireursParPlace.occupant`) rend le tireur quand la lecture des places " +
			"dans les tirs ne tient la place a personne a cet instant ; a defaut d occupant unique, la " +
			"place elle-meme, comme avant la lecture des places",
		// LA LECTURE QUI TOURNE ET NE TRANCHE PAS : les tirs dont l unite nomme un occupant lient la
		// place a un joueur ; une frame hors des tenues que ces liaisons bornent n est tenue par
		// personne (place jamais liee, ecart que deux presences se disputent, film dont l index de
		// tireur n est pas la place).
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{Fichier: pkgReplay + "tirs_par_place.go", Ancre: "horloge.fb.DeclencheN(fallback.NomTireurParLeSiegeDeLaPlace, sansLecture)"},
			{Fichier: pkgReplay + "fire_bursts.go", Ancre: "clock.fb.Declenche(fallback.NomTireurParLeSiegeDeLaPlace)"},
		},
		DatePose:     "2026-11-23",
		CibleRetrait: "la pose des sieges du roster par les liaisons lues dans les tirs, qui rendrait la lecture des places et le siege identiques",
		CritereRetrait: "coverage.controlPlaces.shotsBySeat, shotsUnheld, burstsBySeat et burstsUnheld a 0 sur le corpus du " +
			"gate de rejeu, hors les films dont l index de tireur n est pas la place (un temoin par build)",
		CompteurBranche: true,
	},
}
