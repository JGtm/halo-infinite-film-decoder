package main

// profil.go — PROFIL D'UNE PASSE : ou le temps et la memoire partent pendant UN film.
//
// `-profil-cpu <fichier>` et `-profil-tas <fichier>` (avec `-seulement <film>`, un seul film) font une
// passe du decodeur, comme `executer`, sous `runtime/pprof`. Aucune de ces deux options n'agit sur les
// mesures : sans elles, ce fichier ne s'execute pas. Le profil du tas est ecrit de deux facons : en fin
// de passe (`<fichier>`, avec `alloc_space` et `inuse_space`) et a l'instant du plus haut tas vu par un
// echantillonneur (`<fichier>.pic`), car la fin de passe ne retient que le document rendu.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"
)

const (
	// periodeEchantillonTas : pas de l'echantillonneur du plus haut tas.
	periodeEchantillonTas = 25 * time.Millisecond
	// tauxEchantillonTas : un echantillon d'allocation tous les 4 Kio (le defaut du runtime est 512 Kio).
	tauxEchantillonTas = 4096
)

// survolEtape, s'il est pose, recoit le nom de chaque etape observee de la construction. Nil hors
// profil : la mesure ne le voit pas.
var survolEtape func(etape string)

// profilDemande dit si une des deux options de profil est posee.
func (o options) profilDemande() bool { return o.profilCPU != "" || o.profilTas != "" }

// verifierProfil refuse un profil sans film unique, ou combine a l'usage interne du fils.
func (o options) verifierProfil() error {
	if !o.profilDemande() {
		return nil
	}
	if o.seulement == "" || strings.Contains(o.seulement, ",") {
		return errors.New("-profil-cpu et -profil-tas demandent -seulement avec un seul film")
	}
	if o.enfant {
		return errors.New("-profil-cpu et -profil-tas sont incompatibles avec -enfant")
	}
	return nil
}

// echantillonneurTas garde le profil du tas pris au plus haut tas vu.
type echantillonneurTas struct {
	mu    sync.Mutex
	max   uint64
	blob  []byte
	arret chan struct{}
	fini  chan struct{}
}

func demarrerEchantillonneur() *echantillonneurTas {
	e := &echantillonneurTas{arret: make(chan struct{}), fini: make(chan struct{})}
	go func() {
		defer close(e.fini)
		t := time.NewTicker(periodeEchantillonTas)
		defer t.Stop()
		var ms runtime.MemStats
		for {
			select {
			case <-e.arret:
				return
			case <-t.C:
				runtime.ReadMemStats(&ms)
				if ms.HeapInuse > e.max+e.max/8 {
					e.prendre(ms.HeapInuse)
				}
			}
		}
	}()
	return e
}

func (e *echantillonneurTas) prendre(inuse uint64) {
	var b bytesTampon
	if err := pprof.Lookup("heap").WriteTo(&b, 0); err != nil {
		return
	}
	e.mu.Lock()
	e.max, e.blob = inuse, b.b
	e.mu.Unlock()
}

type bytesTampon struct{ b []byte }

func (t *bytesTampon) Write(p []byte) (int, error) { t.b = append(t.b, p...); return len(p), nil }

func (e *echantillonneurTas) arreter() (uint64, []byte) {
	close(e.arret)
	<-e.fini
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.max, e.blob
}

// profiler fait une passe du film `opt.seulement` sous profil, et rend le code de sortie.
func profiler(ctx context.Context, opt options, stdout, stderr io.Writer) int {
	cible, err := cibleUnique(opt)
	if err != nil {
		dire(stderr, "film-bench : %v\n", err)
		return 1
	}
	if err := profilerPasse(ctx, opt, cible, stdout); err != nil {
		dire(stderr, "film-bench : %v\n", err)
		return 1
	}
	return 0
}

func profilerPasse(ctx context.Context, opt options, cible Cible, stdout io.Writer) (err error) {
	var echant *echantillonneurTas
	if opt.profilTas != "" {
		runtime.MemProfileRate = tauxEchantillonTas
		echant = demarrerEchantillonneur()
	}
	if opt.profilCPU != "" {
		f, errC := os.Create(opt.profilCPU) //nolint:gosec // chemin choisi par l'operateur
		if errC != nil {
			return fmt.Errorf("profil CPU : %w", errC)
		}
		defer func() {
			if errF := f.Close(); err == nil {
				err = errF
			}
		}()
		if errS := pprof.StartCPUProfile(f); errS != nil {
			return fmt.Errorf("profil CPU : %w", errS)
		}
	}
	debut := time.Now()
	survolEtape = func(etape string) { dire(stdout, "etape %-28s +%d ms\n", etape, time.Since(debut).Milliseconds()) }
	res, errP := executer(ctx, cible, ConfigPasse{Vitesse: opt.vitesse()})
	survolEtape = nil
	if opt.profilCPU != "" {
		pprof.StopCPUProfile()
	}
	if errP != nil {
		return errP
	}
	p := res.Passe
	dire(stdout, "decompression %s ms ; decodage %s ms ; publication %s ms ; total %s ms\n",
		ms(p.Decompression), ms(p.Decodage), ms(p.Publication), ms(p.Total))
	if echant == nil {
		return nil
	}
	maxInuse, pic := echant.arreter()
	return ecrireProfilsTas(opt.profilTas, pic, maxInuse, stdout)
}

// ecrireProfilsTas range le profil de fin de passe (apres un ramasse-miettes, le document rendu encore
// en memoire) et celui du plus haut tas.
func ecrireProfilsTas(chemin string, pic []byte, maxInuse uint64, stdout io.Writer) error {
	runtime.GC()
	f, err := os.Create(chemin) //nolint:gosec // chemin choisi par l'operateur
	if err != nil {
		return fmt.Errorf("profil du tas : %w", err)
	}
	if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
		_ = f.Close()
		return fmt.Errorf("profil du tas : %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	if len(pic) > 0 {
		if err := os.WriteFile(chemin+".pic", pic, 0o600); err != nil {
			return fmt.Errorf("profil du plus haut tas : %w", err)
		}
	}
	dire(stdout, "plus haut tas echantillonne : %d Mio\n", maxInuse/octetsParMio)
	return nil
}
