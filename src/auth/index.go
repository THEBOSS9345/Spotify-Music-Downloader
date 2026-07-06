package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"music-downloader/src/domain"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type SpotifyAuthServer struct {
	authenticator *spotifyauth.Authenticator
	state         string
	tokenFile     string
	client        *spotify.Client
	httpClient    *http.Client
	OnAuth        func(client *spotify.Client, httpClient *http.Client, user domain.User)
}

func randomState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type SavedToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       int64  `json:"expiry"`
}

func NewSpotifyAuthServer(clientID, clientSecret, redirectURI string) *SpotifyAuthServer {
	authenticator := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(redirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserReadPrivate,
			spotifyauth.ScopeUserReadEmail,
			spotifyauth.ScopePlaylistReadPrivate,
			spotifyauth.ScopePlaylistReadCollaborative,
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopeUserReadRecentlyPlayed,
			spotifyauth.ScopeUserTopRead,
			spotifyauth.ScopeUserFollowRead,
		),
	)

	return &SpotifyAuthServer{
		authenticator: authenticator,
		state:         randomState(),
		tokenFile:     "spotify_token.json",
	}
}

func (s *SpotifyAuthServer) AuthURL() string {
	return s.authenticator.AuthURL(s.state)
}

func (s *SpotifyAuthServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/auth", s.callbackHandler)
}

func (s *SpotifyAuthServer) callbackHandler(w http.ResponseWriter, r *http.Request) {
	tok, err := s.authenticator.Token(r.Context(), s.state, r)
	if err != nil {
		http.Error(w, "Auth failed: "+err.Error(), http.StatusForbidden)
		return
	}

	s.saveToken(tok)

	httpClient := s.authenticator.Client(r.Context(), tok)
	client := spotify.New(httpClient)
	s.client = client
	s.httpClient = httpClient

	user, err := client.CurrentUser(context.Background())
	if err == nil {
		domainUser := domain.User{
			ID:          user.ID,
			DisplayName: user.DisplayName,
			Email:       user.Email,
		}
		if len(user.Images) > 0 {
			domainUser.AvatarURL = user.Images[0].URL
		}
		if s.OnAuth != nil {
			s.OnAuth(client, httpClient, domainUser)
		}
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *SpotifyAuthServer) saveToken(tok *oauth2.Token) error {
	data := SavedToken{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		TokenType:    tok.TokenType,
		Expiry:       tok.Expiry.Unix(),
	}

	file, err := os.Create(s.tokenFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

func (s *SpotifyAuthServer) LoadToken(ctx context.Context) (*spotify.Client, *http.Client, *domain.User, error) {
	file, err := os.Open(s.tokenFile)
	if err != nil {
		return nil, nil, nil, err
	}
	defer file.Close()

	var saved SavedToken
	if err := json.NewDecoder(file).Decode(&saved); err != nil {
		return nil, nil, nil, err
	}

	tok := &oauth2.Token{
		AccessToken:  saved.AccessToken,
		RefreshToken: saved.RefreshToken,
		TokenType:    saved.TokenType,
		Expiry:       time.Unix(saved.Expiry, 0),
	}

	httpClient := s.authenticator.Client(ctx, tok)
	client := spotify.New(httpClient)
	s.client = client
	s.httpClient = httpClient

	user, err := client.CurrentUser(ctx)
	if err != nil {
		return client, httpClient, nil, nil
	}

	domainUser := domain.User{
		ID:          user.ID,
		DisplayName: user.DisplayName,
		Email:       user.Email,
	}
	if len(user.Images) > 0 {
		domainUser.AvatarURL = user.Images[0].URL
	}

	return client, httpClient, &domainUser, nil
}

func (s *SpotifyAuthServer) Logout() {
	os.Remove(s.tokenFile)
	s.client = nil
}

func (s *SpotifyAuthServer) GetLoginURL() string {
	return s.authenticator.AuthURL(s.state)
}
