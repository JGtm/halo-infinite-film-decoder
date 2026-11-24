package replay

// document_apparence.go — L'APPARENCE DES SPARTANS DANS LE ROSTER : le bloc de personnalisation de chaque humain, en codes bruts
// (`roster[].appearance`), et ce que sa lecture a rencontre (`coverage.appearance`).
//
// LE CONTRAT : des CODES OPAQUES. Le document ne dit pas ce que designe un code (piece d'armure, revetement, couleur...) : la
// table de noms est hors du decodeur. Un code est stable pour un joueur d'un film a l'autre tant qu'il ne change pas
// d'equipement. Chaque octet du bloc est publie UNE fois, dans `pairs` ou dans `rest`.

import (
	"log/slog"
	"strconv"

	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// Appearance est l'apparence d'un humain : le dernier bloc de personnalisation non nul que les paquets de type 8 portent pour lui.
type Appearance struct {
	// Size est la largeur du bloc en octets pour le build du film. La queue de zeros de `rest` est omise : elle se retrouve en
	// completant de zeros jusqu'a `size`, moins 8 octets de tete et 8 octets par paire quand `pairs` est present.
	Size int `json:"size"`
	// Pairs sont les paires (cle, valeur) de 32 bits dans l'ordre du film ; les cles sont les memes pour tous les joueurs d'un
	// build, les valeurs sont propres au joueur. Absent quand la structure du bloc ne se lit pas : tout le bloc est alors dans `rest`.
	Pairs [][2]uint32 `json:"pairs,omitempty"`
	// Rest sont les octets du bloc qui ne sont pas dans `pairs` (base64 en JSON), sans leur queue de zeros.
	Rest []byte `json:"rest,omitempty"`
}

// AppearanceCoverage dit ce que la lecture de l'apparence a rencontre. Absent quand la lecture n'a pas eu lieu.
type AppearanceCoverage struct {
	// Pairs est le nombre de joueurs dont l'apparence est lue en paires ; Raw ceux dont le bloc est publie entier parce que sa tete
	// ne se lit pas ; Unloaded ceux dont tous les blocs sont nuls (apparence non chargee : aucun objet publie) ; Changed ceux dont
	// le bloc a change entre deux paquets du film (le dernier bloc non nul fait foi).
	Pairs    int `json:"pairs"`
	Raw      int `json:"raw"`
	Unloaded int `json:"unloaded"`
	Changed  int `json:"changed"`
	// UnknownBuild : le build du film n'est pas dans le profil, aucune apparence n'est lue.
	UnknownBuild bool `json:"unknownBuild"`
	// Packets, Distinct et Refused comptent les paquets de type 8 : tous, ceux lus, ceux ecartes. NoXUID compte les entrees sans joueur.
	Packets  int `json:"packets"`
	Distinct int `json:"distinct"`
	Refused  int `json:"refused"`
	NoXUID   int `json:"noXUID"`
	// Unmatched est le nombre d'apparences lues dont le joueur n'a pas d'entree au roster publie.
	Unmatched int `json:"unmatched"`
}

// poserLesApparences pose l'apparence de chaque humain sur son entree de roster (meme xuid) et range la couverture dans
// `apparenceCov` : `doc.Coverage` n'existe qu'a la composition de la couverture, qui la pose. Sans lecture faite, rien n'est pose.
func (a *assemblage) poserLesApparences() {
	st := a.opt.AppearanceStats
	if !st.Scanned {
		return
	}
	cov := AppearanceCoverage{Pairs: st.Pairs, Raw: st.Raw, Unloaded: st.Unloaded, Changed: st.Changed, UnknownBuild: st.UnknownBuild,
		Packets: st.Packets, Distinct: st.Distinct, Refused: st.Refused, NoXUID: st.NoXUID}
	parXUID := make(map[string]int, len(a.doc.Roster))
	for i, e := range a.doc.Roster {
		if e.XUID != "" {
			parXUID[e.XUID] = i
		}
	}
	for _, ap := range a.opt.Appearances {
		i, ok := parXUID[strconv.FormatUint(ap.XUID, 10)]
		if !ok {
			cov.Unmatched++
			continue
		}
		a.doc.Roster[i].Appearance = appearanceDuDocument(ap)
	}
	a.apparenceCov = &cov
	slog.InfoContext(a.ctx, "rejeu : apparence", "paires", cov.Pairs, "brut", cov.Raw, "nonChargees", cov.Unloaded,
		"changees", cov.Changed, "buildInconnu", cov.UnknownBuild, "sansEntree", cov.Unmatched)
}

// appearanceDuDocument rend l'apparence publiee d'un bloc lu.
func appearanceDuDocument(ap types.PlayerAppearance) *Appearance {
	out := &Appearance{Size: ap.Size, Rest: ap.Rest}
	if len(ap.Pairs) > 0 {
		out.Pairs = make([][2]uint32, len(ap.Pairs))
		for i, p := range ap.Pairs {
			out.Pairs[i] = [2]uint32{p.Key, p.Value}
		}
	}
	return out
}
