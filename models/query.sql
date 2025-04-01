-- name: CreatePlaylist :exec
INSERT INTO playlists (
  spotify_id, name 
) VALUES (
  $1, $2
) ON CONFLICT DO NOTHING;

-- name: GetPlaylistByName :one
SELECT * FROM playlists
WHERE name = $1 LIMIT 1;

-- name: GetPlaylistByID :one
SELECT * FROM playlists
WHERE spotify_id = $1 LIMIT 1;

-- name: GetPlaylists :many
SELECT * FROM playlists;

-- name: ListPlaylistsByID :many
SELECT spotify_id FROM playlists;

-- name: CreateTrack :exec
INSERT INTO tracks (
  spotify_id, name, duration_ms
) VALUES (
  $1, $2, $3
) ON CONFLICT DO NOTHING;

-- name: GetTrackByName :one
SELECT * FROM tracks
WHERE name = $1 LIMIT 1;

-- name: GetTrackByID :one
SELECT * FROM tracks
WHERE spotify_id = $1 LIMIT 1;

-- name: CreateTrackPlay :exec
INSERT INTO track_plays (
  played_at, track_id, context, playlist_id
) VALUES (
  $1, $2, $3, $4
) ON CONFLICT DO NOTHING;

--TODO:
-- TODO name: GetMostRecentlyPlayedTrack
-- TODO name: GetMostRecentPlayedAtTimestamp
-- TODO name: GetPlayCountForTrackByName
-- TODO name: GetMostRedentPlayOfTrackByName

-- name: CreateLike :exec
INSERT INTO likes DEFAULT VALUES;

-- name: CreateDislike :exec
INSERT INTO dislikes DEFAULT VALUES;

-- TODO:
-- TODO name: GetLikesForTrackByName
-- TODO name: GetLikesForTrackByID
