# Random parity

How far Elegy matches the original's random outcomes in the parity vectors,
and what the harness needs to replay the original's random stream.
Lane D of [SPRINT.md](SPRINT.md). Numbers are from `go test -run
'TestRandomStreams$' -v ./internal/vectors` on the vectors in `engine/testdata/vectors`
(stars-elegy d4eafd9).

## What the vectors record about the stream

- A several-stream vector labels each stream by its DOSBox cycles value
  (`"stream": "cycles 20000"`). An expectation that differed between
  streams appears once per stream with that label; one that agreed appears
  once, unlabelled.
- A single-stream vector records nothing about its stream: no cycles value,
  no tick, no generator state.
- No vector records a startup tick or a generator state.
- stars-elegy `docs/ORACLE.md` "Pinned battle RNG" (d4eafd9) maps cycles
  values to startup ticks (MEASURED by replaying KX-004 events and
  TK-307 scrap-tech draws). `oraclerng` holds 39 of them: the values the
  table lists, keeping only the two ends of its range "1985–1955 → 2032".
  That a cycles value gives the same tick for every start is an
  ASSUMPTION: the table pools runs from different starts.

## Elegy's generator against the specification

stars-elegy specifies `rand(n)` only as "a uniform draw in 0..n−1"
(`KERNEL.md` "Conventions", d4eafd9), plus the order of the year's draws
(`KERNEL.md` "Random draws") and each rule's own draws. The generator and
its seeding are not public rules, so no Elegy generator can match the
original's yet. Elegy's games use `newgame.NewRand` (ELEGY CHOICE), and the
harness uses `math/rand`; both are uniform, which is all the spec asks.

The draw order is checkable without the generator.
`TestConfirmedYearDrawOrderKX004` records the bounds of every draw in each
KX-004 start: the shuffle `Random(2)`, `Random(1)`, mining `Random(100)`s,
then comet `rand(20)`, climate change `rand(20)` and new minerals
`rand(15 − size)`. The events begin at draw 4 in S1 to S9, as KERNEL.md
measured.

## The seam

`internal/vectors/streams.go`, in the vector loader package (its tests,
`TestRandomStreams` among them, are `internal/vectors/streams_test.go`):

- `pvRun` says how `runVectorWith` generates a vector: Elegy's seed variant,
  which stream's expectations to check, and `replay`, a random source per
  year. `runVector(v, k)` is `runVectorWith(v, pvRun{seed: k})`, unchanged
  behaviour.
- `pvStart(v, stream, year)` returns where that year's stream started: the
  vector's `random_start` entry, else the ORACLE.md tick for the stream's
  cycles value, else nothing recorded.
- `pvReplay(v, stream, seeder)` returns the per-year sources, or why there
  are none ("no start recorded", "no generator").
- `pvOriginal` (test code) is the `oraclerng.Seeder` for the original's generator. It
  is nil until stars-elegy publishes the generator; the clean room
  implements it only from that text. Setting it turns on exact replay in
  `TestRandomStreams`, which then fails on any replayed mismatch.

Under replay every year draws from its own start (the oracle runs each year
in a fresh process; ORACLE.md: consecutive years of one start begin from the
same stream), and samples compare exactly.

### Proposed vector field: `random_start`

Not yet in FORMAT.md; for the vectors owner.

```jsonc
"random_start": [
  {"stream": "cycles 20000",  // a stream label; "" in a single-stream vector
   "year": 1,                 // generated year, 1 is the first
   "tick": 164,               // startup tick in ms that seeded the stream; optional
   "state": null}             // generator state before the year's first draw,
                              // in the published generator's terms; optional
]
```

No entry for a stream's year, or an entry with neither `tick` nor `state`,
means no start recorded. An entry wins over the ORACLE.md table.

## Parity per corpus

Counts are expectations. "Other-stream" is one stream's value of an
expectation that differed between streams; the main harness skips them.
"Samples" are unlabelled `sample: true` or `kind: "sample"` expectations.
"Compared" means the harness compares that quantity at all; the rest wait
on Elegy modelling it, whatever the stream.

The three categories are kept apart:

- **Deterministic**: exact expectations of the cases that also hold a
  random outcome, with the reference seed.
- **Results without exact replay**: compared random outcomes under Elegy's
  own 8 seeds. A match shows Elegy reached the original's value with its
  own draws; it is not exact parity.
- **Exact random parity**: compared random outcomes replayed from the
  oracle's stream. 0 everywhere: the generator is not public.

| Corpus | Other-stream compared/all | Samples compared/all | Deterministic matched/compared | Results: reference seed, another seed, never | Exact | Unlocked by the generator alone (ORACLE.md tick) | Also needs a recorded start |
|---|---|---|---|---|---|---|---|
| cb | 1751/2462 | 1008/1057 | 190/190 | 2456, 256, 47 | 0 | 1969 | 790 |
| cb7 | 97/217 | 0/0 | 41/41 | 89, 6, 2 | 0 | 40 | 57 |
| cs | 0/0 | 0/11 | 0/0 | — | 0 | 0 | 0 |
| fm2 | 0/0 | 0/4 | 0/0 | — | 0 | 0 | 0 |
| gt | 0/0 | 33/37 | 42/42 | 30, 3, 0 | 0 | 0 | 33 |
| kx004 | 836/1380 | 0/0 | 158/158 | 799, 0, 37 | 0 | 302 | 534 |
| kx005 | 105/112 | 14/16 | 60/60 | 77, 31, 11 | 0 | 105 | 14 |
| mf | 237/241 | 243/243 | 26/27 | 67, 69, 344 | 0 | 166 | 314 |
| ob | 48/63 | 13/36 | 18/18 | 25, 25, 11 | 0 | 26 | 35 |
| pq | 0/0 | 0/3 | 0/0 | — | 0 | 0 | 0 |
| rw | 0/0 | 0/46 | 0/0 | — | 0 | 0 | 0 |
| sl | 4/8 | 0/0 | 33/33 | 4, 0, 0 | 0 | 4 | 0 |
| tk2 | 0/8 | 0/0 | 0/0 | — | 0 | 0 | 0 |
| tk3 | 182/204 | 0/0 | 34/34 | 124, 47, 11 | 0 | 182 | 0 |
| ug | 0/0 | 0/120 | 0/0 | — | 0 | 0 | 0 |
| wt | 0/48 | 20/40 | 0/0 | 18, 2, 0 | 0 | 1 | 19 |
| **all** | **3260/4743** | **1331/1613** | **602/603** | **3689, 439, 463** | **0** | **2795** | **1796** |

Reading it:

- 4,591 random outcomes are compared (3,260 other-stream, 1,331 samples).
  All of them become exactly checkable once the generator is public and
  each stream's start is known: 2,795 through the ORACLE.md ticks already,
  1,796 only with a recorded start (single-stream vectors, and cycles
  values with no measured tick, such as 25000 and 22000 in cb).
- 1,483 other-stream and 282 sample expectations stay uncompared with any
  stream, because the harness does not compare those quantities (for
  example new-game planet positions in ug and rw).
- The one deterministic mismatch is in mf: MF-02, whose reference seed
  fails in `TestParityVectors` too (SPRINT.md "Known parity failures").
- SPRINT.md's 4,298 other-stream and 327 sample figures come from the main
  harness's skip tally, which counts only inside cases with at least one
  compared expectation, and counts a sample only when it missed. The table
  above counts every expectation.

## Battle records

`TestBattleRecords` (`engine/battle_parity_test.go`) replays every recorded
battle at its year's battle phase, with the record's token order and moves,
and compares each hit. During firing Elegy's only draws are torpedo
accuracy rolls. The harness answers them from the record instead of
drawing, with a test-file `Rand` (`pvAnswer`): each group of rolls (one
salvo's torpedoes at one target) gets a number of hits, and a search finds
numbers with which every hit matches. It backtracks, because two numbers
can give a group the same records and differ later.

The hits are tallied in two kinds:

- **Compared**: hits recorded before the battle's first answered roll. They
  depend on no random outcome.
- **Conditioned**: hits recorded after one. They show that Elegy's
  mechanics reproduce the record given some outcome of the rolls. That is
  not random parity: nothing shows that Elegy's or the original's draws
  give those outcomes. A conditioned mismatch is a mechanics finding.

Any other draw during firing still stops the replay; none was reached.

| Corpus | Records | Replayed whole | Groups answered | Compared matched | Conditioned matched | Hits recorded |
|---|---|---|---|---|---|---|
| cb | 400 | 387 (13 have no hits) | 286 | 8423/8423 | 403/403 | 8826 |
| cb7 | 44 | 44 | 87 | 4106/4106 | 129/129 | 4235 |
| cs | 18 | 18 | 334 | 4/4 | 370/370 | 374 |
| rg | 8 | 8 | 15 | 14/14 | 25/25 | 39 |

Answering the rolls turned up one mechanics difference, now fixed: on a
shielded target the misses' shield damage is a hit record of its own,
before the salvo's hit record (stars-elegy COMBAT.md "Torpedoes and
missiles", CONFIRMED CB-049). Elegy had merged the two.
