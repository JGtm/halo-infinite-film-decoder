package replay

// film_scan_voies.go — LES VOIES DU BALAYAGE : DES LECTURES D APRES LA MARCHE QUI TOURNENT EN MEME
// TEMPS QUE LA CHAINE PRINCIPALE, ET SONT PUBLIEES DANS SON ORDRE.
//
// # CE QUE C EST
//
// A plus d un fil, apres la marche des trames, deux files de lectures partent chacune sur sa goroutine et
// sur un CONTEXTE DE VOIE ([grammar.FilmContext.Deriver]) qui n est qu a elle :
//
//   - la voie du monde (film_scan_voie_monde.go) : lunette, poses, pieces engendrees, socles, vehicules,
//     puis lancers, projectiles et detonations ;
//   - la voie des sieges (film_scan_voie_sieges.go) : equipes et occupants, entrees du tableau des scores.
//
// Le fil principal garde la chaine d aujourd hui, dans son ordre, sur le contexte du film. A l endroit
// exact ou il lisait un balayage confie a une voie, il attend sa valeur ([attendreVoie]) puis appelle le
// `publier…` inchange : journal, versement des replis, entrees, etape observee, tout reste sur le fil
// principal, donc dans l ordre d un fil. Une voie ne publie, ne journalise, ne verse et n observe rien :
// elle lit, et range ses valeurs.
//
// # QUI ECRIT QUOI, DEPUIS QUEL FIL
//
//   - le fil principal prepare tout : [filmScan.demarrerLesVoies] chauffe le contexte
//     ([grammar.FilmContext.Chauffer]), verifie qu il est chaud, derive les deux voies, puis les lance.
//     Chaque voie est ensuite l unique lectrice et ecrivaine de son contexte derive (replis, diagnostics,
//     profil de balayage, observation, memos des pistes et des creations du monde) ;
//   - les memos chauds du parent sont lus par tous, ecrits par personne ;
//   - une valeur d une voie s ecrit une fois, par sa goroutine, AVANT la fermeture de son canal ; le fil
//     principal ne la lit qu apres avoir recu de ce canal ;
//   - a la fin du balayage, depuis le fil principal et apres avoir attendu la fin de chaque voie, les
//     voies sont fusionnees dans le parent, dans un ordre fixe (monde, puis sieges) : les replis
//     s ajoutent, les diagnostics se versent a la suite, les memos du monde passent au parent
//     ([grammar.FilmContext.Fusionner]). Un balayage qui sort avant la fin ne fusionne rien : ce que les
//     voies ont lu d avance et que le fil principal n a pas publie n entre dans aucun compte.
//
// # QUAND ON NE LANCE RIEN
//
// A un fil, ou si la marche des trames a echoue, ou si le contexte n est pas chaud apres [Chauffer] :
// aucune goroutine, aucune derivation, et les `lire…` lisent en serie sur le contexte du film, comme avant.
// [Chauffer] n est meme pas appelee a un fil ou sur une marche en erreur. Une marche en erreur relance, dans
// plusieurs balayages, une marche de repli que seul le contexte principal sait mener.
//
// # SORTIE ANTICIPEE, PANIQUE
//
// Le balayage differe [voiesDuBalayage.joindre] : il demande aux voies de s arreter entre deux lectures,
// puis les attend toutes — aucune goroutine ne survit au balayage, quelle que soit sa sortie. Une panique
// d une voie est capturee et rejouee, telle quelle, sur le fil principal, une seule fois : a l attente de
// la premiere valeur que la voie n a pas pu rendre, ou a la jointure si personne ne l attendait.
//
// # POURQUOI UN LANCEMENT A PART ET PAS `executerEnParallele`
//
// L assistant de la grammaire joint ses taches avant de rendre la main : le fil principal ne pourrait pas
// derouler sa chaine pendant que les voies lisent, ce qui est tout l interet. Ce fichier porte donc son
// propre `go`, le deuxieme de `replay` (garde-rail : `depottest.TestUneSeuleSourceDeGoroutines`), avec le
// meme relais de panique que la tache de fond. Il n est pas dans `fichiersDuBalayage` : les lectures de ses
// voies sont les memes appels que ceux des `lire…` de la chaine (cf. `TestLesLecturesDesVoiesSontCellesDeLaChaine`),
// et le garde des etapes les compterait deux fois.

import (
	"errors"
	"sync/atomic"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// errVoieInterrompue signale une voie arretee sans panique (runtime.Goexit) avant la fin de ses lectures.
var errVoieInterrompue = errors.New("voie du balayage interrompue avant la fin de ses lectures")

// errVoieArretee est ce que rend une valeur qu une voie n a pas lue parce que le balayage l a arretee :
// personne ne l attend plus, c est une garde.
var errVoieArretee = errors.New("voie du balayage arretee avant cette lecture")

// voie est une file serielle de lectures, sur sa goroutine et son contexte derive.
type voie struct {
	// fc : le contexte de voie, ecrit par la seule goroutine de la voie jusqu a `fini`.
	fc *grammar.FilmContext
	// fini se ferme une fois, par la goroutine de la voie, quand elle a rendu la main (panique comprise).
	fini chan struct{}
	// arret : le fil principal demande l arret entre deux lectures.
	arret atomic.Bool
	// panique : celle qui a interrompu la voie, ecrite avant la fermeture de `fini`. Nil si la voie s est
	// terminee, ou arretee sur demande.
	panique any
	// clotures ferment, avec la panique qui a interrompu la voie, les valeurs qu elle n a pas rendues.
	clotures []func(p any)
}

// livrer range la valeur d une lecture et la rend a ses lecteurs. Fil de la voie seulement.
func livrer[T any](a *attendue[T], v T) {
	a.val = v
	a.clore(nil)
}

// nouvelleAttendue ouvre une valeur que la voie `l` rendra, et enregistre sa cloture.
func nouvelleAttendue[T any](l *voie) *attendue[T] {
	a := &attendue[T]{pret: make(chan struct{})}
	l.clotures = append(l.clotures, a.clore)
	return a
}

// arretee dit si le balayage a demande l arret de la voie. Consultee entre deux lectures.
func (l *voie) arretee() bool { return l.arret.Load() }

// lancer demarre la goroutine de la voie : `corps` enchaine ses lectures, dans son ordre.
func (l *voie) lancer(corps func()) {
	l.fini = make(chan struct{})
	go func() {
		complet := false
		defer func() {
			p := recover()
			if p == nil && !complet && !l.arretee() {
				p = errVoieInterrompue
			}
			l.panique = p
			cloture := p
			if cloture == nil {
				cloture = errVoieArretee
			}
			for _, c := range l.clotures {
				c(cloture)
			}
			close(l.fini)
		}()
		corps()
		complet = true
	}()
}

// voiesDuBalayage porte les voies d un balayage. Un pointeur nil est l absence de voie : tous les
// accesseurs rendent alors « pas de valeur d avance » et les lectures se font en serie.
type voiesDuBalayage struct {
	parent *grammar.FilmContext
	monde  *voieDuMonde
	sieges *voieDesSieges
	// relayee dit qu une panique a deja ete rejouee : le fil principal seul la lit et l ecrit.
	relayee bool
}

// lesVoies rend les voies dans l ordre fixe de la fusion.
func (v *voiesDuBalayage) lesVoies() []*voie { return []*voie{v.monde.voie, v.sieges.voie} }

// demarrerLesVoies lance les voies quand le balayage peut les tenir, et rend nil sinon. Fil principal,
// juste apres la marche des trames : les voies partent avant les positions et lisent pendant que la
// chaine principale les publie.
//
// Conditions : plus d un fil (donc la tache de fond, qui rend la table des joueurs), marche des trames sans
// erreur, contexte chaud apres [grammar.FilmContext.Chauffer].
// Chauffer ne se lance qu a ces deux premieres conditions : a un fil le contexte n est pas touche.
func (s *filmScan) demarrerLesVoies() *voiesDuBalayage {
	if s.fc.Fils() <= 1 || s.errMarche != nil || s.fond == nil {
		return nil
	}
	s.fc.Chauffer()
	if !s.fc.EstChaud() {
		return nil
	}
	v := &voiesDuBalayage{parent: s.fc}
	v.monde = lancerLaVoieDuMonde(s.fc.Deriver(), s.world)
	v.sieges = lancerLaVoieDesSieges(s.fc.Deriver())
	return v
}

// attendreVoie rend la valeur de `a`, et rejoue sur le fil principal la panique qui a empeche la voie de la
// produire (une seule fois par balayage).
func attendreVoie[T any](v *voiesDuBalayage, a *attendue[T]) T {
	<-a.pret
	if a.panique != nil && !v.relayee {
		v.relayee = true
		panic(a.panique)
	}
	return a.val
}

// fusionner verse chaque voie dans le parent, dans l ordre fixe, apres avoir attendu sa fin : seul le
// fil principal ecrit le parent, et seulement quand plus aucune voie ne l ecrit. Appelee une fois, a la
// fin du balayage ; sans voie, no-op.
func (v *voiesDuBalayage) fusionner() {
	if v == nil {
		return
	}
	for _, l := range v.lesVoies() {
		<-l.fini
		v.parent.Fusionner(l.fc)
	}
}

// joindre arrete les voies entre deux lectures puis les attend toutes : aucune goroutine ne survit au
// balayage, que ses valeurs aient ete consommees ou non. Une panique que personne n a encore rejouee l est
// ici. Sans voie : no-op.
func (v *voiesDuBalayage) joindre() {
	if v == nil {
		return
	}
	for _, l := range v.lesVoies() {
		l.arret.Store(true)
	}
	for _, l := range v.lesVoies() {
		<-l.fini
	}
	for _, l := range v.lesVoies() {
		if l.panique != nil && !v.relayee {
			v.relayee = true
			panic(l.panique)
		}
	}
}
