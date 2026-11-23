package main

import (
	"fmt"
	"io"
	"time"
)

// Entete est ce que le releve dit de lui-meme : quand, sur quoi, comment. Aucun chemin, aucun
// nom d'utilisateur, aucun nom de machine : le fichier se range au depot.
type Entete struct {
	Date       time.Time
	Arbre      string // `git describe` de l'arbre mesure, ou « inconnu »
	Machine    Machine
	Passes     int
	PlafondGiB int
}

// ResultatFilm est le releve d'un film : ses statistiques par phase, son pic et ses empreintes.
type ResultatFilm struct {
	Film           Film
	Stats          map[Phase]Stat
	PicMio         float64
	EmpreinteDoc   string
	EmpreinteFaits string
}

// colonnesDuReleve est l'en-tete de colonnes du TSV.
const colonnesDuReleve = "film\tmode\tversion\tjoueurs\tmorceaux\tfils\tphase\tmediane_ms\tmin_ms\tmax_ms\tpasses\tpic_mio\tempreinte_document\tempreinte_faits"

// ms rend une duree en millisecondes, a deux decimales.
func ms(d time.Duration) string { return fmt.Sprintf("%.2f", float64(d)/float64(time.Millisecond)) }

// ecrireReleve ecrit le TSV : un en-tete commente, la ligne des colonnes, puis une ligne par film
// et par phase (le pic et les empreintes du film se repetent sur chacune de ses lignes).
func ecrireReleve(w io.Writer, e Entete, resultats []ResultatFilm) error {
	lignes := []string{
		"# film-bench : releve A (le decodeur seul, options minimales)",
		"# date\t" + e.Date.UTC().Format(time.RFC3339),
		"# arbre\t" + e.Arbre,
		"# go\t" + e.Machine.Go,
		"# systeme\t" + e.Machine.Systeme,
		"# processeur\t" + e.Machine.Processeur,
		fmt.Sprintf("# coeurs\t%d", e.Machine.Coeurs),
		fmt.Sprintf("# memoire_mio\t%d", e.Machine.MemoireMio),
		fmt.Sprintf("# gomaxprocs\t%d", e.Machine.GoMaxProcs),
		fmt.Sprintf("# passes\t%d (apres une passe d'echauffement ecartee)", e.Passes),
		fmt.Sprintf("# plafond_pic_gio\t%d", e.PlafondGiB),
		"# unites : durees en millisecondes ; pic en Mio de l'empreinte memguard (memoire du tas Go) ; empreintes SHA-256",
		colonnesDuReleve,
	}
	for _, l := range lignes {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}
	for _, r := range resultats {
		for _, ph := range ordreDesPhases {
			s := r.Stats[ph]
			_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%d\t%.0f\t%s\t%s\n",
				r.Film.Nom, r.Film.Mode, r.Film.Version, r.Film.Joueurs, r.Film.Morceaux, filsDeCetteVersion,
				ph, ms(s.Mediane), ms(s.Min), ms(s.Max), s.Passes, r.PicMio, r.EmpreinteDoc, r.EmpreinteFaits)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ecrireResume ecrit le tableau lisible : les medianes par film et le pic.
func ecrireResume(w io.Writer, resultats []ResultatFilm) error {
	if _, err := fmt.Fprintf(w, "%-8s %12s %11s %10s %11s %10s %9s\n", "film", "decompress.", "chargement",
		"decodage", "publication", "total", "pic Mio"); err != nil {
		return err
	}
	for _, r := range resultats {
		_, err := fmt.Fprintf(w, "%-8s %12s %11s %10s %11s %10s %9.0f\n", r.Film.Nom,
			ms(r.Stats[PhaseDecompression].Mediane), ms(r.Stats[PhaseChargement].Mediane),
			ms(r.Stats[PhaseDecodage].Mediane), ms(r.Stats[PhasePublication].Mediane),
			ms(r.Stats[PhaseTotal].Mediane), r.PicMio)
		if err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, "medianes en ms ; la decompression est comprise dans le chargement ; total = chargement + decodage + publication")
	return err
}
