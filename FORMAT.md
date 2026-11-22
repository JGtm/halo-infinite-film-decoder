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
  table), score to win, time limit, minimum respawn delay, the seconds the variant adds to a respawn after a suicide and after a
  betrayal, the friendly-fire flag (the flag numbered 3 of the set of rule flags), overtime and last-chance durations, starting
  weapons and equipment, and the option hashes with their values (the flag return delay is the option `0x2b16b76f`); with the start time and the two map-variant GUIDs of the table above.

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

**Seat of a table row.** The rank of a row is the participant index of its occupant on most films, not on all: on 31 of 1,606
films with a readable table, one to six rows are listed by the type-8 packets under another seat. On the 28 of them that were
fully decoded, the chunk replication reading names the listed seat on 46 of the 48 rows (the other two have no reading), never
the rank, and the team table has no player at the rank on 33 of the 46. When the first type-8 packet lists every occupant of the
table under a seat nobody else holds, that seat replaces the rank (`SiegesParLesListages`); the table then lists who is there at the start, and has no interleaved
vacant row left to settle.

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

The packet is sent when the population changes, not on a cadence: the first packet that names a player dates their arrival, the first
packet that no longer names them dates their departure, and a player named by the first packet is there at the start. The decoder
reads the packets the source has already cut (no bit scan), skips a packet identical to the previous one, and keeps per human the
runs during which an XUID is listed under one engine seat (the `R(5)` index): a player who leaves and returns has two runs, and a
joiner who takes a leaver's seat opens a run of their own. On 1,455 films: 49,625 packets, 7,641 distinct, none refused; every human of the
published roster is listed; the 154,247 lives of listed humans fall inside the runs but 36 (a body still moves 2 to 10 s after its
player left). libfilmshell reads the same packets with the same method (first listing, first listing without the player).

**Type 12, bot metadata** (a state, written again at the head of each chunk and at each change of bots). `R(32)` entry count; per
bot three `R(32)` (absolute index, slot, bot id) and a player-record body. A bot's team is a signed byte of the 44-byte block, at
a position that differs between `HI_1_12_0` and `HI_1_13_0`. Two readers coexist: a scan for names (a UTF-16 big-endian run of
printable ASCII closed by `0x0000`) and the writer's grammar. A team is kept only if the writer's walk closes the packet.

**Status.** Read: type 8 in full; type 12 by the writer's grammar. Recovered: bot names and slots by the bit scan, which fixes the
kill-source layer's roster; in a packet that closes, a scanned name that is the strict suffix of a name the walk reads is dropped
as a fragment and counted. Not read: a bot's team when the walk does not close or the build's team position is unknown.

*Sources: `internal/grammar/roster_type8.go`, `roster_listages.go`, `bot_metadata.go`; `internal/profile/equipe_bot.go`.*

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
Three of the 52 (genres 12, 14 and 106) have a layout measured against another decoder and not read in the game; they are read only
on builds `HI_1_11_0`, `HI_1_12_0` and `HI_1_13_0` (the profile's per-build table, `internal/profile/ecriture.go`), and on any other build the view stops after
the genre like the 58.

The 123 genres are a static table of the game (`FUN_140e453b4`), held here as a string. Whether the film's writer is the reader
the decoder was ported from is decided by the film's own version table (section 2), compared genre by genre with the
executable's native versions, with no build name and no major version:

- **equal**: all 123 genres at their native versions. The end of view A is the start of view B, a reading and not a search;
- **prefix**: fewer genres, each at its native version (an older writer). The end of view A is trusted only if reading view B
  from it closes the packet. Genre numbering beyond 107 is presumed, not read;
- **unreadable**: no identity section, an empty or longer table, or a differing version. View A is read as far as its head.

The game writes the event list, a `0` bit, then view B, with no length prefix (`FUN_142f2c3b0`).

**Game events (genre 82, `PlayerGameEventSmall`).** The mode script's reply to a player: a kill, an assist, a flag capture, a personal
score. The payload is `R(32)` **rank**, `R(8)` sequence number, a property bag (`R(3)` entries, each a name `R(32)` and a typed value),
a text bag (`R(1)`; if set a `R(32)` text word, `R(3)` slots, each a 3-bit type then its value: type 1 is a participant, `R(1)` absent or
`R(5)` index), and a 32-bit mask of the recipients (the first bit is participant 0). The message names no response and carries no
points: the rank is a position in the event tree of the film's mode scripts, so the same rank is another response in another mode. The
scripts are named in chunk 0: strings ending in `.lua`, written byte by byte 3 bits after a byte boundary
(`internal/grammar/film_scripts.go`); `slayerevents.lua`, or `arenaflagevents.lua` with `flageventstubs.lua`, fix the tree. The game sends
one copy per recipient, and may send a copy again 10 to 25 ticks later with the same sequence number: the decoder keeps the first copy
of an identical (rank, sequence, text word, mask) seen in the previous second. A participant slot is the film's player index, the same
index space as the recipient mask. On 11 films it reads exactly the messages libfilmshell reads (2,085 of them carry personal score);
a view A that stops at an unread message loses the ones behind it, which the films of builds 38 and 39 measured do, for a cause that is still open.

**Score responses.** A message names its response only through a table: for each group of event scripts, every rank of its event tree, with
the response's script name (`killPersonalScore`), its kind and, for a personal-score response, its points, taken from libfilmshell
(`reponsesdejeu/data/reponses.tsv`, see `NOTICE.md`). The group is the table group whose script files are exactly those of the table that the
film's chunk 0 lists (libfilmshell's rule); a film that lists the files of no group names no response. The tree lists the events that are
responses: a rank outside it is an event that is not one (most of a film's messages), or a response the table does not hold. A response that
has a text of its own is attributed only if the message carries that text word (or none), and a rank the tree does not place for certain
needs the text word; a rank the message does not confirm is counted apart. The recipient mask gives one action per recipient, and an index
is resolved to the xuid of the roster human who holds it and is present (`presence`) at the frame of the message: a human who left keeps
the index while a bot declared on it is not in the roster, so a message after the departure is counted, not given to the departed.
A `%pe` (the enemy killed) or `%pa` (the player betrayed) in the response's label is the first participant of the text bag, so for
`betrayalPersonalScore` the recipient is the betrayer and the first participant the betrayed player. The replay document publishes these as
`scoreActions[]`, on the frame `(TS - origin) / step` of the positions, with `coverage.scoreActions` counting what was read, published and
left out: messages that are not score (announcements, medals, kill-feed lines), score responses the game pays zero, ranks outside the tree,
ranks the message does not confirm, and the actions not placed. `coverage.scoreActions.complete` says the reading is whole (no stopped
frame, a readable view A, a known script group, no unconfirmed rank, every action with points placed). It is necessary, not sufficient, for
a player's sum of points to equal the personal score: the points come from the response table, which can differ from the game (vehicle
actions on older builds) or know nothing of a mode written outside the tree. False, the consumer shows "insufficient data"; a consumer
that holds the official score compares the sum to it. A mode written without a point scale (Assault, a custom mode) has no response that
gives points: nothing is invented.

**Damage messages (genre 0, `damage_aftermath`).** One effect of damage that lands on a unit. The header names the unit hit (reference 0)
and the unit that caused the damage (reference 1, the same domain as reference 0 of a weapon-fire message); the payload names the damage
effect by its tag id (`R(1)` gate, then `R(32)`), the attacker's player index (`R(1)` gate that reads `R(5)` when it is *clear*, the inverse of
the effect's gate: a fall or the map writes no index), a 19-bit direction and, after three small fields (5, 5 and 6 bits), the scale: 15 on a
full direct hit. The effect belongs to a weapon, a melee strike, a grenade, the world...: the table `damageeffects/data/effects.tsv`, taken from
libfilmshell (see `NOTICE.md`), says which. The replay document publishes an **estimate** of the hits from them: a hit is a weapon effect
counted once per (attacker, unit hit, packet), projectiles included, on the frame of the packet; the weapon is the one of the attacker's last
shot in the two seconds before. The victim is the player whose track holds the unit (slot = unit index + 512) at that frame. Per player,
`shotStats[]` gives the shots (the sum of the jumps of the game's per-player shot numbering, modulo 256, which counts the shots the film does
not write), the weapon-fire events written, the damage messages and the hits, and two ratios: the *coverage* (events written over shots) and the
*duplicates* (messages over hits). A player's data are `sufficient` when the coverage is at least 0.7 and the duplicates below 1.2: measured
against the API's `shots_hit`, the hits of those players are within 10 % for about four players in five, and the shots within 10 % for more than
four in five (chronicle v125). When they are not sufficient the consumer shows "insufficient data", never a count. A film whose messages the view
A cannot read to the end has no hit: all its players are insufficient.

**Damage in points.** Each damage message that removes points from a unit with a track becomes a `damage[]` row (frame, attacker, victim, effect,
weapon, points taken from the shield and from the body, tenths of a point). The points come from the amounts table of libfilmshell
(`damageeffects`, MIT, credited in `NOTICE.md`) applied to the unit's real life, which the film writes in the biped's two vitality components
(`object-body-vitality`, health from -1 to 1; `object-shield-vitality`, shield from 0 to 4, 1 being full): the messages of one unit at one instant are
computed together from the life read just before that instant; a direct hit of a fixed-amount effect at full scale (15) removes its amount from the
shield, then the body (multipliers and flags of the table); a known effect that removes nothing removes zero; any other hit (area, reduced scale,
unknown effect) is *measured* from what the life loses, shared between the measured hits and never beyond what remains. A shield rises only when
the film writes it, never by an assumed delay. A Spartan has 90 body points and 140 shield points. `shotStats[]` adds `damageDealt` and `damageTaken`
(a player's damage to himself is not counted as dealt; taken counts players only), under the same `sufficient` verdict as the hits. A message whose
unit has no track (a vehicle, an object) is neither published nor counted. Estimates: against the API's `damage_dealt` and `damage_taken`, for the `sufficient` players the median
ratio is 1.01 and 1.02 and about three players in four are within 10 % (chronicle v125).
The totals of `shotStats[]` cover the whole film, while `damage[]` holds only the messages whose instant falls inside the frame axis (before frame 0
or after the last frame a message counts in the totals and has no row). A weapon hit on the shooter himself (an area weapon) is neither a hit nor
damage dealt: it is counted apart (`coverage.hits.selfHits`, `selfDamages`). The biped's vitality records are re-read on the lists the walk does not
locate, as the death channel does; the frames that stay unreadable without the view A stopping there are counted (`coverage.hits.vitalsUnread`) and enter,
with the stopped frames, the share of frames above which no player is `sufficient`. LIMIT: that share is per film; lost messages may concern a
single player.

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
  delta packet states an entity's archetype. On builds `HI_1_11_0` to `HI_1_13_0` the gate is read only for an archetype whose creation mask is
  not empty (it is absent for archetypes 7, 13, 14, 20, 23, 28, 29, 46, 47 and 49), and a gate of 0 ends the record: the mask and
  the component data are absent. On any other build the gate and the mask are always read. Archetype 13 (`managed-object-property-name`)
  carries the payload of each of its tags in its creation state on those same builds, and only the tags elsewhere.
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

**Scoreboard rows.** The statborg holds one entity per team and one per player, in contiguous datum indexes from 3 (two teams, then
a row per seat); a record's slot is twice that index. Every such entity raises component 57 in its datum bitmap, so the last seat of
a match is read from the datum tables of the film (`internal/grammar/signaux/statborg_bande.go`) and records are anchored up to that
slot, 58 on a 25-seat match. A row is a list of components of two values (A, B): component `c` side A or B is libfilmshell's
slot `2c` or `2c+1`. Named and measured: 0A mode score, 1B personal score, 2A kills, 2B deaths, 3A assists, 4A suicides, 4B
betrayals, 5B longest kill spree, 6A headshot kills, 8A melee kills, 8B grenade kills, 9A power weapon kills, 10B spawns. A row
is cleared to zero when its player leaves. Components 0 to 27 (current-round values) are the ones the reader of the decoder
reproduces (`FUN_140C18794`): an emission always decodes all of them in its packet, so an anchored record on which one of them does
not decode is not an emission and is dropped whole, the components read before it included. A stop on a component of another reader
(frozen round values 28 to 55, outcomes 56, entry 57) keeps what was read before it.

**Whose row it is.** Each statborg entity carries its own entry, component 57 (`statborg-entry-index-and-type`): a 32-bit value and an
8-bit kind, written at creation and read at the keyframes (`internal/grammar/statborg_entries.go`, one record per entity and keyframe).
Kind 0 is a player's row, and the top 16 bits of the value are the player's engine id, the seat that the type-8 session listings give
(`RosterListing.Slot`); kind 1 is a team's row, and the low 16 bits are its number; any other kind is a vacant entity. The player of a
record is the one the listings show under that id at the record's time, taking the entry of the last keyframe that does not follow the
record and does not precede its round (`internal/facts/objectives/entry_identity.go`). Nothing is paired by time or by counters, and a
match of several rounds is named like any other. The entries are trusted only when they prove the layout of the scoreboard: the table
opens on its two team rows, so two distinct team numbers must be read in kind 1. A film whose reading does not land on component 57 (builds
whose scoreboard layout the film does not carry) reads every entry as zero, which would give one engine id to every player row and name
a single player on all of them; such a film is named by the identity bridge instead, and its rows that the bridge does not pair publish
nothing. When the rows are named by the entries, the kills, deaths and assists components are read with their counter channels in
domain: a record whose kills or deaths (2A, 2B) or assists (3A) channel is negative, or whose assists record carries a negative
channel C, is misaligned and dropped; the signed points of the assists component (3B, 3D) are legitimate negatives. The identity
bridge does not apply this rule: a film named by the bridge reads those components as they are.

**Rounds played.** Component 56 of the same archetype (`statborg-round-outcomes`) is 64 bits, 32 two-bit round outcomes, zero until the
round is played. The reader keeps the mask of rounds that ever show a non-zero outcome at a keyframe (`StatborgEntryStats.DecidedRounds`).
A round is played when its number is at most the last decided round plus one (the round in progress). Records of a round beyond that
are anchored noise (a single component, value 32, carrying a valid round designator) and do not make a scoreboard round; the rounds
dropped are listed in `coverage.score.roundsUnplayed`. The round clock cannot tell them apart: 18 of 51 films of several rounds have
a silent clock. A row that is cleared when its player leaves does not retract the player's last value: a zero after a positive value of
a base counter (kills, deaths, assists, personal score) is dropped.

**Entity table and guard.** The reader keeps a table slot -> (full id, archetype, provenance of the binding). The game's guard on an
update requires the id to be in its datum table (slot in range, stored id equal to the record's) and otherwise reads no bit of the
body; the decoder reproduces this, and an update not in the table ends view B as a *rejection*. A binding comes from a NEW the
decoder read, from the chunk's keyframe (record chain or header search), by anticipation, or by inference (section 7).

**Archetypes named in the reference table.** 0 game engine; 4 high-frequency; 5 player (the participant); 6 statborg; 9 managed
player; 14 crew; 21 flock; 35 biped (a player's body); 37 equipment or item; 38 generic rigid body; 40 vehicle (presumed); 41
projectile; 42 ground weapon; 43 device. libfilmshell also names 36 (item) and 39 (rigid body), which this table leaves unnamed.

**Who controls a body.** A biped's `unit-control` component (`i18`) opens with a gate and, when it is set, a 5-bit player index, then an
optional second gate and 6-bit index, then an optional 32-bit word. A biped is created with the index of the player who controls it
and the component is written again, gate closed, when that player gives the body up, which a death does: on 21 films the closing
read falls within 0.1 s of the end of the published life for 2,816 of 3,075 lives that have one (3,036 within 0.5 s). The index is the one the player table, the kill feed and the type-8
seats use; it names a player only through the seat the type-8 packets give it at that instant (a joiner can take a leaver's seat), and
an index no human holds is a bot (bot metadata gives its seat). The record that creates the body is read before its slot is bound to
the biped archetype, so the decoder keeps each read until the end of the packet and keeps it if the entity table then binds the slot to a
biped. On 21 films, 3,230 lives are named by their controller as they are by the published tracks and none by a different human; the
film `4f77afc1` shows lives of a bot (seat 1 from 1,101 s) published under the human who left that seat at 86 s.

**How a body ends.** The same component tells how a life ends, and a second record tells the other way. A player who dies gives the body up (the closing read above); the game deletes the body some seconds later. A body that a round's end or a reset removes is deleted while a player still controls it, with no closing read. The decoder reads both, and does not judge them: the replay document publishes `tracks[].endReason`, `released` when the controller gives the body up (with `releaseFrame`, the frame of that read, on the document's axis), `deleted` when the body is deleted still controlled or when another player's read takes the slot with no deletion read in between, and nothing when no read says it, or when the body ends at the film's last frame or past it (the film stops writing and lets its lives go: not a life end, counted in `coverage.tracks.lifeEnds.filmEnd`). An end on the very frame where the slot's next track opens belongs to the track before. A DEL record is kept only when the walk saw its slot carry a biped of the record's generation (the two top bits of the id) and the packet's list start is proven; a DEL of a walk that starts at an unproven point does not unbind the world and is only counted (`coverage.tracks.lifeEnds.deletionsUnproven`). A body opens at the first read that puts a player in control and closes at the first release, deletion of the slot or read of another player; a track belongs to the last body opened at its first point, so the first `unit-control` read of the body bounds its birth with no time window. Measured on 99 films (18,006 tracks): 15,905 released, 546 deleted (37 of them by a taken slot), 906 with no controlled read (bots and old builds, mostly), 120 controlled with no end before the slot's next track, 529 ending at the film's last frame or past it, 96 unproven DELs. Of the 15,290 closed lives a kill claims, 15,113 end in a release from 2 frames before to 6 frames after the track's end and none beyond; of 653 closed lives with neither kill nor respawn countdown whose body is controlled to the end of the reads, 495 are deleted (16 by a taken slot), 158 have no published end (film end, no end read before the slot's next track, or no controlled read) and none is released. Whether a release is a death is the consumer's reading: a release also follows the end of a match, and libfilmshell counts a release as a death only where its scoreboard counts one.

**Death and parenting.** An entity that dies writes its own dead-state component (`object-dead-state`, `i11` on archetypes 35 and 40) in the packet where it dies: a vehicle's carries the killer's player index and the damage effect, a biped's also the victim's player index. A biped inside a vehicle is not linked to it from the vehicle side; the biped's own parent-state component (`object-parent-state`, `i10`) is a transition, attached to a parent entity or free, and the last one before the biped's death says what it was in. A player killed with a destroyed vehicle is therefore read from two facts, with no time window: the biped's dead-state is written in the vehicle's death packet (or in a later packet that also writes the biped's free read), and the biped's last parent state before it was attached to the vehicle or to a part mounted on it. A death whose damage effect is the end-of-round cleanup (4208020) is not counted.

**Match clock and respawn countdown.** Two components of the engine entities are kept, in the order of the stream.
The game engine entity (`ti=0`, `game-engine-round-timer-component`, `i5`) writes the round's limit and the timer's time as two 16-bit
quanta over 0 to 36,000 s, then a 5-bit state; it writes them every second while the clock runs. The state is the index of a
table of timer rates: on the 1,345 films measured, 1 (stopped), 6 (counting down one second per second) and 19 (counting up one
second per second) make 99.9 % of the reads, and each other value under 0.02 %. The limit is the variant's time limit to the nearest
quantum (300 s is written as 300.21 s, 240 s as 239.78 s); the timer is reset to the limit when a round starts. A player entity of the engine (`ti=5`,
`player-respawn-timer-component`, `i1`) writes a flag, the seconds left and the seconds in all of its player's countdown, one write
per second while it runs; the flag is 0 in 99.3 % of the reads and its meaning is not established (the decoder does not use it); the seconds in all equal the
variant's minimum respawn delay on 3,729 of 3,931 countdowns of 25 measured films, and exceed it by one second on 202.
The index of the player that an engine player entity belongs to is the `R(6)` of its creation state, after the version prefix: it is
read from a NEW record in a delta packet, or, for an entity that is there when the film starts, from the keyframe record. That index
names the seat the entity was created for, not the occupant at a given instant: a player who arrives and takes over a seat reads the
index of the one who left, so a countdown is only attributed when the index has had a single occupant.

**Return gauge of a dropped flag.** The managed objects of a mode (`ti=13`, "managed-object property") carry named network
properties; a delta packet writes one only when it changes. A dropped flag's return gauge is the scalar value (`i1`, a variant of tag 3,
24 bits quantised over -100..+100, read as a fraction of 0 to 1) of the property of one such object. The decoder reads it from the
records of view B that the walk already reads (`internal/grammar/canal_des_proprietes_gerees_trames.go`): each component of the
archetype's two value kinds, `i1` (one value) and `i2..i33` (one value per player, indexed by the component's rank), is kept with its slot,
its packet's clock and its tag. The gauge fills at one unit over the variant's flag return delay, `CTFOptions_FlagReturnTime` (option hash
`0x2b16b76f`, in seconds, a number): on 41 films the median increment of a gauge is 0.9985 to 1.0010 times the reciprocal of the delay on 37,
the lower quartile on the four others, which are 60 s games with many defenders. All 46 measured films of the flag modes write the option (15 s
on 2, 24 s on 29, 40 s on 10, 60 s on 5) and none of the 33 of other modes. The anchoring pass (`ScanManagedProperties`, which finds the records of the same archetype bit by bit in the delta packets) reads the same
property too: on 1,039 dropped flags of 30 films, the walk alone lacks the series of 37 that the pass reads (11 of them returns to base
at 0.95 or more) and has 35 that the pass lacks, so the publication takes, for each dropped interval, the walk's series when it has
one and the pass's otherwise. The values of a zone's state (owner, capture gauge) are read by the pass and not by the walk: on 9
Strongholds films the pass finds all of the 335 captures that the statborg names, within three frames, the walk 333.

**Status.** Read: record type, id, mask, and the components with a ported reader, in NEW and DELTA records. Recovered: the
archetype of a slot with no NEW in view, from the bindings above; records anchored by header patterns where a body cannot be
read: biped records, world-object tracks and creations (bit passes after the walk), scoreboard records, and kills of a view A that
did not close. Not read: a component with no reader (the traversal stops at its start with zero bits consumed and the rest of the
view is an opaque tail); the 32 bits of a DEL; the creation records of entities born between two keyframes (bound by anticipation
instead). An entity present at the start of a chunk but absent from its keyframe table, and designated by an update before any
NEW, is a measured gap.

*Sources: `internal/grammar/frame_records.go`, `canal_des_proprietes_gerees_trames.go`, `unit_control.go`, `unit_control_scan.go`, `entete.go`, `traverse.go`, `ecrivain_invariants.go`, `world.go`, `frame_infer.go`,
`canal_des_horloges.go`, `capture.go`, `default_state_arch.go`, `doc.go`, `testdata/ecs_table.tsv`, `testdata/ecs_table.README.md`; libfilmshell `payload.rs`, `events.rs`.*

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

**Clock and medals.** An event time counts from the film's first packet: the zero of the highlights clock sits within a tick (16.7 ms) of it. On the 11 films libfilmshell reads, its zero, placed by pairing each death record with its kill event, falls between -11.3 and +2.4 ms of the first packet. A medal is identified by its two bytes: the medal type byte `[59]` is the game's medal index (unique among the 157 of the game's table), the hint byte `[47]` its sorting weight; per player and per medal, this decoder and libfilmshell count the same on those 11 films (351 keys, names included). The replay document places a medal on its frame by `(ms - originMs) / frameIntervalMs`, with no death-feed offset. The document also declares a `medalLabels` table (name, description, image and rarity per medal, keyed by the medal's English name): like `vehicleLabels` it is a layer the host fills when it serves the document, never written by the decoder.

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
(`internal/facts/fallback/`) holds about 130 entries across all layers (127 declarations at the time of writing); each fallback
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
| Game variant | engine type, playback flags, match information by path | the same message; the two agree on every compared field of the films both read (`NOTICE.md`); the respawn delay, the two penalties, the friendly-fire flag and the time limit agree on 1,345 films of 1,345 |
| Match clock | every read of the clock component, 1 per second | a change of limit or state only |
