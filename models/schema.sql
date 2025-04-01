CREATE TABLE tracks (
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  spotify_id varchar(64) PRIMARY KEY,
  name varchar(100) NOT NULL,
  duration_ms INT
);

CREATE TABLE playlists (
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  spotify_id varchar(64) PRIMARY KEY,
  name varchar(100) NOT NULL
);

CREATE TYPE track_play_context AS ENUM ('artist', 'playlist', 'album', 'show');

CREATE TABLE track_plays (
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  played_at TIMESTAMP PRIMARY KEY,
  track_id varchar(100) references tracks(spotify_id) NOT NULL,
  context track_play_context,
  playlist_id varchar(100) references playlists(spotify_id)
);

CREATE TABLE likes (
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE TABLE dislikes (
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
