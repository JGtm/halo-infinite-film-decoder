package grammar

// golden_minibobine_proprietes_test.go — LES DEUX LECTURES DES PROPRIETES RESEAU DE ti=13 DANS LE GOLDEN DE LA MINI-BOBINE :
// la passe qui ancre les records (`managedProperties`) et la marche des trames (`managedPropertiesMarche`). La seconde
// echoue si le canal de la marche disparait ou lit autrement ; la premiere, si la passe change.

func famillesProprietesGerees(r *recueil, fc *FilmContext) {
	props, err := ScanManagedProperties(fc)
	var premier any
	if len(props.Reads) > 0 {
		premier = props.Reads[0]
	}
	r.ajouter("managedProperties", len(props.Reads), props, premier, err)

	marche, err := ScanMarcheDesTrames(fc)
	valeurs := marche.ProprietesGerees
	var premiere any
	if len(valeurs.Lectures) > 0 {
		premiere = valeurs.Lectures[0]
	}
	r.ajouter("managedPropertiesMarche", len(valeurs.Lectures), valeurs, premiere, err)
}
