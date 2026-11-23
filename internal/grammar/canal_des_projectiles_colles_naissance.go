package grammar

// canal_des_projectiles_colles_naissance.go : CE QUE LA CREATION D UN PROJECTILE DIT DE LUI, relu pour le canal des projectiles colles.
//
// Le record de creation d un projectile (ti=41) porte, dans son etat par defaut ([consumeDefaultStateTI41]) :
//   - le mot de 32 bits inconditionnel du bloc de proprietes multijoueur ([MPPWord32]) : l identifiant du tag `proj` de l objet, son TYPE (le meme
//     que celui des quatre grenades, avant le decalage d un bit qui en fait un identifiant global, cf. [GrenadeTypeIDsByRank]) ;
//   - derriere le drapeau 2, la reference de CATEGORIE 1 : l unite qui a cree le projectile. Meme espace de handle que le parent de i10
//     ([parentHandleBase]).
//
// LE MOT ET LA REFERENCE SE RELISENT, ils ne se capturent pas a la marche : la marche lit cet etat sans observateur, et poser un crochet sur son
// observation commune ferait dependre ces deux valeurs de ceux qui la partagent. Le canal reprend donc le record a son en-tete (`HeaderBit` +
// [woNewHeaderBits]) avec un lecteur a lui, charge du profil du film, et deroule le MEME deserialiseur : aucun second lecteur de l etat
// ([lecteurDIdentite] fait de meme pour les objets ti=37).
//
// UNE SEULE REFERENCE DE CATEGORIE 1 PEUT SE LIRE DANS CET ETAT : un compte different (zero quand la porte est fermee) ne donne aucun createur,
// jamais le premier d une liste.
//
// UN FILM SANS DESERIALISEUR D ETAT DE CREATION DU PROJECTILE ([ProfilDeBalayage] sans `DeserEtatParArchetype`) ne livre ni type ni createur : le
// deserialiseur ne s y deroule pas, et un relecteur qui s y deroulerait lirait des bits qui ne sont pas les siens.

import (
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
	"github.com/JGtm/halo-infinite-film-decoder/types"
)

// naissanceDeProjectile est ce que la creation de la vie courante d une cle a donne.
type naissanceDeProjectile struct {
	us        uint64
	tag       uint32
	aTag      bool
	createur  types.LifeKey
	aCreateur bool
}

// lecteurDeNaissance relit l etat de creation d un projectile : l observation qui retient le mot et les references, et le contexte de lecture.
type lecteurDeNaissance struct {
	ctx     ContexteDeLecture
	mot     uint64
	present bool
	refs    []UnitRefRead
}

// nouveauLecteurDeNaissance prepare la relecture pour le film `fc`, ou rend nil quand son profil ne deroule pas l etat de creation du projectile.
func nouveauLecteurDeNaissance(fc *FilmContext) *lecteurDeNaissance {
	if fc == nil {
		return nil
	}
	profil := fc.ProfilDeBalayage()
	if !profil.Grammaire.DeserEtatParArchetype {
		return nil
	}
	l := &lecteurDeNaissance{}
	obs := NouvelleObservation()
	obs.MppHook = func(f MPPField, v uint64, present bool) {
		if f == MPPWord32 {
			l.mot, l.present = v, present
		}
	}
	obs.UnitRefHook = func(u UnitRefRead) {
		if u.Kind == UnitRefVarWidth && u.Probe && u.Present {
			l.refs = append(l.refs, u)
		}
	}
	l.ctx = ContexteDeLecture{Profil: profil, Obs: obs}
	return l
}

// lire rend la naissance du projectile que le record NEW `r` du paquet `p` cree, avec le type et le createur quand ils se lisent.
func (l *lecteurDeNaissance) lire(p *lecture.Paquet, r *FrameRecord) naissanceDeProjectile {
	n := naissanceDeProjectile{us: p.TS}
	if l == nil {
		return n
	}
	l.mot, l.present, l.refs = 0, false, l.refs[:0]
	br := LecteurSur(p.Payload)
	br.PoserContexte(l.ctx)
	br.SetBitPos(r.HeaderBit + woNewHeaderBits)
	consumeDefaultStateTI41(br)
	if l.present {
		n.tag, n.aTag = uint32(l.mot), true //nolint:gosec // mot de 32 bits (R(32))
	}
	if len(l.refs) == 1 {
		u := l.refs[0]
		n.createur = types.LifeKey{Slot: (u.Val & parentHandleValueMask) + parentHandleBase, Gen: u.Tail}
		n.aCreateur = true
	}
	return n
}
