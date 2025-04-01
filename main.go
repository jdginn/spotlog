package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/zmb3/spotify/v2"
	"github.com/zmb3/spotify/v2/auth"

	"github.com/jdginn/spotlog/models"
)

var (
	auth  = spotifyauth.New(spotifyauth.WithRedirectURL(os.Getenv("REDIRECT_URI")), spotifyauth.WithScopes(spotifyauth.ScopeUserReadRecentlyPlayed))
	ch    = make(chan *spotify.Client)
	state = "abc123"
)

func main() {
	ctx := context.Background()
	// Open connection to database
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)
	db := models.New(conn)

	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/like", func(w http.ResponseWriter, r *http.Request) {
		err := db.CreateLike(ctx)
		if err != nil {
			log.Println(fmt.Errorf("Error registering like: %w", err))
		} else {
			w.Write([]byte("Like registered"))
		}
	})
	http.HandleFunc("/dislike", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Dislike registered"))
		// err := db.CreateDislike(ctx)
		// if err != nil {
		// 	log.Println(fmt.Errorf("Error registering dislike: %w", err))
		// } else {
		// 	w.Write([]byte("Dislike registered"))
		// }
	})
	http.HandleFunc("/console", serveConsole)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go func() {
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal(err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("You are logged in as:", user.ID)

	for {
		log.Println("Pulling most recently played...")
		if err := updateRecentlyPlayed(ctx, client, db); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Minute * 15)
	}

}

func serveConsole(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/console.html")
	if err != nil {
		http.Error(w, "Couldn't load template", http.StatusInternalServerError)
		log.Println("Error loading template:", err)
		return
	}
	tmpl.Execute(w, nil)
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}

	// use the token to get an authenticated client
	client := spotify.New(auth.Client(r.Context(), tok))
	fmt.Fprintf(w, "Login Completed!")
	ch <- client
}

func updateRecentlyPlayed(ctx context.Context, client *spotify.Client, db *models.Queries) error {
	recentlyPlayed, err := client.PlayerRecentlyPlayedOpt(ctx, &spotify.RecentlyPlayedOptions{Limit: 50, BeforeEpochMs: time.Now().UnixMilli()})
	if err != nil {
		return fmt.Errorf("Error pulling most recently played tracks: %w")
	}
	for _, apiTrack := range recentlyPlayed {
		err := db.CreateTrack(ctx, models.CreateTrackParams{
			SpotifyID:  apiTrack.Track.ID.String(),
			Name:       apiTrack.Track.Name,
			DurationMs: pgtype.Int4{Int32: int32(apiTrack.Track.Duration), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("Error creating Track in database: %w", err)
		}
		playlistContext := new(models.NullTrackPlayContext)
		playlistContext.Scan(apiTrack.PlaybackContext.Type)
		playlistIDs, err := db.ListPlaylistsByID(ctx)

		if err != nil {
			return fmt.Errorf("Error looking up playlist names: %w", err)
		}
		switch playlistContext.TrackPlayContext {
		case models.TrackPlayContextPlaylist:

			playlistID := spotify.ID(apiTrack.PlaybackContext.URI[len("spotify:playlist:"):])

			if !slices.Contains(playlistIDs, playlistID.String()) {
				apiPlaylist, err := client.GetPlaylist(ctx, spotify.ID(playlistID))
				if err != nil {
					err = db.CreateTrackPlay(ctx, models.CreateTrackPlayParams{
						PlayedAt: pgtype.Timestamp{Time: apiTrack.PlayedAt.UTC(), Valid: true},
						TrackID:  apiTrack.Track.ID.String(),
						Context:  *playlistContext,
					})
				} else {
					err = db.CreatePlaylist(ctx, models.CreatePlaylistParams{
						Name:      apiPlaylist.Name,
						SpotifyID: apiPlaylist.ID.String(),
					})
					if err != nil {
						return fmt.Errorf("Error creating Playlist in database: %w", err)
					}
				}
			}
			err = db.CreateTrackPlay(ctx, models.CreateTrackPlayParams{
				PlayedAt:   pgtype.Timestamp{Time: apiTrack.PlayedAt.UTC(), Valid: true},
				TrackID:    apiTrack.Track.ID.String(),
				Context:    *playlistContext,
				PlaylistID: pgtype.Text{String: playlistID.String(), Valid: true},
			})
			if err != nil {
				return fmt.Errorf("Error creating TrackPlay in database: %w", err)
			}
		case models.TrackPlayContextAlbum:
			err = db.CreateTrackPlay(ctx, models.CreateTrackPlayParams{
				PlayedAt: pgtype.Timestamp{Time: apiTrack.PlayedAt.UTC(), Valid: true},
				TrackID:  apiTrack.Track.ID.String(),
				Context:  *playlistContext,
			})
			if err != nil {
				return fmt.Errorf("Error creating TrackPlay in database: %w", err)
			}
		default:
			err = db.CreateTrackPlay(ctx, models.CreateTrackPlayParams{
				PlayedAt: pgtype.Timestamp{Time: apiTrack.PlayedAt.UTC(), Valid: true},
				TrackID:  apiTrack.Track.ID.String(),
			})
		}
	}
	return nil
}
