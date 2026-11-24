package replay

import "github.com/JGtm/halo-infinite-film-decoder/types"

// filmfacts_projectiles_colles.go : LES PROJECTILES QUI SE COLLENT DANS LE COMPLEMENT DE LA SECTION 1 : les deux
// drapeaux et les comptes du canal, puis chaque lecture (horodatage en delta, projectile, parent, noeud, les trois
// valeurs de la pose, un octet de drapeaux, puis ce que les drapeaux annoncent : l ecart a la creation, le tag, le
// createur). Tout est ecrit brut. Une liste vide se relit nil.

// coutMinimalProjectileColle : horodatage, projectile (slot, generation), parent (slot, generation), noeud, trois
// valeurs de pose et les drapeaux, un octet chacun au moins.
const coutMinimalProjectileColle = 10

// Les drapeaux d une lecture.
const (
	drapeauCreation = 1 << iota
	drapeauNaissance
	drapeauTag
	drapeauCreateur
)

func encodeProjectilesColles(w *gwriter, rs []types.ProjectileAttachRead, st types.ProjectileAttachStats) {
	w.bool8(st.Scanned)
	w.bool8(st.Absent)
	for _, v := range []int{st.Frames, st.UnreadFrames, st.Records, st.Read, st.Created, st.Desynced, st.Repeats,
		st.NoBirth, st.NoTag, st.NoCreator, st.WideLocations} {
		w.u(uint64(v)) //nolint:gosec // compteurs positifs
	}
	w.u(uint64(len(rs)))
	var dernier uint64
	for _, m := range rs {
		w.u(m.TimestampUS - dernier)
		dernier = m.TimestampUS
		for _, v := range [...]uint32{m.Life.Slot, m.Life.Gen, m.Parent.Slot, m.Parent.Gen, m.Node,
			m.Pose[0], m.Pose[1], m.Pose[2]} {
			w.u(uint64(v))
		}
		w.u(drapeauxDeLaLecture(m))
		if m.HasBirth {
			w.u(m.TimestampUS - m.BornUS)
		}
		if m.HasTag {
			w.u(uint64(m.Tag))
		}
		if m.HasCreator {
			w.u(uint64(m.Creator.Slot))
			w.u(uint64(m.Creator.Gen))
		}
	}
}

// drapeauxDeLaLecture rend l octet de drapeaux de la lecture `m`.
func drapeauxDeLaLecture(m types.ProjectileAttachRead) uint64 {
	var d uint64
	for _, c := range [...]struct {
		drapeau uint64
		vrai    bool
	}{{drapeauCreation, m.Created}, {drapeauNaissance, m.HasBirth}, {drapeauTag, m.HasTag}, {drapeauCreateur, m.HasCreator}} {
		if c.vrai {
			d |= c.drapeau
		}
	}
	return d
}

func decodeProjectilesColles(r *greader) ([]types.ProjectileAttachRead, types.ProjectileAttachStats) {
	var st types.ProjectileAttachStats
	st.Scanned, st.Absent = r.bool8(), r.bool8()
	for _, v := range []*int{&st.Frames, &st.UnreadFrames, &st.Records, &st.Read, &st.Created, &st.Desynced,
		&st.Repeats, &st.NoBirth, &st.NoTag, &st.NoCreator, &st.WideLocations} {
		*v = int(r.u()) //nolint:gosec // compteurs ecrits positifs
	}
	n := r.compte(coutMinimalProjectileColle)
	var out []types.ProjectileAttachRead
	var dernier uint64
	for k := 0; k < n && r.err == nil; k++ {
		dernier += r.u()
		var v [8]uint32
		for i := range v {
			v[i] = uint32(r.u()) //nolint:gosec // ecrit depuis un uint32
		}
		m := types.ProjectileAttachRead{TimestampUS: dernier, Life: types.LifeKey{Slot: v[0], Gen: v[1]},
			Parent: types.LifeKey{Slot: v[2], Gen: v[3]}, Node: v[4], Pose: [3]uint32{v[5], v[6], v[7]}}
		d := r.u()
		m.Created, m.HasBirth, m.HasTag, m.HasCreator = d&drapeauCreation != 0, d&drapeauNaissance != 0, d&drapeauTag != 0,
			d&drapeauCreateur != 0
		if m.HasBirth {
			m.BornUS = dernier - r.u()
		}
		if m.HasTag {
			m.Tag = uint32(r.u()) //nolint:gosec // ecrit depuis un uint32
		}
		if m.HasCreator {
			m.Creator = types.LifeKey{Slot: uint32(r.u()), Gen: uint32(r.u())} //nolint:gosec // ecrits depuis des uint32
		}
		out = append(out, m)
	}
	return out, st
}
