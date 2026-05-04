# Countdown Music Per-Time Design

## Overview

Extend the countdown music feature from a single global track to a global playlist where each track can be toggled into a random pool, and each `countdown_time` independently selects either random-from-pool or a specific track.

---

## Data Model

### New struct: `MusicTrack`

```go
type MusicTrack struct {
    Path     string `json:"path"`
    InRandom bool   `json:"in_random"` // whether this track is included in the random pool
}
```

### New struct: `CountdownTimeMusic`

```go
type CountdownTimeMusic struct {
    Time  string `json:"time"`  // matches a value in Settings.CountdownTimes
    Mode  string `json:"mode"`  // "random" | "index"
    Index int    `json:"index"` // used when Mode == "index"; 0-based index into CountdownMusics
}
```

### `Settings` changes

Remove:
```go
CountdownMusic string `json:"countdown_music"`
```

Add:
```go
CountdownMusics        []MusicTrack         `json:"countdown_musics"`
CountdownTimeMusicMap  []CountdownTimeMusic `json:"countdown_time_music_map"`
```

`CountdownVolume float64` is unchanged.

### Migration (in `LoadConfig`)

If the old `countdown_music` field is non-empty and `countdown_musics` is empty, migrate:
- Append `MusicTrack{Path: oldValue, InRandom: true}` to `CountdownMusics`
- Clear `countdown_music`

---

## Backend (`app.go`)

| Function | Signature | Description |
|---|---|---|
| `SelectCountdownMusics` | `() ([]string, error)` | Opens multi-file dialog filtered to `*.mp3`; returns selected paths. Frontend appends to its local list. |
| `GetActiveCountdownMusicData` | `(time string) (string, error)` | Looks up the `CountdownTimeMusic` for the given time string. If mode is `"random"`, picks a random track from tracks where `InRandom == true`. If mode is `"index"`, picks `CountdownMusics[Index]`. Reads file, validates MP3 magic bytes, returns base64 data URL. |
| `GetCountdownMusicData` | `(index int) (string, error)` | Reads `CountdownMusics[index]` by path, validates, returns base64 data URL. Used by settings page preview. |
| `ValidateRandomPool` | `() bool` | Returns true if at least one track has `InRandom == true` and the file exists. |

`isMP3()` is unchanged.

### Random selection logic in `GetActiveCountdownMusicData`

1. Filter `CountdownMusics` to entries where `InRandom == true`.
2. If the filtered list is empty, return an error (`"隨機清單為空"`).
3. Pick a random index using `rand.Intn(len(pool))` (seeded at startup or via `rand.New`).
4. Read and return that track's data.

### Fallback for unset `CountdownTimeMusic`

If a `countdown_time` has no entry in `CountdownTimeMusicMap`, treat it as mode `"random"`.

---

## Frontend

### Settings Page (`SettingsPage.svelte`)

**Global music list section** (new):
- "新增音樂" button → calls `SelectCountdownMusics()`, appends returned paths to local list as `MusicTrack` objects with `InRandom: true` by default.
- Each track row: filename (basename only) | "加入隨機" toggle checkbox | "▶ 試聽" button | "✕ 刪除" button.
- "▶ 試聽" calls `GetCountdownMusicData(index)` and plays preview audio (same pause/play logic as current single-track preview).

**Countdown times section** (extend existing list):
- Each `countdown_time` row gains a mode selector on the right:
  - Radio or toggle: 「隨機」| 「指定」
  - When "指定": show a dropdown listing all tracks from the global list (display basename).
  - When "隨機": no extra controls shown.

**Save validation**:
- Before calling `SaveSettings`, if any `countdown_time` has mode `"random"`, call `ValidateRandomPool()`.
- If it returns false, show an error message (e.g., "請至少將一首音樂加入隨機清單") and block save.

### `CountdownOverlay.svelte`

Single change: replace `GetCountdownMusicData()` with `GetActiveCountdownMusicData(triggerTime)` where `triggerTime` is the time string passed into the overlay (already available from the trigger logic in `App.svelte`).

---

## Data Flow

```
[Settings Page]
  User adds tracks → local MusicTrack[]
  User sets InRandom toggles
  User sets per-time mode/index
  Save → SaveSettings()
        → data/config.json

[CountdownOverlay triggered at time T]
  GetActiveCountdownMusicData(T)
    → lookup CountdownTimeMusicMap for T
    → mode == random: pick from InRandom pool
    → mode == index:  pick CountdownMusics[Index]
    → read file, validate MP3, return base64
  Audio element plays
```

---

## Constraints & Validation

- A `countdown_time` with mode `"random"` requires at least one `MusicTrack` with `InRandom: true` — enforced at save time in the frontend.
- If a track file is missing at playback time, `GetActiveCountdownMusicData` returns an error; `CountdownOverlay` silently skips music (same current behavior for missing file).
- `CountdownTimeMusicMap` entries whose `Time` value no longer exists in `CountdownTimes` are ignored and cleaned up on next save.
