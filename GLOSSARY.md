# Glossary

The Go sources of this module are written in French and use many coded names. This glossary maps each internal term to an
English equivalent, a definition, and the name that libfilmshell (OpenSpartan, MIT), a Rust decoder of the same format, gives
the same thing. "no equivalent" means libfilmshell has no distinct name for the concept; "approx." means the closest concept,
not an identical one. libfilmshell names are those of its source files (`container.rs`, `payload.rs`, `snapshot.rs`,
`registry.rs`, `events.rs`, `components.rs`, `levels.rs` and others). The last column gives a file of this repository where
the term is visible. [`FORMAT.md`](FORMAT.md) explains the format these terms describe.

## Film structure

| Term in the code | English | Definition | libfilmshell | Seen in |
|---|---|---|---|---|
| morceau, chunk | chunk | One of the numbered files `chunk_NN.bin` that make up a film. Chunk 0 is the header, the last one holds the highlights. | `Chunk`, `ChunkKind` | `internal/source/film.go` |
| bobine | reel | A film, or a partial set of its chunks, taken as a test object. A "partial reel" lacks chunk 0. | no equivalent | `internal/source/doc.go` |
| mini-bobine | minimal film fixture | A few real chunks of one match, with fictitious player ids, shipped with the module's tests. | no equivalent | `replay/testdata/minifilm_000d5950/` |
| chunk_00, registre | header chunk | The first chunk (manifest type 1): registry, versions, build identity, game variant, player table. "Le registre" can also mean the component registry alone. | header chunk; registry | `internal/grammar/film_identity.go` |
| temps forts, pied | highlights chunk, footer | The last chunk (manifest type 3): a timeline of kills, deaths, medals and mode events, written bit-packed. | HighlightEvents chunk | `internal/grammar/highlight_events.go` |
| paquet | packet | The unit of a data chunk: a 16-byte header (type, size, time) and a payload. | `Packet` (its `kind` is the type) | `internal/source/film.go` |
| CHUNK_END | chunk-end packet | The type-7 packet of size 0 that closes a data chunk. | no equivalent | `internal/source/doc.go` |
| paquet delta, trame | delta packet, frame | A type-0 packet: the replication data of one tick (events, entities, control). "Trame" is the packet taken as the unit of the walk. | tick packet, type-0 payload; `Frame` | `internal/grammar/marche_trames.go` |
| image-clé | keyframe | A type-2 packet: the entity table of the whole replication world, written at the head of each chunk so a player can seek to it. | seek snapshot; `ENTITY_TABLE` | `internal/grammar/keyframe_entrees.go` |
| bloc de type 1 | type-1 block, datum block | A type-1 packet written just before the keyframe: per slot flags, generation and component bitmap. | no equivalent (types 1 and 2 are skipped in playback) | `internal/grammar/type1_datums.go` |
| instantané de tête | head snapshot | What a chunk holds between its keyframe and its first delta packet, for example the bot packets. | seek snapshot (approx.) | `internal/grammar/bot_metadata.go` |
| roster (paquet de type 8) | roster packet | A type-8 packet: the match's players, each with the same record as the chunk-0 player table. | player structs of type-8 packets | `internal/grammar/roster_type8.go` |
| BOT_METADATA (type 12) | bot metadata packet | A type-12 packet: the match's bots, with their team. | bot records (kind 12) | `internal/grammar/bot_metadata.go` |
| table des joueurs | player table | The 32 player records (occupied or vacant) at the end of the chunk-0 body. | roster table of chunk 0 | `internal/grammar/player_table.go` |
| slot vacant | vacant slot | An all-zero record of the player table; its length is computed, not measured. | no equivalent | `internal/grammar/player_table.go` |
| variante de partie | game variant | The Bond message in the chunk-0 body that holds the rules of the mode. | game variant (`variant.rs`) | `internal/grammar/film_variante_de_partie.go` |
| variante de carte | map variant | The map's variant, named in chunk 0 by an asset id and a version id (two GUIDs). | map variant ids (`header.rs`) | `internal/grammar/film_infos_de_match.go` |
| infos de match | match information | What chunk 0 says of the recorded match: variant fields, map-variant ids, start time. Returned by `decfilm.ReadMatchInfo`. | `Header`, `Variant` | `internal/profile/infos_de_match.go` |
| Bond | Bond | Microsoft Bond CompactBinary v2, the serialisation of the game variant, written byte by byte into the bit stream. | Bond Compact Binary | `internal/grammar/film_variante_bond.go` |
| section d'identification | identity section | The part of chunk 0 after the registry: version, build and flavour strings, build id, changelist. Five old films lack it. | header (`build`, `branch`) | `internal/grammar/film_identity.go` |
| contrôle de corruption | corruption check | The one bit of chunk 0 that adds a guard bit and a 32-bit sentinel after each present component. | no equivalent | `internal/grammar/controle_corruption_du_film.go` |
| version majeure, version de format | film major version, format version | The two leading `u32` of chunk 0. The major version selects the highlights block layout; the format version selects the registry and version-table widths. | the leading `41` and `27`, read as a header count, not named | `internal/grammar/film_major_version.go`, `film_format_version.go` |
| table par type | version table | The 123 `u32` after the registry, one per message kind of view A. | event versions (`header.rs`) | `internal/grammar/film_identity.go` |

## Reading a delta packet

| Term in the code | English | Definition | libfilmshell | Seen in |
|---|---|---|---|---|
| bit de configuration | configuration bit | The first bit of a delta packet and of a keyframe; it selects the integer ranges of view A. It is 1 on the measured films. | flag bit (`[flag:1]`) | `internal/grammar/frame_records.go` |
| vue A (rang 0) | view A, event view | The first part of a delta packet: the tick's event messages, ended by a 0 bit. | EventView | `internal/grammar/vue_a_lecture.go` |
| vue B (rang 1) | view B, entity view | The second part: the entity records (NEW, DEL, DELTA), ended by record type 0. | EntityView | `internal/grammar/frame_infer.go` |
| vue C (rang 2) | view C, control view | The third part: per-player control messages, not entities. | ControlView; `control_view` | `internal/grammar/frame_vue_controle.go` |
| rang | rank | The number of a view (0, 1, 2) in the order the game reads them. | no equivalent | `internal/grammar/frame_vue_messages.go` |
| tête (de paquet) | packet head | The start of a delta packet: configuration bit, continuation bit and genre of the first view A message. | no equivalent | `internal/grammar/vue_a_lecture.go` |
| genre | genre, message kind | The 7-bit selector (0 to 122) of a view A message's kind. | event type (`type:7`), event kind | `internal/grammar/vue_a_genres.go` |
| kind | control kind | The 2-bit selector of a view C entry (0, 1, 2 carry a body, 3 does not). | `kind` in `control_view` | `internal/grammar/frame_vue_controle.go` |
| référence (de message) | message reference, handle | A view A variable-width integer naming an object: 8 to 13 bits of value plus 2 bits of generation. | `Handle` | `internal/grammar/varwidth.go` |
| catégorie (de plage) | range category | The number 0 to 8 that fixes the width of a message reference. | handle kind (`handle_kinds`) | `internal/grammar/varwidth.go` |
| classe de la vue A | view A class | The verdict on a film's genre table: equal to the game's, a prefix of it, or unreadable. | no equivalent | `internal/grammar/vue_a_versions.go` |
| genre présumé | presumed genre | A genre numbered beyond 107 in a prefix film, whose identity is assumed and not read. | no equivalent | `internal/grammar/vue_a_versions.go` |
| message de tick | tick message | The update of slot 123 (archetype 4) whose component 0 is an 8-bit counter, 35 bits at the default frame. | tick message (`TICK_MESSAGE`) | `internal/grammar/localisateur.go` |
| signature stricte | strict signature | The search for the tick message to find the start of view B when view A cannot decide. | `find_tick_message` | `internal/grammar/localisateur.go` |
| enregistrement, record | entity record | One entry of view B: NEW, DEL or DELTA. | entry (`EntityOp`) | `internal/grammar/frame_records.go` |
| NEW | create | A record that creates an entity: type on 6 bits, default state, gate, mask, components. | create (op 1) | `internal/grammar/frame_records.go` |
| DEL | delete | A record that deletes an entity: id, then 32 bits that are skipped. | delete (op 2) | `internal/grammar/frame_records.go` |
| DELTA | update | A record that updates an entity: id, baseline selector, mask, components; no type. | update (op 3) | `internal/grammar/frame_records.go` |
| sélecteur de baseline | baseline selector | A bit, followed by 7 bits when it is 1, between the id and the mask of a DELTA. It is 0 on the measured films. | baseline | `internal/grammar/frame_records.go` |
| masque épars, dense | sparse mask, dense mask | The presence mask of components: a 3-bit count then 6-bit indices (sparse), or a 64-bit bitmask (dense). | form 0 list, form 1 mask | `internal/grammar/ecrivain_invariants.go` |
| état par défaut | default state, creation state | The archetype's creation data, read by a dedicated deserialiser at the start of a NEW. | creation data | `internal/grammar/traverse.go` |
| état complet | full state | The reading of all components of an archetype in registry order, with no mask, in a keyframe entry. | state (`m`) | `internal/grammar/keyframe_fullstate_loop.go` |
| n1, n2 | creation and state size words | The two 32-bit words of a keyframe entry: sizes of the game's creation and state structures, not lengths in the stream. | `n`, `m` | `internal/grammar/keyframe_fullstate_loop.go` |
| terminateur | terminator | The end marker of a view: a 0 bit (views A and C) or record type 0, the bits `000` (view B). | op 0, `[0]` | `internal/grammar/marche_trames_rangs.go` |
| fermeture au bit, paquet fermé | closure to the bit, closed packet | A packet is closed when the reading ends on view C's terminator with 0 to 7 zero bits left. | the walk ends exactly; the control view ends within 8 bits | `internal/grammar/frame_vue_controle.go` |
| fermeture refusée | closure refused | The views were read to their terminators but the packet does not close: a width is wrong at an unknown place. | no equivalent | `internal/grammar/lecture/paquet.go` |
| queue opaque | opaque tail | The rest of a packet that the walk did not cross from a known position, for a typed cause. | `Stop` (approx.) | `internal/grammar/lecture/paquet.go` |
| rejet | rejection | The exit of view B at a DELTA whose id is not in the table; the game then reads no bit of the body. | `UnknownEntity` stop (approx.) | `internal/grammar/frame_infer.go` |
| arrêt typé | typed stop | A named cause for which a reading ends before its terminator (unported component, unread genre, overflow). | `Stop` | `internal/grammar/frame_vue_controle.go` |
| étendue | extent | An interval of a reading, in bits: chunk, packet, first bit, length. The unit of provenance and closure. | span (`FrameSpan`, approx.) | `internal/grammar/lecture/etendue.go` |
| marche | walk | A reading pass over the packets of a film in stream order. "La marche de production" is the only walk of delta packets. | walk (`Walker`) | `internal/grammar/marche_trames.go` |
| porté, non porté | ported, not ported | A component or message reader is ported when the game's function has been transcribed; a present component that is not ported stops the traversal. | known and unknown components (`Stop::Component`) | `internal/grammar/testdata/ecs_table.README.md` |
| désynchronisation, desync | desynchronisation | A record's reading stops on an unknown or wrong width. | `Stop` | `internal/grammar/traverse.go` |

## Entities

| Term in the code | English | Definition | libfilmshell | Seen in |
|---|---|---|---|---|
| ti | type index, archetype index | The archetype number, which is the registry block number (0 to 49). | entity type (registry block) | `internal/grammar/registry.go` |
| archétype | archetype | An entity type: the ordered list of components a record can carry. | entity type | `internal/grammar/registry.go` |
| composant | component | A named block of data of an archetype, read by its own deserialiser. | component | `internal/grammar/registry.go` |
| iNN | component index NN | The index of a component within its archetype (`ti=35 i0`); the bit of the presence mask. | component index | `internal/grammar/testdata/ecs_table.README.md` |
| registre | component registry | The table of component names per archetype (blocks of 64 entries of 260 bytes) at the start of chunk 0. | registry | `internal/grammar/registry.go` |
| niveau (du registre) | registry level, precision level | The `u32` after a registry entry's name, passed to the component's reader. | row version | `internal/grammar/registry.go` |
| emplacement, slot | slot | The index of an entity in the entity table; the low 30 bits of the entity id. | index | `internal/grammar/frame_records.go` |
| génération | generation | The two high bits of the entity id; the game increments it when a slot is reused. | generation | `internal/grammar/frame_records.go` |
| eid | entity id | The 32-bit id of an entity: generation in bits 30 and 31, slot below. | `[id:32]` | `internal/grammar/frame_records.go` |
| poignée | handle (of a keyframe entry) | The third 32-bit field of a keyframe entry header: all ones when there is none. | handle (third header field) | `internal/grammar/keyframe_datums.go` |
| datum | datum | The game's name for one entry of its entity table (200 bytes: full id and archetype). | no equivalent | `internal/grammar/keyframe_datums.go` |
| table de datums, table d'entités | datum table, entity table | Two senses: the slot -> (eid, archetype) table that the game consults and a keyframe reproduces; and the type-1 block before it. | entity table (`ENTITY_TABLE`) | `internal/grammar/keyframe_datums.go`, `type1_datums.go` |
| monde (World) | world | The state a reader keeps during a walk: slot -> (eid, archetype, provenance of the binding). | `Walker` type map | `internal/grammar/world.go` |
| liaison | binding | The association of a slot with an archetype, with its provenance (NEW read, keyframe chain, header search, anticipation, inference). | `Walker` type map | `internal/grammar/lecture/record.go` |
| liaison par anticipation | binding by anticipation | A fallback: the archetype of an unknown slot is taken from a later chunk's keyframe. | next-snapshot tie-break (`seed_ahead`) | `internal/grammar/keyframe_anticipe.go` |
| NEW de tête | leading NEW records | NEW records at the head of an event list, found by chaining to locate the start of view B. | no equivalent | `internal/grammar/debut_de_liste.go` |
| bande de slots bipèdes | biped slot band | The range of slots that keyframes give to bipeds (players' bodies). | entity index ranges 512-767 (approx.) | `internal/grammar/offline_biped_band.go` |
| statborg | statborg | The game entity (archetype 6) that holds a player's or team's score counters. This decoder has no parser for it and finds its records by anchoring. | statborg component names | `internal/grammar/signaux/statborg_film.go` |
| xuid | XUID | The 64-bit Xbox user id of a player. | xuid | `internal/grammar/player_table.go` |
| vie (LifeKey) | life | One existence of a player's body, identified by (slot, generation). | life | `types/grammar_equipement.go` |
| pont d'identité | identity bridge | The single stage of six readings (teleports, positions, biped creations, death feed, index table, clock) that links bodies and slots to players. | roster and holders (approx.) | `internal/grammar/pont_identite.go` |
| mort d'objet, dead-state | object death, dead-state | The `object-dead-state` component (`i11`) of an entity, written when it dies. A vehicle's carries the killer's film index (`EnumB`) and the damage effect (`SrcTag0`); a biped's also carries the victim's film index (`EnumA`). | no equivalent | `internal/grammar/object_deaths.go` |
| état de parenté, occupation | parent state, occupancy read | The `object-parent-state` component (`i10`) of a biped: a transition, attached to a parent entity (a vehicle or one of its mounted parts) or free. | no equivalent | `internal/grammar/vehicle_occupancy_march.go` |
| occupant tué avec le véhicule | occupant killed with the vehicle | A biped whose dead-state is written in the vehicle's death packet (or in a later packet that also writes its free read) while its last parent state before it was attached to the vehicle or its parts. Read, with no time window and no distance. | no equivalent | `replay/vehicle_destruction_killed.go` |

## Positions

| Term in the code | English | Definition | libfilmshell | Seen in |
|---|---|---|---|---|
| position quantifiée | quantised position | A coordinate written as a quantum index on w bits per axis; the world value is computed from the bounds. | quantised position | `internal/grammar/lecteur_position.go` |
| région (BSP) | compression region | A zone of a map, one per structure BSP of its level, with its world bounding box. | box, `LevelBox` | `internal/profile/map_bounds.go` |
| index de région | region index | The number, read on indexW bits, of the row of the region table whose widths serve the three axes. | box index | `internal/grammar/lecteur_position.go` |
| niveau de précision (d'un site) | position precision level | An immediate of a position reader's call site (16, 15 or 12) that selects the table row; never read from the stream. | level (approx.) | `internal/profile/loi_largeurs.go` |
| loi des largeurs | width law | The game's formula giving the width of each axis from the bounds and the level. | no equivalent (widths from a level catalogue) | `internal/profile/loi_largeurs.go` |
| catalogue de carte | map-bounds catalogue | A JSON file supplied by the caller: bounds and widths per region of each map. Without an entry the map is refused. | level catalogue (`levels.rs`) | `internal/profile/map_bounds.go` |
| garde de pleine précision | full-precision guard | The game's condition under which a position is read as 96 raw bits (three floats). | no equivalent | `internal/grammar/lecteur_position.go` |
| i0 | component 0 position | Component 0 of an object archetype, the absolute position, and the split of its three axis widths. | position component | `internal/grammar/i0_layout.go` |
| MPP | multiplayer-properties block | The `object-multiplayer-properties` block of an object, two of whose widths depend on the format version. | multiplayer properties | `internal/profile/mpp_widths.go` |

## Profile, builds and provenance

| Term in the code | English | Definition | libfilmshell | Seen in |
|---|---|---|---|---|
| profil | profile | The immutable table of what is not in the bit stream (widths, bounds), keyed by what the film writes; each line carries its proof. | no equivalent (constants and `Params`) | `internal/profile/profile_table.go` |
| build | build | The string `HI_1_13_0`, `HI_1_12_0`... read in clear in chunk 0. A build absent from the table gives `ErrUnknownBuild`. | branch string | `internal/profile/build_profile.go` |
| relue, mesurée, présumée | read, measured, presumed | The three provenances of a profile value: read from the executable, measured on a witness film by an oracle, or neither. | no equivalent | `internal/profile/profile_table.go` |
| oracle | oracle | A measurement, internal or external to the film, that confirms a reading without deciding it. A proof, never a production path. | no equivalent | `internal/profile/build_profile.go` |
| image statique | static image | The values of the game binary's globals before it runs, as opposed to those it rewrites at load. | no equivalent | `internal/grammar/varwidth.go` |
| écrivain, lecteur (du jeu) | the game's writer, reader | The serialiser that produces the film and the function that reads it back. The decoder ports the reader and relies on the writer for invariants. | the game's writers and readers | `internal/grammar/ecrivain_invariants.go` |
| FUN_, DAT_ | Ghidra auto-names | Names of functions and globals of `HaloInfinite.exe` (image base `0x140000000`). | game function names | `internal/grammar/lecteur_position.go` |
| témoin | witness | A film chosen to check a change; also the reference decoder a reading is compared with. | no equivalent | `replay/testdata/` |

## Recovery, fallbacks and measurement

| Term in the code | English | Definition | libfilmshell | Seen in |
|---|---|---|---|---|
| repli | fallback | A named, ordered, counted and dated heuristic, tried after the reading and removed when its count is zero. | no equivalent (`skipped` stops are nearest) | `internal/facts/fallback/repli.go` |
| registre des replis | fallback registry | The single list of fallbacks (name, decided fact, typed condition, date, retirement criterion). | no equivalent | `internal/facts/fallback/registre.go` |
| récupération | recovery | A search that replaces a missing reading and marks its records as recovered; named and counted. | scan-based reading (`replication.rs`) | `internal/grammar/recuperations.go` |
| balayage | scan | A reading pass over a whole film, or a search for a pattern at every bit offset. | scan | `internal/grammar/highlight_events.go` |
| ancre | anchor | The position of a keyframe record header validated by its guards, which bounds the next record. | header found by scan | `internal/grammar/keyframe_world.go` |
| ancre élue | elected anchor | An anchor chosen by the election fallback when the chain of bounds breaks. | no equivalent | `internal/grammar/keyframe_world.go` |
| ancrage | anchoring | Locating biped records in delta packets by their header pattern (a recovery), without reading how they chain. | scanning for object updates | `internal/grammar/ancres_bipedes.go` |
| preuve (d'un record) | proof of a record | A keyframe record is proved when its traversal ends exactly on the next anchor. | no equivalent | `internal/grammar/keyframe_closure.go` |
| faits | facts | What the grammar says of a match (kill sources, objectives, scores, identities), stored with the revision of the layer that produced it. | model (`Film` data) | `internal/facts/` |
| cuisson | build step ("cook") | Turning a film into facts and then a replay document. | `Film::from_chunks` (approx.) | `replay/build_from_film.go` |
| recuisson | rebuild | A cook run again on an existing artifact after a layer's revision changed. | no equivalent | `internal/grammar/rev.go` |
| révision de couche | layer revision | A constant that fingerprints a layer's production sources; it changes when behaviour changes. | no equivalent | `internal/grammar/rev.go`, `revision/` |
| taux de fermeture | closure rate | The share of delta packets closed to the bit: a measure of width correctness, not of fact coverage. | no equivalent | `internal/grammar/frame_closure.go` |
| carte de fermeture | closure map | A per-build or per-film report of the closure rate and of the stop cause of each unclosed packet. | no equivalent | `internal/grammar/frame_closure_detail.go` |
| distribution, canal | distribution, channel | The walk hands what it reads to channels (movement states, continuous fire, deaths) that subscribe to its hooks instead of re-reading the film. | no equivalent | `internal/grammar/distribuer.go` |
| crochet, observation | hook, observation | A function the reader calls during the walk to publish a value; interpretation stays in the hooks, the structure carries none. | no equivalent | `internal/grammar/observateur.go` |
| bot | bot | A player controlled by the game; named by type-12 packets and counted like players. | bot | `internal/grammar/bot_metadata.go` |
| BTB | Big Team Battle | A match with large teams, which can have up to 63 chunks. | BTB | `internal/source/source.go` |
