# NOTICE — origin of the data files of `libs/film`

This module is licensed AGPL-3.0 (see `README.md`). It contains two kinds of data files that are not source
code: five small tables embedded in the binary, and test fixtures made of film bytes. This page says where
each comes from. Nothing here is Microsoft, 343 Industries or Halo Studios software; the module is not
affiliated with them.

## Embedded tables (`go:embed`)

All five are identifiers, labels and points, not content of the game: no mesh, texture, sound or script.

| File | What it is | Origin |
|---|---|---|
| `damagetag/data/jpt_ids.txt` | 468 identifiers (32-bit, hexadecimal) of the `jpt!` damage-effect tags that a film can name as the source of a death | Listed on 2026-07-26 by the maintainer's own tool from the headers of the 126 `.module` archives of his Halo Infinite installation (fourCC group at offset `+0x14`, identifier at `+0x28`). The file keeps the identifiers only; the header says how it was produced |
| `damagetag/data/labels.tsv` | 468 lines, one per identifier: class (`ARME`, `MELEE`, `VEHICULE`, `GRENADE`, `OBJET_EXPLOSIF`, `DEGAT_GLOBAL`, `INCONNU`), status (`VALIDE`, `SOUS_RESERVE`, `AMBIGU`, `INCONNU`), name, detail | Precomputed by the maintainer from the same listing and from measurements on his own films. The names are the display names the game gives its weapons and vehicles; the classes and statuses are the maintainer's, each with the reason in the `detail` column. A line marked `AMBIGU` is never published by the decoder |
| `killicon/data/rules.tsv` | 58 rules that bridge a damage source to the name of a kill-feed icon sprite | Written by the maintainer; each rule carries its justification. The file names sprites, it does not contain them |
| `reponsesdejeu/data/reponses.tsv` | 4,804 lines: for each of the 11 groups of mode-script event files, every rank of its event tree (the response's name in the script, its kind, and for the 423 personal-score responses the points and English label; text ids; whether the rank is unsure) that a game-event message of a film can name | **Taken from libfilmshell** (Dend, OpenSpartan, MIT licence, `src/game/events_data.rs`, commit `ad17521`), with its credit header kept in the file and its licence text in `reponsesdejeu/data/LICENSE-libfilmshell`. Kept: every rank of the trees, with the name and the kind of each response and the points of those that give score (the medal and vehicle enumerations and the announcer text of the source are not kept). Transformed: one tab-separated line per rank, text ids as hexadecimal, the group named by its script files. The ranks were checked against the maintainer's own films and against the API's personal score (see the replay chronicle, v119) |
| `medalname/data/medals.tsv` | 157 lines, one per medal of the game: the sorting weight and the medal index that a highlight event of a film carries (bytes 47 and 59), and the medal's English name | **Taken from libfilmshell** (Dend, OpenSpartan, MIT licence, `src/game/medals_data.rs`, last commit on that file `291f85e`), with its credit header kept in the file and its licence text in `medalname/data/LICENSE-libfilmshell`. Kept: for each medal, the game's index (the discriminant of the source's `Medal` enum), the sorting weight and the English name (the kind, the difficulty and the personal score of the source are not kept). Transformed: one tab-separated line per medal, ordered by weight then index. Checked against the 124 pairs measured on the maintainer's own films (all 124 found, same weight, index and name) and against the `sortingWeight` of the official Waypoint medal catalogue (151 medals, all equal) |
| `damageeffects/data/effects.tsv` | 468 lines, one per damage effect of the game: its tag id and what it belongs to (a weapon, a melee strike, a grenade, the world...) | **Taken from libfilmshell** (Dend, OpenSpartan, MIT licence, `src/game/damage_data.rs`, commit `ad17521`), with its credit header kept in the file and its licence text in `damageeffects/data/LICENSE-libfilmshell`. Kept: the tag, the cause and, for the effects that deal damage, the amounts (bounds, shield and body multipliers, flags, radius). Transformed: one tab-separated line per effect, the cause in lowercase, the flags in hexadecimal; the weapon owners and projectile lists of the source are not kept. The set of weapon effects was checked against the API's hit counts (see the replay chronicle, v124) |

## Test fixtures

- **Film chunks** (`replay/testdata/minifilm_*`, `internal/facts/killsource/testdata/minibobine_*`,
  `internal/grammar/signaux/testdata/pied_bloc_*`, `internal/grammar/testdata/v41_chunk_he.bin`, `internal/grammar/testdata/vue_a_*.bin`): byte
  ranges of the Theater films of matches played on the maintainer's account, matches that are public in the
  game's match history. Each directory has a `PROVENANCE.txt` saying which bytes of which film it keeps and
  how to regenerate it. They are the only bytes of the game's own recordings in the module, and they are
  small cuts, not whole films.
- **Player identifiers are fictitious.** The Xbox identifiers (XUIDs) and the gamertags that a film carries
  were replaced, in the chunk bytes, in the decoded facts, in the golden files and in the tests, by invented
  ones. The rule is deterministic: the invented identifier of a real one is `0x0009000000000000` plus the
  first eight bytes of SHA-256 of a public seed (`levelup/identifiants-de-test-fictifs/v1`) followed by the
  real identifier on eight big-endian bytes, read big-endian and masked to 48 bits. The range is the one the
  decoder requires to locate a film's player table. `testdata/IDENTITES.tsv` lists every invented identifier
  of the module (and the seven made-up identifiers some tests use, which belong to nobody); a test
  refuses any identifier of that range that is not in the list. The real identifiers are not in the
  repository. The replacement keeps every length and every position, so the decoded values (indexes, times,
  counts) are unchanged.
- **Derived fixtures** (`*.golden`, `facts.json`, `inputs_*.bin.gz`, `equivalence/*`, the JSON fixtures):
  outputs of this decoder on the films above, regenerated by the tests that read them; they are data, not
  recordings.
- **Map bounds catalogue** (`internal/grammar/testdata/map_quant_bounds.json`): for each map, the minimum and
  maximum of the world bounds on the three axes and the axis widths, as numbers measured by the maintainer's
  own tool on the game's maps; no mesh, no geometry. It is a byte-for-byte copy of the catalogue the
  maintainer's application ships, kept equal by `TestLesBornesDeTestSontLeCatalogueLivre` (which runs when the
  module is tested inside that application's repository).
- **Mini reels of `testdata/corpus_objectifs`**: prefixes of two matches of the maintainer, kept byte for
  byte as taken; see that folder's `README.md`. Each chunk is cut before the first session-population packet
  (type 8), the packet that carries player identifiers in a replication chunk: they hold no XUID and no
  gamertag, and no test of this module reads them.
- **Notes on the game's binary** (`internal/grammar/testdata/ecs_table.tsv`, and the `Preuve` / source comments of the
  profile tables): component names, offsets and function addresses that the maintainer read from the game's
  executable with a disassembler, so that a decoder line can say where it comes from. They are
  observations, not code of the game.
- **Maps, weapons and medal names** that appear in fixtures and tables are the names the game itself shows
  for them.

## Prior art consulted

- **libfilmshell** (Dend, MIT licence, https://github.com/OpenSpartan/libfilmshell, tree published on 2026-11-09;
  the exact commit read was not recorded). Its reader showed that a position names its quantization region (an
  index, then three axes at the widths of THAT region), where this decoder applied the widths of the played
  region to every index. The reading was then checked against the game's executable (`FUN_14076e524`, the
  per-region table filled by `FUN_140be9a14`) and measured on the maintainer's own films. No line of libfilmshell
  code is copied or translated into this module; the region tables of the map catalogue are produced from the
  game's files by `cmd/mapquant-build`, not taken from libfilmshell's own tables.
- **libfilmshell again, for the match information of `chunk_00`** (same author, licence and tree). The paths of
  the game variant's Bond fields that this decoder reads (score to win, time limit, respawn time, overtime,
  starting loadout, options, string block), the two 16-byte blocks that hold the map variant's asset and version
  identifiers, and the start time were taken from its description of the format, then checked on the
  maintainer's own films: the decoder agrees with libfilmshell on every field of the 1,791 films that library
  reads, and differs on the name and description of one film, whose string table says otherwise. The Bond reader
  of this module is its own code (a reader by field paths over the bit stream, the strings found through the
  variant's offset table); no line of libfilmshell code is copied or translated.
- **libfilmshell, a third time, for the score responses of the game-event messages** (same author and licence, commit
  `ad17521`). Unlike the two entries above, a DATA file is taken here: `reponsesdejeu/data/reponses.tsv` reproduces,
  in this module's own format, the event trees of `src/game/events_data.rs` (names, kinds and personal-score points), because
  that table comes from the game's mode scripts, which this decoder does not read. The file keeps the credit header,
  the licence text sits next to it (`reponsesdejeu/data/LICENSE-libfilmshell`) and the README names libfilmshell
  in its credits. The reading of the messages themselves (the bit layout, the script group, the recipients) is this
  module's own code; no line of libfilmshell code is copied or translated.
- **libfilmshell, a fourth time, for the medals of the game** (same author and licence, last commit on the file
  `291f85e`). `medalname/data/medals.tsv` reproduces, in this module's own format, the medal list of
  `src/game/medals_data.rs` (index, sorting weight, English name), because that list comes from the game's files, which
  this decoder does not read. The file keeps the credit header, the licence text sits next to it
  (`medalname/data/LICENSE-libfilmshell`) and the README names libfilmshell in its credits. The reading of the highlight
  events is this module's own code; no line of libfilmshell code is copied or translated.
- **libfilmshell, a fifth time, for the end of a life** (same author and licence). The method of `src/owners.rs`,
  `src/motion.rs` and `src/lives.rs` is taken: a life's biped is created with the player who controls it, the player gives it up when he
  dies (a death) and the game deletes it still controlled when a round ends, so the two ends are read apart. The
  replay document publishes them as `tracks[].endReason` and `releaseFrame`, and does not judge which end is a death
  (libfilmshell counts a release as a death only where its scoreboard counts one, `died_at` of `src/lives.rs`). The reading of the `unit-control`
  component and of the DEL records, and the body-by-body bookkeeping of the assembly (`replay/tracks_fin_par_commande.go`),
  are this module's own code; no line of libfilmshell code is copied or translated.
- **libfilmshell, a sixth time, for the damage effects** (same author, licence and commit): `damageeffects/data/effects.tsv` reproduces, in this module's own format, the cause and the damage amounts of each damage effect of `src/game/damage_data.rs`, because that table comes from the game's damage tags, which this decoder does not read. Same credit header, same licence file; the README names libfilmshell in its credits. The **method** of `src/game/hit_damage.rs` is also taken: a hit of a fixed-amount effect is applied to the unit's real life (shield first, then body, with the tag's flags), the hits that the table cannot size are measured on what the life loses, and a headshot that finishes the unit takes what remains. The reading of the damage messages, of the two vitality components of the biped, and the code that applies the method (`damageeffects/effects.go`, `replay/degats_en_points.go`, `replay/vie_des_bipedes.go`) are this module's own code; no line of libfilmshell code is copied or translated.
- **libfilmshell, a seventh time, for the frozen round values of the scoreboard** (same author and licence, tree published
  on 2026-11-09). Its reader of the statborg components (`src/components.rs`, `StatFinalizedRounds`) gave the layout of
  components 28 to 55: a 32-bit mask of ended rounds, then two values per set bit, each behind a "default" flag; its
  snapshot reader (`src/gamestate.rs`) reads the 28 current-round rows only. The layout matches the game's reader
  (`FUN_142ed3c50`, already ported for the walk as `grammar.consumeStatborgFinalized`), and the order of the values, the
  meaning of the default flag, the rounds a record carries and the player the values belong to were measured on the
  maintainer's own films (see `FORMAT.md`, section 5). The reader of the record components (`internal/grammar/signaux/statborg_figees.go`)
  and the publication per player and round (`internal/facts/objectives/frozen_rounds.go`, `replay/score_timeline_figees.go`)
  are this module's own code; no line of libfilmshell code is copied or translated.

- **libfilmshell, an eighth time, for view A** (same author and licence, tree published on 2026-11-09). The widths of the payloads of genres 13 (`motor_system_interruption`),
  27 (`controlToggleResponse`) and 117 (`EquipmentTranslocatorTeleportEffects`) of view A (`src/events.rs`, the `match` on the event type) are taken, as
  is the attribution of genres 75 (`AIDialog`) and 76 (`Dialogue2D`) to the spoken line and the announcement; the widths were then proved on the maintainer's own
  films (a frame read with these widths ends where its declared end is). The reading, the chatter and announcement channel and the publication
  (`internal/grammar/vue_a_charges_effets.go`, `internal/grammar/repliques.go`, `replay/document_repliques.go`) are this module's own code; no line of libfilmshell code is copied or translated.
- **A contributor's 3D replay reader** (public repository `llampwall/installation-08`, a fork of this decoder, revision `a66b791`). The contributor found that the optional 5-bit
  field of a control entry carries the crouch button (bit of value 1) and the jump button (bit of value 4), proposed publishing the player's gestures
  read in view A (the melee strike and the started throw), and measured that the reloads written twice at the same instant are copies of the reload in progress. This module's
  readings (`internal/grammar/commandes_de_controle.go`, `internal/grammar/pas_des_commandes.go`, `internal/grammar/gestes.go`, `replay/document_stances_commandes.go`,
  `replay/document_gestes.go`) are written again under its own rules and measured again on the maintainer's films; no line of the fork is copied. The slide the fork deduces from the sprint and the
  speed is not taken. Four further additions are taken from the same work and written again: the film source by path (`filmcache/source_par_chemin.go`), the map catalogue read from bytes (`decfilm/catalogue_octets.go`), the three axes (`forward`, `up`, `scale`) of a weapon pad in the query layer (`replay/map_weapon_pads.go`), and the ammunition of each inventory delta kept in the facts file (`replay/filmfacts_munitions_delta.go`). The contributor also wrote, in his copy, the reading of the desired weapon set from the delta frames and the publication of the ammunition carried (`internal/grammar/canal_du_jeu_darmes.go`, `replay/document_arme_tiree.go`, `replay/ammo_reads.go`), and the ideas of publishing the pad candidates, of reading equipment pads and of tying the explosion of a throw to it (`replay/ground_weapon_pad_candidates.go`, `replay/powerup_pads.go`, `replay/grenades_detonation.go`); all are written again under this module's own rules, with their own coverage and a place in the facts file. The same copy also publishes the positions and the aim at full rate, and the height and the flight of the projectiles at the film's rate (`tracks[].fullRate`, `projectiles[].z` and `projectiles[].fullRate` here, in `replay/tracks_plein_debit.go` and `replay/projectiles_plein_debit.go`), written again with this module's shared rank index and its own counters, without the columns of the stick.

## Third-party code

The module imports `github.com/klauspost/compress`, `github.com/pelletier/go-toml/v2` and `golang.org/x/sys`
(see `go.mod`), and `libs/gamefiles` and `libs/himap` of the same repository. Through `libs/himap` its tests
reach the vendored GPL-3.0 `ooz` decoder of `libs/gamefiles`; see `libs/gamefiles/ooz/NOTICE.md`.
