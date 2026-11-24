package grammar

// keyframe_etats_predecodage.go — LES MOTS PAR RECORD, CALCULES SUR PLUSIEURS FILS AU DEBUT DU
// BALAYAGE DES ETATS.
//
// La fenetre de bits derriere la lecture ([canalDeLEtatCompletBipede.fenetresDerriereLaLecture])
// demande, pour chaque payload d image-cle qui porte un record bipede non admis, les familles
// d arme et la marque de portage de ses records ([motsParRecord]) : un passage bit a bit du payload,
// pur, et le plus gros des trois jeux de fenetres. Quand le contexte permet plusieurs fils
// ([FilmContext.Fils]), [canalDeLEtatCompletBipede.predecoderLesMots] fait ces passages EN
// PARALLELE avant la distribution, et la fenetre lit le resultat range dans le canal au lieu de le
// calculer. A un fil, rien n est pre-calcule et rien ne change.
//
// CE QUE LE PRE-CALCUL COUVRE. Tout payload d image-cle des chunks du film qui porte au moins un
// record bipede : l admission d un record se decide plus tard, par la lecture de son corps, et le
// pre-passage ne peut pas la connaitre. Le passage de mots calcule donc aussi les payloads dont tous
// les bipedes seront admis (travail perdu, jamais lu). Un payload sans bipede n est pas calcule : la
// fenetre n a rien a lui demander.
//
// POURQUOI LE RESULTAT EST CELUI DE LA FENETRE. Un record du payload a pour mots ceux dont la
// fenetre de 32 bits COMMENCE dans son etendue ; l etendue d un record se lit sur les seuls debuts
// de records ([recordContaining]), et le type d un record ne sert qu a filtrer la sortie. La
// fenetre rend « muets » les records admis (type non resolu) ; le pre-passage ne les rend pas
// muets et calcule donc aussi leurs mots, que personne ne lit : la fenetre ne consulte que les
// records non admis, par leur debut, et le mot d un record non admis ne depend pas du mutisme des
// autres. Le debut d un record est unique dans un payload (la marche avance strictement).
//
// OU LE RESULTAT VIT. Dans le canal, jamais dans le contexte : il depend des jeux de mots de CET
// appel (le catalogue des familles recu, le seul jeu de l option « armes seules »). Il est range
// depuis le fil appelant, apres la jointure, et vide a la cloture du canal.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

// motsDuPayload est ce que la fenetre tire d UN payload : les familles par debut de record et les
// debuts de record porteurs de la marque de portage, pour TOUS les records bipedes du payload.
type motsDuPayload struct {
	// nbRecords : le nombre de records du payload quand le calcul a ete fait ; la fenetre n emploie
	// le resultat que pour un paquet de meme compte, sinon elle recalcule.
	nbRecords int
	familles  map[uint32][]uint32
	marques   map[uint32]bool
}

// jeuxDeMots rend les jeux de mots cherches d un passage : les familles d arme, puis la marque de
// portage sauf pour les armes seules.
func (c *canalDeLEtatCompletBipede) jeuxDeMots() []map[uint32]bool {
	jeux := []map[uint32]bool{c.known}
	if !c.armesSeules {
		jeux = append(jeux, carrierMarkViews)
	}
	return jeux
}

// famillesEtMarques range la sortie de [motsParRecord] comme la fenetre la lit : les familles par
// debut de record, et les debuts de record porteurs de la marque (vide sans le jeu de la marque).
func famillesEtMarques(mots [][]recordFamilies) (map[uint32][]uint32, map[uint32]bool) {
	familles, marques := map[uint32][]uint32{}, map[uint32]bool{}
	for _, rf := range mots[0] {
		familles[rf.Rec.Debut] = rf.Families
	}
	if len(mots) > 1 {
		for _, rf := range mots[1] {
			marques[rf.Rec.Debut] = true
		}
	}
	return familles, marques
}

// predecoderLesMots calcule les mots de chaque payload d image-cle du film sur les fils du contexte
// et les range dans le canal. A un fil, ou sans payload a calculer, il ne fait rien et ne lance
// aucune goroutine.
func (c *canalDeLEtatCompletBipede) predecoderLesMots() {
	if c.fc == nil || c.fc.Fils() <= 1 { // un contexte nil se lit comme avant : sans pre-calcul
		return
	}
	n := c.fc.Fils()
	charges := c.fc.payloadsDesImagesCles()
	if len(charges) == 0 {
		return
	}
	marche, jeux := c.fc.MarcheDImageCle(), c.jeuxDeMots()
	calcules := make([]*motsDuPayload, len(charges))
	executerEnParallele(c.fc.suivi, n, len(charges), func(i int) {
		calcules[i] = motsDuPayloadDe(charges[i], marche, jeux)
	})
	// Depuis le fil appelant, apres la jointure : aucune ecriture partagee pendant le calcul.
	c.motsAvance = make(map[clePayload]motsDuPayload, len(charges))
	for i, m := range calcules {
		if m != nil {
			c.motsAvance[clePayload{debut: &charges[i][0], n: len(charges[i])}] = *m
		}
	}
}

// motsDuPayloadDe calcule les mots d UN payload : ses ancres (le memo du contexte quand elles y sont,
// sinon la marche, comme la distribution les lirait), puis un passage de mots pour tous ses records.
// Nil quand le payload ne porte aucun record bipede. Elle ne touche aucun etat du canal.
func motsDuPayloadDe(pay []byte, marche MarcheDImageCle, jeux []map[uint32]bool) *motsDuPayload {
	bornes := keyframeBornesDe(marche.Marcher(pay).Records)
	recs, bipede := make([]lecture.Record, len(bornes)), false
	for i, b := range bornes {
		recs[i] = lecture.Record{TI: int16(b.TI), Debut: uint32(b.Bit)} //nolint:gosec // archetype sur six bits, position d un payload
		bipede = bipede || b.TI == keyframeBipedTI
	}
	if !bipede {
		return nil
	}
	familles, marques := famillesEtMarques(motsParRecord(pay, recs, keyframeBipedTI, jeux...))
	return &motsDuPayload{nbRecords: len(recs), familles: familles, marques: marques}
}

// motsDuPaquet rend les familles et les marques que la fenetre lit pour le paquet `p`, dont `recs`
// est la copie des records (les admis rendus muets) : le resultat pre-calcule quand il existe pour ce
// payload et ce nombre de records, sinon le calcul direct. Un resultat pre-calcule se consomme une
// fois.
func (c *canalDeLEtatCompletBipede) motsDuPaquet(p *lecture.Paquet, recs []lecture.Record) (map[uint32][]uint32, map[uint32]bool) {
	if len(c.motsAvance) > 0 && len(p.Payload) > 0 {
		cle := clePayload{debut: &p.Payload[0], n: len(p.Payload)}
		if m, ok := c.motsAvance[cle]; ok {
			delete(c.motsAvance, cle)
			if m.nbRecords == len(recs) {
				return m.familles, m.marques
			}
		}
	}
	return famillesEtMarques(motsParRecord(p.Payload, recs, keyframeBipedTI, c.jeuxDeMots()...))
}
