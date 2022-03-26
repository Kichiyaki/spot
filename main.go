package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	clientID := flag.String("clientID", "", "")
	clientSecret := flag.String("clientSecret", "", "")
	artistsStr := flag.String("artists", "", "")
	market := flag.String("market", spotify.CountryUSA, "")
	destDir := flag.String("dest", "./download", "where to download songs")
	flag.Parse()

	if *clientID == "" {
		log.Fatalln("the clientID flag cannot be blank")
	}

	if *clientSecret == "" {
		log.Fatalln("the clientSecret flag cannot be blank")
	}

	client, err := newClient(*clientID, *clientSecret)
	if err != nil {
		log.Fatalln("newClient:", err)
	}

	artists := strings.Split(*artistsStr, ",")

	for _, q := range artists {
		log.Printf("----------%s----------", q)

		artist, err := searchArtist(client, q)
		if err != nil {
			log.Printf("artist '%s' not found: %v", q, err)
			continue
		}

		log.Printf("Looking for albums...")
		albums, err := getAlbums(client, artist.ID, *market)
		if err != nil {
			log.Printf("albums not found: %v", err)
			continue
		}
		log.Printf("found %d albums", len(albums))

		for _, album := range albums {
			log.Printf("Downloading %s", album.Name)
			if err := downloadAlbum(artist.Name, album, *destDir); err != nil {
				log.Printf("couldn't download album '%s': %v", err)
			}
		}

		log.Printf("----------%s----------", strings.Repeat("-", len(q)))
	}
}

func newClient(clientID, clientSecret string) (*spotify.Client, error) {
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotifyauth.TokenURL,
	}

	token, err := config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("config.Token: %w", err)
	}

	return spotify.New(spotifyauth.New().Client(ctx, token)), nil
}

func searchArtist(client *spotify.Client, q string) (spotify.FullArtist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := client.Search(ctx, q, spotify.SearchTypeArtist)
	if err != nil {
		return spotify.FullArtist{}, err
	}

	for _, artist := range result.Artists.Artists {
		if artist.Name == q {
			return artist, nil
		}
	}

	return spotify.FullArtist{}, errors.New("artist not found")
}

func getAlbums(client *spotify.Client, artistID spotify.ID, market string) ([]spotify.SimpleAlbum, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := client.GetArtistAlbums(ctx, artistID, []spotify.AlbumType{spotify.AlbumTypeAlbum}, spotify.Market(market))
	if err != nil {
		return nil, err
	}

	return result.Albums, nil
}

func downloadAlbum(artistName string, album spotify.SimpleAlbum, destDir string) error {
	url, ok := album.ExternalURLs["spotify"]
	if !ok {
		return errors.New("album.ExternalURLs: missing key 'spotify'")
	}

	dir := path.Join(destDir, slug.Make(artistName), slug.Make(album.Name))

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("os.MkdirAll: %w", err)
	}

	cmd := exec.Command("spotdl", url)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run: %w", err)
	}

	return nil
}
