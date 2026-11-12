# The Halo Infinite Theater film format, and how this decoder reads it

A Theater film is the recording the game keeps of a match: a set of numbered files (chunks) that the Halo web services serve
for each match. This document describes how a film is built, level by level, and says for each level what this decoder
**reads** from the structure the film writes, what it **recovers** by a named search (a fallback, or a counted recovery), and
what it does **not read**. It is written for a developer who has not opened this repository. The Go sources are in French;
[`GLOSSARY.md`](GLOSSARY.md) maps their vocabulary to English and to that of libfilmshell (OpenSpartan, MIT), a Rust decoder of
the same format, which this document compares with where the two differ.

The format is not documented by the game's authors; it was recovered from the executable and by measuring films. Each section
ends with the files its statements come from.

## Conventions

- Bits are read most-significant first. `R(n)` reads n bits as an unsigned number. A read past the end of a buffer returns zero
  bits; the game's reader also rejects a packet that consumes more bits than it holds, and the decoder keeps both facts.
- Container headers are little-endian. A packet payload is a bit stream with no alignment.
- `FUN_xxxxxxxx` and `DAT_xxxxxxxx` are Ghidra auto-names of functions and globals of `HaloInfinite.exe` (image base
  `0x140000000`). The executable is not distributed here; the names let a reader with a copy find where a width comes from.
- In a *Sources* line, a bare file name is in the directory named just before it; libfilmshell files are in its `src/`.
- A width in the decoder's profile has one of three provenances: **read** (the executable's function is cited), **measured**
  (the executable of that build is not available; the value was measured on witness films against an oracle internal to the
  film) or **presumed** (neither; a test lists them).
- A delta packet is **closed** when the reading ends on the terminator of its last view with 0 to 7 zero bits left (section 4);
  closure is the decoder's measure of whether widths are right.

## Overview

```
film
 |-- chunk 0          manifest type 1   header: component registry, versions, build identity, game variant, player table
 |-- chunks 1 .. n-1  manifest type 2   replay chunks: a stream of packets (types 1, 2, 0, 8, 12, ... and 7 at the end)
 `-- chunk n          manifest type 3   highlights: kills, deaths and medals of the Theater timeline (the last chunk)
```

The header says how to read the replay chunks, which carry the match tick by tick as bit-packed replication data. The
highlights chunk is an independent timeline.

## 1. The container

The services give a manifest with each chunk's `{index, chunk type, start time in ms}`. This module fetches nothing: the caller
supplies the chunk bytes and, when it has it, the manifest. Only types 1, 2 and 3 occur in the manifests the authors
measured; a chunk present but absent from the manifest gets type 0, a convention of the decoder and not a value of the format.
A Big Team Battle film has up to 63 chunks, so its highlights chunk is `chunk_62` and the loader has no upper bound.

Chunks are indexed by the number in the file name, not by rank in the set supplied: a partial film that starts at `chunk_01` has
a data chunk at rank 0, which must not be taken for the header. At a numbering gap the decoder reads up to the gap and counts the
chunks it abandons.

**Compression.** The services deliver each chunk as zlib (first byte `0x78`). The decoder also reads a chunk stored raw or with
zstd (magic `28 b5 2f fd`), as its cache may keep it; the form is told from the first bytes, never from the file name. Inflated
chunks are kept for the life of a loaded film (a compressed chunk can weigh 89 MiB) and a packet payload is a slice of them.

**Packets.** An inflated replay chunk is a sequence of packets, a 16-byte header then the payload:

```
 | u16 type | 2 bytes | u32 size | u64 time (us) | payload ... |       little-endian header; the last packet of a data
                                                                        chunk is type 7 (CHUNK_END), size 0
```

The two bytes after the type are skipped; what they hold is not established. The packet walk, in order: (1) stop if
`offset + 16 + size` exceeds the chunk; (2) emit the packet, size 0 included; (3) stop after a type-7 packet; (4) stop, without
emitting, on a size-0 packet that is not type 7. On the measured films a size-0 packet in a data chunk is always CHUNK_END, last
in its chunk; degenerate size-0 headers occur only in chunk 0, which is never walked as packets. libfilmshell is stricter: it
accepts a chunk only if its walk ends exactly on the last byte, and otherwise splits the whole film on the marker `A0 7B 42`.

**Status.** Read: packet headers, the walk, chunk indexing, the three storage forms. Recovered: nothing. Not read: the two
header bytes; the payload of any type other than 0, 1, 2, 7, 8, 12. libfilmshell calls type 6 chunk bookkeeping and type 10 a
per-tick companion of type 0; no file here says what they hold (unverified).

*Sources: `internal/source/doc.go`, `film.go`, `zstd.go`, `source.go`; `internal/grammar/signaux/statborg_film.go`; libfilmshell
`container.rs`, `chunk.rs`.*

## 2. Chunk 0: the header

Chunk 0 is the game's film structure serialised field by field; it is not a packet stream. The offsets below are those of the
in-memory structure on `HI_1_12_0` and `HI_1_13_0`; the stream carries the fields contiguously, and everything after `0x0CB45C`
is one bit off byte alignment. Earlier builds have a shorter registry and version table, so every later offset moves: the decoder
keeps no absolute offset and derives positions from the end of the registry and from the build string.

```
offset    width                content
0x000000  32 bits              u32 film major version  (41 on HI_1_13_0, 40 on HI_1_12_0)
0x000004  32 bits              u32 format version      (27 on HI_1_12_0 and HI_1_13_0)
0x000008  0x659000 bits        component registry: 50 blocks of 0x4100 bytes (49 blocks before HI_1_12_0)
0x0CB208  0xF60 bits           version table: 123 u32, one per message kind of the event view (fewer on older builds)
0x0CB3F4  32 bytes             version string, e.g. "6.10026.19225.0"
0x0CB414  32 bytes             build string, e.g. "HI_1_13_0"                  <- the profile's build key
0x0CB434  32 bytes             flavour string, "release"
0x0CB454  32 bits              build id           0x0CB458  32 bits   changelist
0x0CB45C  1 bit                corruption-check flag; shifts everything after it by one bit
0x0CB460  2 x 256 bytes        two name fields (measured empty)
0x0CB660  32 bits              match start time (Unix seconds, the recorder's clock)
0x0CB664  3 x 32 bits          measured zero
0x0CB670  3 x 4096 bytes       three blocks, not interpreted
0x0CE670  2 x 16 bytes         map variant: asset id and version id (two GUIDs)
0x0CE690  ...                  body: game options, game variant, then 32 player records
```

**Versions and the profile.** The two leading `u32` are not interchangeable. The **format version** chooses the width of the
registry and of the version table (the game keeps one lookup table for each, keyed by it); the **major version** chooses the
layout of an event block of the highlights chunk (section 8); the **build** string chooses the width of structures the format
version does not separate. The decoder's *profile* is a table keyed by these three values, each line carrying its provenance. A
key absent from the table is never read as the nearest one: the decoder returns `ErrUnknownBuild` or `ErrUnknownFormat`, counts
it, and sets the film aside. Measured on 1,351 header chunks, (major, format) occurs as (41, 27) x 1123 `HI_1_13_0`; (40, 27) x
146 `HI_1_12_0`; (40, 25) x 39 `HI_1_11_0`; (39, 24) x 26 `HI_1_10_0`; (37, 24) x 10 `HI_1_8_0`; (38, 24) x 1 `HI_1_9_0`;
(33, 21) x 1 `HI_1_4_1`; and five films of format 20, which have no identity section. Major 40 occurs on two builds. Two rows of
the game's own format table (formats 21 and 25) disagree with the measurement, and the decoder follows the measurement.

**Registry.** A table of component names per entity type. An entity type is an *archetype*, identified by its block number `ti`
(0 to 49). A block is 64 entries of 260 bytes: a NUL-padded ASCII component name (`0x100` bytes) then a little-endian `u32`.
The named entries come first; an entry's position in its block is the *component index* `i`, the bit that an entity record's
presence mask addresses. The `HI_1_13_0` registry names 1,067 entries. It is identical between films of one build, so an FNV-1a
hash of the named entries identifies a build's component grammar; the hash is shared by `HI_1_8_0`/`HI_1_9_0`,
`HI_1_12_0`/`HI_1_13_0`, and `HI_1_4_1` with the major-33 films that have no identity. The `u32` after a name is passed by the game
to the component's reader: this decoder calls it the precision level, libfilmshell the row version, and whether it is a level, a
version or a mixture is not established. It is not the source of an absolute biped position's width (section 6).

**Identity.** The build string is found by searching for the prefix `HI_` in a 4,096-byte window after the registry, because the
registry's size varies. Five films of the authors' cache have no identity section; they are set aside and never read under a
neighbouring build's profile.

**Corruption-check flag.** The bit at `0x0CB45C`. When set, each present component of a record is followed by one guard bit and,
if that is 1, a 32-bit sentinel (`0x0bcddcba`); the same flag adds a guard bit and sentinel after each message of view A. The
game copies it into its run-time state only for films of major version `0x29`. It is 0 on all 1,605 films of the authors'
cache, so the guard path has never been exercised on a real film.

**Body.** The game reads the body (`FUN_1407ee138`, writer `FUN_1407ec560`) in this order: `R(3) R(3) R(2) R(7) R(64) R(32) R(3)
R(32) R(32) R(32)` (the first two are `game_mode` and `game_simulation`); a Bond message; a string of at most `0x80` bytes; two
`R(32)`; `R(1)` then, if set, `R(0x700)`; seven short fields; `R(1)` then, if set, the **game variant**; two strings of at most
`0x100` bytes; `R(32)`; `R(0x6C0)`; then the 32 player records. The variant is a Microsoft Bond CompactBinary (version 2) message
written byte by byte into the bit stream: a structure starts with its length, integers are 7-bit varints, a field header packs
`type | id << 5`. The decoder walks the whole body to the player table, skipping each Bond structure by its length, and reads
from the variant, by field path:

- the engine type and the playback flags `killcamEnabled` and `playOfTheGameEnabled`, which steer a few readers of view A;
- the match information that `decfilm.ReadMatchInfo` returns: name and description (first language of the variant's string
  table), score to win, time limit, minimum respawn delay, overtime and last-chance durations, starting weapons and equipment,
  and the option hashes with their values; with the start time and the two map-variant GUIDs of the table above.

A field the film does not write is absent from the result, never zero (the writer omits an optional field equal to its
default). If the walk cannot cross the variant (a type it cannot skip, a length that does not land on its end byte, a depth over
32), `VarianteLue` is false and no field is returned. The field paths were measured against libfilmshell, which reads the same
message, and were not re-read in the executable.

**Player table.** The body ends with 32 player records, one per slot of the match (the writer's bound, `0x28A00 / 0x1450`),
occupied or vacant:

```
 85 bits    header: 3 booleans (1/0/0 when occupied), u32, signed 2-bit value, 48-bit token
 64 bits    XUID (Xbox user id, 0x0009... to 0x000A..., read most-significant first)
 11 bits + N    presence mask, prefixed by the rank of its high bit      12 bits + N    length, then N bytes
 8 bits + M     length, then M words of 32 bits                          832 bits       104 raw bytes
 <= 16 UTF-16 units   gamertag, ending after the NUL unit                128 bits       16 raw bytes
 32 bits    desired-representation      64 bits    a key     46 bits    six short fields (10 + 14 + 6 + 8 + 7 + 1)
 per build  personalisation block: 1852 bytes (HI_1_12_0, HI_1_13_0), 1492 (HI_1_10_0, HI_1_11_0), 1312 (HI_1_8_0, HI_1_9_0),
            2052 (HI_1_4_1)
 352 bits   44 raw bytes (a second copy of the name; a bot's team sits here, section 3)       32 bits    tail
```

A vacant record (header `0/0/0`, null XUID) has a length computed from the same grammar. The personalisation width is read from
the executable on `HI_1_13_0` and measured elsewhere; the cause of the +200 bytes on `HI_1_4_1` is not established. The 32-slot
bound holds on all 1,351 measured films. Players who join later appear in type 8 packets.

**Status.** Read: both versions, the registry, the version table, the three strings, the corruption flag, the start time, the
body up to the player table, the variant fields listed above, the two GUIDs, the player records. Recovered: the build string, by
prefix search; the start of the player table, by a search (`chercherDepart`) that accepts a start only if the walk closes at 32
slots and visits every record the scan found. The code says the body holds serialisers that are not decoded; it also computes
the table's start by walking the body, and does not use that value for the table. Whether the two agree is unverified. Not read:
every variant field off the listed paths (the film holds option hashes, not names), the two name fields, the three 4,096-byte
blocks, the values measured zero. libfilmshell also reads a map `levl` tag id from a fixed bit of the body; this decoder does not.

*Sources: `internal/grammar/film_identity.go`, `film_format_version.go`, `film_major_version.go`, `film_variante_de_partie.go`,
`film_variante_bond.go`, `film_infos_de_match.go`, `player_table.go`, `registry.go`, `profil_balayage.go`;
`internal/profile/profile_table.go`, `build_profile.go`, `infos_de_match.go`, `registre_empreintes.go`; `NOTICE.md`; libfilmshell
`header.rs`, `registry.rs`, `variant.rs`.*

## 3. Roster and bot packets

Two packet types repeat the player record of chunk 0.

**Type 8, the roster** (the match's players). The payload is `R(32)` entries; each is, from format version `0x15`, a gate
`R(1)` and, if the gate is 0, `R(5)` (the controller index); then the XUID as 64 bits; then a player-record body, the same as in
chunk 0. Differences with chunk 0: no 85-bit header, no 32-bit tail, and the XUID bytes are in stream order, so the value read
most-significant-first is byte-reversed. The game rejects the packet only when it reads more bits than the payload holds; the
decoder reports both the overflow and the bits left over.

**Type 12, bot metadata** (a state, written again at the head of each chunk and at each change of bots). `R(32)` entry count; per
bot three `R(32)` (absolute index, slot, bot id) and a player-record body. A bot's team is a signed byte of the 44-byte block, at
a position that differs between `HI_1_12_0` and `HI_1_13_0`. Two readers coexist: a scan for names (a UTF-16 big-endian run of
printable ASCII closed by `0x0000`) and the writer's grammar. A team is kept only if the writer's walk closes the packet.

**Status.** Read: type 8 in full; type 12 by the writer's grammar. Recovered: bot names and slots by the bit scan, which fixes the
kill-source layer's roster; in a packet that closes, a scanned name that is the strict suffix of a name the walk reads is dropped
as a fragment and counted. Not read: a bot's team when the walk does not close or the build's team position is unknown.

*Sources: `internal/grammar/roster_type8.go`, `bot_metadata.go`; `internal/profile/equipe_bot.go`.*

## 4. A delta packet (type 0)

A delta packet is one tick of the simulation. Its payload is one bit stream of fixed structure: a configuration bit, then three
*views* in rank order, which this decoder calls view A (rank 0, events), view B (rank 1, entities) and view C (rank 2, control).

```
 bit 0    config:1     1 on every one of the 30,418 payloads of one measured film
 view A   { 1 | genre:7 | 3 x ( present:1 [ref] ) | payload(genre) | [guard:1 [32]] }*   then 0
 view B   { record }*   then record type 0 (the three bits 000)
 view C   { 1 | kind:2 | entry }*   then 0
 padding  0 to 7 bits, all zero when the packet is well formed
```

**Configuration bit.** The game keeps it in a global (`DAT_144706104`). When 1, the variable-width integers of view A take their
range from a per-category table; when 0, all use one default range. The decoder implements only the per-category table, so a
packet whose bit is 0 is read as far as its head. The first bit of a keyframe (section 7) is the same bit.

**View A, the event view.** A list of messages (weapon fire, damage, kills, pickups, effects...). A message is a continuation bit
`1`, a **genre** (`R(7)`, the message kind, 0 to 122; a value from 123 refuses the message), three optional references, a payload
read by the genre's reader, and the corruption guard. A reference is a presence bit and, if set, a **handle**: `W` bits of value
and two tail bits (a generation). `W` depends on a category (0 to 8) that the genre's descriptor gives each reference: 13 bits for
categories 0, 1, 7 and 8, 8 bits for 2, 3 and 5, 9 bits for 4 and 6; category 1 first reads a probe bit and, if it is 1, reads as
category 4. The stream carries no message length, so view A can be read to its terminator only if every payload on the way is
read. Of the 123 genres, 13 have no payload, 52 have a ported payload reader, and the other 58 stop the view after their genre.

The 123 genres are a static table of the game (`FUN_140e453b4`), held here as a string. Whether the film's writer is the reader
the decoder was ported from is decided by the film's own version table (section 2), compared genre by genre with the
executable's native versions, with no build name and no major version:

- **equal**: all 123 genres at their native versions. The end of view A is the start of view B, a reading and not a search;
- **prefix**: fewer genres, each at its native version (an older writer). The end of view A is trusted only if reading view B
  from it closes the packet. Genre numbering beyond 107 is presumed, not read;
- **unreadable**: no identity section, an empty or longer table, or a differing version. View A is read as far as its head.

The game writes the event list, a `0` bit, then view B, with no length prefix (`FUN_142f2c3b0`).

**View B, the entity view** is a list of records ended by record type 0 (section 5). **View C, the control view** loops on
`{ R(1); 0 ends; kind = R(2); handler(kind) }`; a source path in the game's code calls it `replication_control_view`, and its
entries are per-player control messages, not entities.

```
kind 0 (a participant): R(1) [R(7)]   R(5) control index (0..31)
    a:R(1) [block of 0x68 bytes: R(1)[R(2)]  R(6) R(6)  R(1)[R(5)]  R(1)[R(6)]  R(1)[R(5)]  action block]
    b:R(1) [block of 0xbc bytes]
kinds 1 and 2: a body that depends on run-time state        kind 3: no handler, zero bits read, the loop goes on
```

**Closure.** View C is the last rank, so a correct reading ends on its terminator with 0 to 7 zero bits left. A packet is *closed*
when it ends so and no rule of the game's writer is contradicted; a reading that reaches a terminator without closing the packet
has used a wrong width upstream, and its records are not trusted. The zero-bit requirement is this decoder's; the game rejects a
packet only on overflow, and libfilmshell accepts a view C ending within 0 to 7 bits of the payload's end without checking them.
Every delta packet ends in one of three states: **closed**; **closure refused** (the views were read to their terminators but the
packet does not close, so a width is wrong at an unknown place); **opaque tail** (the reading stopped at a known place for a typed
cause: an unported component, a view A message with no reader, a start of view B not found, a payload that ends inside a view, a
view C kind or block not read).

**Tick message.** Measured on the films, the first record of view B of a packet with events is an update of entity slot 123
(archetype `ti=4`, "high-frequency") naming one component, component 0, an 8-bit tick counter. Its width at the default frame is
35 bits: `1` (update), 13 of slot, 2 of generation, baseline `0`, sparse mask `0`, count `001`, index `000000`, then the counter.
libfilmshell's pattern fixes the generation at 1; this decoder's strict signature accepts any generation.

**Status.** Read: the head of every packet; view A message by message on equal films, and on prefix films when the packet closes;
view B records of ported components; view C kind 0 entries (the continuous-fire state of a player's triggers comes from them).
Recovered: the start of view B when view A cannot decide, tried in order by the tick-message signature, a chain of leading NEW
records of the event list, and the first start from which the walk closes (`repli_debut_de_liste_ferme_au_bit`); the kill-source
layer adds a free-width signature search (`repli_localisation_largeur_libre`) and a chained reading of kill messages before view B
(`kills_rattrapes.go`). Not read: view A past the first message with no ported reader, or when the configuration bit is 0; view C
kinds 1 and 2 and the `0xbc` block (each stops the view and is counted as a typed stop).

*Sources: `internal/grammar/marche_trames_rangs.go`, `vue_a_lecture.go`, `vue_a_versions.go`, `vue_a_genres.go`, `vue_a_charges.go`,
`varwidth.go`, `localisateur.go`, `frame_vue_controle.go`, `frame_records.go`, `ecrivain_invariants.go`,
`internal/grammar/lecture/paquet.go`; libfilmshell `payload.rs`, `control.rs`.*

## 5. An entity record (view B)

Each record starts with a record type, then an entity id.

```
record type   1                 -> DELTA (update)
              0 + R(2):  0 end   1 NEW (create)   2 DEL (delete)   3 DELTA
entity id     low: R(13)        slot (entity index)            [width presumed, see below]
              tag: R(2)         generation (the two high bits of the 32-bit id)
```

The 32-bit **entity id** is `generation << 30 | slot`. The **slot** (low 30 bits) indexes a table of up to `0x1fff` entries; the
**generation** is incremented by the game when a slot is reused, so two ids with one slot and different generations are different
entities, and the game's guard compares all 32 bits. The 13-bit width is the value of the game's static image
(`DAT_144706100 = 0x1FFF`); two writers of the game rewrite that global at run time, so the decoder takes 13 as presumed and uses
it on every film. Older source comments say the width differs between films (11 and 14 are cited); that is not established.
libfilmshell also uses 13.

- **NEW**: `1 | id | ti:6 | creation state | gate:1 | mask | component data`. `ti` of 50 or more is a misread. The creation state
  is the archetype's default state, read by a per-archetype deserialiser; there is no size word. A NEW is the only place where a
  delta packet states an entity's archetype.
- **DELTA**: `3 | id | baseline:1 [7 bits] | mask | component data`; no type, no creation state. The baseline bit is 0 in every
  measured film.
- **DEL**: `2 | id | 32 bits`; the decoder skips the 32 bits, whose meaning is not established.

**Component mask.** `R(1)`: when 0, `R(3)` gives a count and that many `R(6)` component indices follow, strictly increasing
(sparse, up to 7); when 1, `R(64)` is a dense bitmask (the writer uses it only above 7 components). Present components are read in
ascending order. The decoder checks the writer's rules for masks and counts a packet that breaks one.

**Component data.** The width is not in the stream. It comes from the component's deserialiser, found by (archetype, component
name) through the registry, and may depend on the build, the format version and run-time tables (the multiplayer-properties
block of objects, the map's position widths). This is where the port of the game's readers lives. The status of the 1,067
registry entries of the reference build is in `internal/grammar/testdata/ecs_table.tsv`: 606 ported, 79 partly ported, 350 not
ported, 32 with a written grammar and no caller, plus 14 spelling aliases. The statborg archetype (scoreboard counters) has no
parser: its bits are consumed to keep the chain aligned and its values are found by anchoring on record headers, because an
earlier port read 841 records out of 841 wrongly.

**Entity table and guard.** The reader keeps a table slot -> (full id, archetype, provenance of the binding). The game's guard on an
update requires the id to be in its datum table (slot in range, stored id equal to the record's) and otherwise reads no bit of the
body; the decoder reproduces this, and an update not in the table ends view B as a *rejection*. A binding comes from a NEW the
decoder read, from the chunk's keyframe (record chain or header search), by anticipation, or by inference (section 7).

**Archetypes named in the reference table.** 0 game engine; 4 high-frequency; 5 player (the participant); 6 statborg; 9 managed
player; 14 crew; 21 flock; 35 biped (a player's body); 37 equipment or item; 38 generic rigid body; 40 vehicle (presumed); 41
projectile; 42 ground weapon; 43 device. libfilmshell also names 36 (item) and 39 (rigid body), which this table leaves unnamed.

**Status.** Read: record type, id, mask, and the components with a ported reader, in NEW and DELTA records. Recovered: the
archetype of a slot with no NEW in view, from the bindings above; records anchored by header patterns where a body cannot be
read: biped records, world-object tracks and creations (bit passes after the walk), scoreboard records, and kills of a view A that
did not close. Not read: a component with no reader (the traversal stops at its start with zero bits consumed and the rest of the
view is an opaque tail); the 32 bits of a DEL; the creation records of entities born between two keyframes (bound by anticipation
instead). An entity present at the start of a chunk but absent from its keyframe table, and designated by an update before any
NEW, is a measured gap.

*Sources: `internal/grammar/frame_records.go`, `entete.go`, `traverse.go`, `ecrivain_invariants.go`, `world.go`, `frame_infer.go`,
`doc.go`, `testdata/ecs_table.tsv`, `testdata/ecs_table.README.md`; libfilmshell `payload.rs`, `events.rs`.*

## 6. Positions

Positions are quantised: a gate, an optional region index and three quantised axes.

```
position   gate:1    0 -> region index: indexW bits (a row of the map's region table)
                     1 -> default row (bounds +/-20000 on each axis), index = -1
           x: w_x bits   y: w_y bits   z: w_z bits          (widths from the row for (region, level))
value      min + step * (q + 0.5),    step = (max - min) / 2^w            per axis
```

**Region.** A map is divided into compression regions, one per structure BSP of its level, each with a world bounding box. The
film carries only quantum indices; the boxes come from the map's data in the game files. The decoder reads them from a JSON
catalogue supplied by the caller and refuses a map with no entry rather than read it with another map's widths (the catalogue
used by the module's tests lists 101 maps and 196 regions). `indexW` is 1 when the map declares one region, otherwise
`ceilLog2` of the raw region count. The index selects the row, so an axis's width is that of the row the stream designates, not
that of the region the match is played in.

**Precision level.** The level only chooses a row. It is an immediate at each reader's call site, never a value of the stream or
the registry: 16 for the position components of bipeds, objects, flocks and vehicles, 15 for a projectile-detonation message, 12
for a projectile-impact message.

**Width law**, from the game's map-load routine (`FUN_140be9a14` calling `FUN_140be9b88`):

```
step(L) = 2^(16-L) / 120   for L <= 16        step(L) = (1/120) / 2^(L-16)   for L > 16     (1/120: constant of the executable)
w(axis) = min( 26, ceilLog2( min( ceil( extent / (2 * step(L)) ), 2^22 ) ) )                  w = 26 on all axes if step(L) < 1e-4
```

At level 16 the default row (extent 40,000) gives 22 bits per axis; the test catalogue gives 11 to 20. The law matches the 79 maps
of an earlier catalogue to the bit.

**Other forms.** When the game's full-precision guard is true (`FUN_14076f91c`, the scope of a keyframe entry's full-state read),
a position is 96 raw bits (three floats). The object position component (world-object component 0) reads a precision bit first;
if it is 1, three axes over +/-100 (14 bits each at level 16) follow with no gate or index. When the index designates a region with
no row in the map's table, the game reads zero bits per axis, raises no error and does not fall back to the played region; the
decoder does the same and counts it (`IndexSansLigne`). libfilmshell reads such a box with the played region's widths.

**Status.** Read: gate, index, axes, width law, each call site's level. Recovered: nothing by search. Not read: a handful of call
sites keep an older reader because reading them as the game does lowered closure on the witness films; the source counts eleven,
each a dated exception in `lecteur_position_exceptions.go` (the precision-bit-set branch of the world-object and biped absolute
positions, a few tactical-map components, the crew order, two vectors of the unit actor state). A catalogue entry with no
per-index table reads only the played region's index, and the reading stops after any other index.

*Sources: `internal/grammar/lecteur_position.go`, `lecteur_position_exceptions.go`, `components_position_monde.go`, `quantize.go`,
`testdata/map_quant_bounds.json`; `internal/profile/loi_largeurs.go`, `map_bounds.go`.*

## 7. The keyframe (type 2) and the datum block (type 1)

At the head of each replay chunk the game writes what a player needs to seek to it: a datum block (type 1), then the keyframe
(type 2). A *datum* is the game's name for one entry of its entity table.

**Type 1.** 343,019 bytes on the reference films (27 of 27 chunks of one film, 5 of 5 of another). Its grammar is the writer's
(`FUN_1429883ec`) and its size closes by arithmetic:

```
for slot in 0 .. 8190 (8,191 slots):  flags:6  generation:8  generation counter:32  view mask:33    = 79 bits
for slot in 0 .. 8190:                component presence bitmap: 256 bits
5 x R(32)                             tail            total 8,191 x 335 + 160 = 2,744,145 bits -> 343,019 bytes + 7 padding bits
```

On the witness films the flags are `0x5` (live), `0x0` with a generation (freed) or `0x0` without (never allocated), and the counter
is always the generation plus 1 (measured on 221,157 entries of one film). The bitmap is the presence mask of the archetype the
keyframe gives the slot, and is constant per archetype. 8,191 (`0x1fff`) is the table size on the two witness films; the reader
derives it from the block's size.

**Type 2.** The game's reader (`FUN_142e2bfd0`) reads the whole entity table: a first bit (the configuration bit, when the format
version is above 7), then one entry per slot, in slot order 0 to `0x1ffe`, empty slots included, with no index or count:

```
entry header, 108 bits:  id:32  type:32  handle:32  flags:4  byte:8
  id      generation in the two high bits, slot below       type    0xffffffff = empty slot: nothing follows the header
  handle  0xffffffff = none
if type is set:  n:32 [creation state of the type, if n > 0] [R(32) control word, with the corruption flag]
                 m:32 [state: each component of the archetype in registry order, no mask, no length]
```

`n` and `m` are the sizes of the game's structures, not lengths in the stream. An entry keeps its id and type after its entity is
deleted, until the slot is reused, so a delta carrying the old id still passes the game's guard. The flags are 3 on nearly every
entity (also 0 and 1); their meaning is not traced.

Creation and state carry no length, so reading the table as the game does would need every type's creation and state readers.
The decoder instead rebuilds the table by two mechanisms, the second a recovery:

1. *The record chain.* A header is accepted where its guards hold (generation non-zero, slot above the previous and below `0x2000`,
   archetype word below 50); its body is read by the full-state loop, and the record is proved when its components end exactly on
   the next accepted header. When the chain breaks, an anchor may be elected by a named fallback
   (`repli_ancre_d_image_cle_par_election`).
2. *The header search.* Entry headers are found at any bit by their invariants: slot below the table size, type below 64, the 17
   id bits above the slot zero, no high bit set in the byte, flags 0 (no handle, creation or state), 1 or 3 (`n` at most 256; 212 is
   the largest measured), slots strictly increasing. A first pass keeps generation-non-zero entries only, because generation 0 is
   also what the allocator writes on a slot's fourth allocation and admitting it everywhere lets bit coincidences in other entries'
   data in; a second keeps generation-0 entries only between two recognised entries. Against libfilmshell's table on five films it
   found 102,155 of 102,157 entries, none false. Its binding carries the full id and replaces a chain or NEW binding that disagrees.

An update whose slot neither mechanism binds, and that no NEW named, is bound by **anticipation**: the reader consults the table
built from the keyframes of all chunks and takes the first later declaration of that (slot, generation). On one dense film 74.7 %
of the rejected slots were declared by the next chunk's keyframe. The creation record is still not read. libfilmshell also finds
each live entry by its header (the index climbs and fixed bits rule out chance).

**Status.** Read: the type-1 grammar and its arithmetic (an exported reader used by measurement tools, not by the production walk);
the 108-bit header; the record chain with the entry bodies of ported archetypes, by the full-state loop. Recovered (each binding
is marked with its provenance and counted): elected anchors, the header search (`DatumBindings`, `DatumAmbiguous`), anticipation,
and an archetype inferred from the records that follow. Not read: creation and state of entries found only by the header search;
the flags; entry bodies past the first component without a ported reader.

*Sources: `internal/grammar/type1_datums.go`, `keyframe_entrees.go`, `keyframe_datums.go`, `keyframe_world.go`,
`keyframe_fullstate_loop.go`, `keyframe_liaison.go`, `keyframe_anticipe.go`, `lecture/record.go`; libfilmshell `snapshot.rs`.*

## 8. The highlights chunk

The last chunk (manifest type 3) is the Theater timeline: kills, deaths, medals and mode events, each with a time in milliseconds.
This decoder does not cut it into packets. It scans a bit-packed sequence of event records that begin at any bit offset, so a scan
aligned on bytes misses most of them (XUIDs spread uniformly over the eight alignments). The layout follows the SPNKr project's
Python reader, as libfilmshell's does.

```
event record, anchored on its XUID:   64 bits XUID (little-endian, in 2e15 .. 3e15) | 0x2d or 0x25 | 0xc0 | ...
event block: the 60 bytes that immediately precede the end marker 00 00 2e e0
```

The decoder tries every bit position for `0xc0` preceded by `0x2d` or `0x25` and a plausible XUID, searches the next 20,000 bits
for the end marker, reads the 60 bytes before it, and keeps the closest XUID when several claim one marker. The block decodes by
the film's major version. For versions up to 38 and from 41: gamertag UTF-16LE `[0:32]`, padding, type hint `[47]`, time in ms
(u32 big-endian) `[48:52]`, padding, is-medal `[55]`, padding, medal type `[59]`. For versions 39 and 40 the gamertag is at
`[12:44]` and `[0:12]` is padding; the rest is as above.

The type is a medal when is-medal is 1 and the hint is a medal sorting weight (50, 51, 52, 100, 101, 150, 200, 205, 210, 220, 225,
230, 235, 240, 245, 250); otherwise 10 is a mode event, 20 a death, 50 a kill; any other hint is ignored. libfilmshell reads the
same records and layout, counts an unknown hint as a kill, reads the player's team at byte 38, and expects the end marker at a
fixed 15,406 bits from the XUID where this decoder searches. It also reports that the chunk starts with a packet of type 9; whether
a 16-byte packet header precedes the events here, and whether the scan crosses it without effect, is unverified.

**Status.** Read: the event block, by major version. Recovered: every record, by bit scan and marker search. Not read: the team
byte, the chunk's packet structure, records with an unknown hint.

*Sources: `internal/grammar/highlight_events.go`; libfilmshell `highlights.rs`, `chunk.rs`.*

## 9. Closure by build, and what is open

**Closure by build.** The share of delta packets the reading closes (section 4), measured on 2026-11-12 on 38 films, each with its
map in the catalogue:

| Build | HI_1_13_0 | HI_1_12_0 | HI_1_11_0 | HI_1_10_0 | HI_1_9_0 | HI_1_8_0 |
|---|---|---|---|---|---|---|
| Closed packets | 99.85 % | 99.95 % | 99.85 % | 96.56 % | 98.81 % | 73.43 % |

Closure says whether packet widths are right, not whether every field is decoded to a fact: a closed packet may hold components
skipped by width. The profile holds values read from the executable only for `HI_1_12_0` and `HI_1_13_0`; for the other builds
they are measured on witness films. Fallbacks and recoveries still fire on films of every build. The fallback registry
(`internal/facts/fallback/`) holds about 130 entries across all layers (129 declarations at the time of writing); each fallback
that fired is counted in the replay document.

**Open questions**, as the code leaves them:
- the width of the entity id's low part is presumed 13 on every film (section 5); the meaning of the `u32` after each registry
  name, of the two packet-header bytes, of the keyframe flags and of the 32 bits of a DEL record;
- the frame configuration carries an off-by-default hypothesis of a 32-bit prefix per record, gated by the same game predicate
  as the corruption flag; that flag being 0 on every measured film, the hypothesis cannot be tested on them;
- the view C short form (the game derives two field widths from the engine type) is kept and arbitrated by closure, not read per film;
- the payload widths of three view A genres (12, 14, 106) were measured against an independent decoder, not read in the executable.

## 10. Where this decoder and libfilmshell differ

| Subject | This decoder | libfilmshell |
|---|---|---|
| Packet walk | stops at the first incoherent header | refuses a chunk whose walk does not end on its last byte, then splits on a byte marker |
| Closure | the last 0 to 7 bits must be zero | fewer than 8 bits left |
| View C kind 3 | zero bits read, loop continues | stops |
| Region index with no row | zero bits per axis, counted | the played region's widths |
| Highlights | unknown hint ignored; end marker searched | unknown hint counted as a kill; end marker at a fixed distance |
| Registry first `u32` | the film major version | a header count (41), not the number of blocks |
| Game variant | engine type, playback flags, match information by path | the same message; the two agree on every compared field of the films both read (`NOTICE.md`) |
