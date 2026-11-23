package replay

// film_scan_fond.go — LA TACHE DE FOND DU BALAYAGE : LES LECTURES QUI NE DEPENDENT QUE DU FILM.
//
// Quand le contexte permet plus d un fil ([grammar.FilmContext.Fils]), UNE goroutine lit, pendant la
// marche des trames, trois choses qui ne dependent que du film charge et du roster de l appelant :
//
//   - les lectures du film seul du pont d identite ([grammar.LireLesFaitsDuFilmSeul] : temps forts,
//     medailles, table d index, origine d horloge) ;
//   - les noms de script que le chunk 0 declare ([grammar.ScanScriptsDuFilm]) ;
//   - la table des joueurs du film aux listes de la session ([grammar.ScanFilmPlayerTableAuxListages]).
//
// # POURQUOI C EST SANS ETAT PARTAGE
//
// La tache ne recoit ni le contexte du film, ni le profil de balayage, ni le rapport des replis, ni
// l observateur, ni le journal : seulement le film (immuable apres son chargement, aucun accesseur
// ne le mute) et un roster. Les trois lectures n ecrivent aucune variable de paquet (le seul etat
// partage qu elles frolent est le `sync.Map` des empreintes inconnues, que ces lectures n appellent
// pas). Le roster passe par [rosterOf], qui ne lit que ses deux arguments et rend une tranche neuve :
// la tache en garde une COPIE du roster de l appelant.
//
// # CE QUI RESTE SUR LE FIL APPELANT, DANS L ORDRE D UN FIL
//
// Tout ce qui s observe ou se journalise : le versement des replis que les faits ont constates
// (fait par le pont), la consignation et le journal de la table des joueurs, les etapes observees, le
// compteur de replis. Chaque consommateur attend SA valeur ([fondDuFilm.lesFaits],
// [fondDuFilm.lesScripts], [fondDuFilm.laTable]) la ou il la lisait, et la prend telle que la lecture
// en serie l aurait rendue : la sortie et l ordre des etapes sont ceux d un fil.
//
// # SORTIE ANTICIPEE, PANIQUE
//
// Le balayage rend la main avant d avoir consomme les trois valeurs quand la marche ou les positions
// echouent : [fondDuFilm.joindre], differe par l appelant, attend alors la tache (aucune goroutine ne
// survit au balayage). Une panique de la tache est capturee et rejouee, telle quelle, sur le fil
// appelant au moment ou il attend la valeur que la tache n a pas rendue (ou a la jointure) ; elle
// n est rejouee qu une fois.
//
// CE FICHIER PORTE L UNE DES DEUX INSTRUCTIONS `go` DE `replay` (l autre : film_scan_voies.go ; garde-rail :
// `depottest.TestUneSeuleSourceDeGoroutines`). Il n est pas dans `fichiersDuBalayage` : ses lectures
// sont des COPIES de celles des consommateurs, et le garde de l ordre des etapes les compterait deux
// fois (cf. `TestLesLecturesDuFondSontCellesDesConsommateurs`).

import (
	"errors"
	"slices"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/source"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// tableLueDAvance est la table des joueurs du film aux listes de la session, avec l erreur de sa lecture.
type tableLueDAvance struct {
	lu  grammar.TableDuFilmAuxListages
	err error
}

// errFondInterrompu signale une tache de fond arretee sans panique (runtime.Goexit) avant d avoir
// rendu toutes ses valeurs.
var errFondInterrompu = errors.New("tache de fond du balayage interrompue avant la fin de ses lectures")

// attendue est une valeur que la tache rend. `pret` se ferme une fois, par la tache seule, apres
// l ecriture de `val` ou de `panique` : un lecteur qui a recu de `pret` voit l une ou l autre.
type attendue[T any] struct {
	pret    chan struct{}
	val     T
	panique any
	clos    bool // lu et ecrit par la seule tache
}

// clore rend la valeur a ses lecteurs ; `p` non nil est la panique qui a empeche de la produire.
func (a *attendue[T]) clore(p any) {
	if a.clos {
		return
	}
	a.clos = true
	a.panique = p
	close(a.pret)
}

// fondDuFilm est la tache de fond d un balayage. Un pointeur nil est l absence de tache (un fil) :
// tous ses accesseurs sont alors des no-op qui rendent « pas de valeur d avance ».
type fondDuFilm struct {
	faits   attendue[grammar.FaitsDuFilmSeul]
	scripts attendue[[]string]
	table   attendue[tableLueDAvance]
	// relayee dit qu une panique a deja ete rejouee : le fil appelant seul la lit et l ecrit.
	relayee bool
}

// demarrerLeFond lance la tache quand le contexte permet plus d un fil, et rend nil sinon : a un fil,
// aucune goroutine n est lancee et les consommateurs lisent en serie, comme avant.
func (s *filmScan) demarrerLeFond() *fondDuFilm {
	if s.fc.Fils() <= 1 {
		return nil
	}
	f := &fondDuFilm{
		faits:   attendue[grammar.FaitsDuFilmSeul]{pret: make(chan struct{})},
		scripts: attendue[[]string]{pret: make(chan struct{})},
		table:   attendue[tableLueDAvance]{pret: make(chan struct{})},
	}
	extra := slices.Clone(s.opt.RosterXUIDs)
	go f.lireLeFond(s.film, func(morts []types.Death) []uint64 { return rosterOf(morts, extra) })
	return f
}

// lireLeFond est le corps de la tache : les trois lectures, dans l ordre ou le balayage en a besoin (le pont,
// puis les scripts, puis la table). Chaque valeur est rendue des qu elle est prete.
func (f *fondDuFilm) lireLeFond(film *source.Film, roster func([]types.Death) []uint64) {
	defer f.terminer()
	f.faits.val = grammar.LireLesFaitsDuFilmSeul(film, roster)
	f.faits.clore(nil)
	f.scripts.val = grammar.ScanScriptsDuFilm(film)
	f.scripts.clore(nil)
	lu, err := grammar.ScanFilmPlayerTableAuxListages(film)
	f.table.val = tableLueDAvance{lu: lu, err: err}
	f.table.clore(nil)
}

// terminer ferme ce que la tache n a pas rendu, avec la panique qui l en a empechee. Differe par
// [fondDuFilm.lireLeFond] : il est la seule fonction qui rattrape la panique.
func (f *fondDuFilm) terminer() {
	p := recover()
	if p == nil && !f.table.clos {
		p = errFondInterrompu
	}
	f.faits.clore(p)
	f.scripts.clore(p)
	f.table.clore(p)
}

// attendre rend la valeur de `a` quand la tache l a produite, et rejoue sa panique sur le fil appelant
// (une seule fois par tache) quand elle ne l a pas produite.
func attendre[T any](f *fondDuFilm, a *attendue[T]) T {
	<-a.pret
	if a.panique != nil && !f.relayee {
		f.relayee = true
		panic(a.panique)
	}
	return a.val
}

// lesFaits rend les lectures du film seul du pont, ou nil sans tache (le pont les lit alors lui-meme).
func (f *fondDuFilm) lesFaits() *grammar.FaitsDuFilmSeul {
	if f == nil {
		return nil
	}
	v := attendre(f, &f.faits)
	return &v
}

// lesScripts rend les noms de script du film, et faux sans tache (l appelant les lit alors lui-meme).
func (f *fondDuFilm) lesScripts() ([]string, bool) {
	if f == nil {
		return nil, false
	}
	return attendre(f, &f.scripts), true
}

// laTable rend la table des joueurs du film lue d avance, ou nil sans tache.
func (f *fondDuFilm) laTable() *tableLueDAvance {
	if f == nil {
		return nil
	}
	v := attendre(f, &f.table)
	return &v
}

// joindre attend la fin de la tache : aucune goroutine ne survit au balayage, que ses valeurs aient
// ete consommees ou non. Une panique que personne n a encore rejouee l est ici. Sans tache : no-op.
func (f *fondDuFilm) joindre() {
	if f == nil {
		return
	}
	<-f.table.pret
	for _, p := range []any{f.faits.panique, f.scripts.panique, f.table.panique} {
		if p != nil && !f.relayee {
			f.relayee = true
			panic(p)
		}
	}
}
