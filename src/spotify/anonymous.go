package spotify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"music-downloader/src/domain"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const spotifyTOTPSecret = "GM3TMMJTGYZTQNZVGM4DINJZHA4TGOBYGMZTCMRTGEYDSMJRHE4TEOBUG4YTCMRUGQ4DQOJUGQYTAMRRGA2TCMJSHE3TCMBY"

const fetchPlaylistHash = "bb67e0af06e8d6f52b531f97468ee4acd44cd0f82b988e15c2ea47b1148efc77"

type anonClient struct {
	mu            sync.Mutex
	httpClient    *http.Client
	accessToken   string
	clientToken   string
	clientID      string
	clientVersion string
	deviceID      string
	cookies       map[string]string
	expiresAt     time.Time
}

func newAnonClient() *anonClient {
	return &anonClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cookies:    make(map[string]string),
	}
}

func generateSpotifyTOTP(now time.Time) (string, error) {
	key, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/secret?secret=%s", spotifyTOTPSecret))
	if err != nil {
		return "", err
	}
	code, err := totp.GenerateCode(key.Secret(), now)
	if err != nil {
		return "", err
	}
	return code, nil
}

func (c *anonClient) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	valid := c.accessToken != "" && c.clientToken != "" && time.Now().Before(c.expiresAt.Add(-30*time.Second))
	c.mu.Unlock()
	if valid {
		return nil
	}

	if err := c.requestAccessToken(ctx); err != nil {
		return err
	}
	if err := c.getSessionInfo(ctx); err != nil {
		return err
	}
	return c.getClientToken(ctx)
}

func (c *anonClient) invalidate() {
	c.mu.Lock()
	c.accessToken = ""
	c.clientToken = ""
	c.expiresAt = time.Time{}
	c.mu.Unlock()
}

func (c *anonClient) requestAccessToken(ctx context.Context) error {
	code, err := generateSpotifyTOTP(time.Now())
	if err != nil {
		return fmt.Errorf("totp generate: %w", err)
	}

	offsets := []time.Duration{0, -30 * time.Second, 30 * time.Second}
	for _, offset := range offsets {
		if offset != 0 {
			code, err = generateSpotifyTOTP(time.Now().Add(offset))
			if err != nil {
				return fmt.Errorf("totp generate: %w", err)
			}
		}

		req, _ := http.NewRequestWithContext(ctx, "GET", "https://open.spotify.com/api/token", nil)
		q := req.URL.Query()
		q.Add("reason", "init")
		q.Add("productType", "web-player")
		q.Add("totp", code)
		q.Add("totpVer", "61")
		q.Add("totpServer", code)
		req.URL.RawQuery = q.Encode()

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("token request: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var data struct {
				AccessToken string `json:"accessToken"`
				ClientID    string `json:"clientId"`
				ExpiresAtMs int64  `json:"accessTokenExpirationTimestampMs"`
			}
			if err := json.Unmarshal(body, &data); err != nil {
				return fmt.Errorf("parse token: %w", err)
			}

			c.mu.Lock()
			c.accessToken = data.AccessToken
			c.clientID = data.ClientID
			c.expiresAt = time.UnixMilli(data.ExpiresAtMs)
			c.mu.Unlock()
			return nil
		}
		resp.Body.Close()
	}

	return fmt.Errorf("token request failed after retries")
}

func (c *anonClient) getSessionInfo(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://open.spotify.com", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("session request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	re := regexp.MustCompile(`<script id="appServerConfig" type="text/plain">([^<]+)</script>`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		decoded, err := base64.StdEncoding.DecodeString(matches[1])
		if err == nil {
			var cfg struct {
				ClientVersion string `json:"clientVersion"`
			}
			if json.Unmarshal(decoded, &cfg) == nil && cfg.ClientVersion != "" {
				c.clientVersion = cfg.ClientVersion
			}
		}
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sp_t" && c.deviceID == "" {
			c.deviceID = cookie.Value
		}
		c.cookies[cookie.Name] = cookie.Value
	}

	if c.clientVersion == "" {
		c.clientVersion = "1800000000"
	}
	if c.deviceID == "" {
		c.deviceID = fmt.Sprintf("dev-%d", rand.Int63())
	}

	return nil
}

func (c *anonClient) getClientToken(ctx context.Context) error {
	c.mu.Lock()
	clientID := c.clientID
	deviceID := c.deviceID
	clientVersion := c.clientVersion
	c.mu.Unlock()

	if clientID == "" || deviceID == "" {
		if err := c.getSessionInfo(ctx); err != nil {
			return err
		}
		c.mu.Lock()
		clientID = c.clientID
		deviceID = c.deviceID
		clientVersion = c.clientVersion
		c.mu.Unlock()
	}

	payload := map[string]interface{}{
		"client_data": map[string]interface{}{
			"client_version": clientVersion,
			"client_id":      clientID,
			"js_sdk_data": map[string]interface{}{
				"device_brand": "unknown",
				"device_model": "unknown",
				"os":           "windows",
				"os_version":   "NT 10.0",
				"device_id":    deviceID,
				"device_type":  "computer",
			},
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://clienttoken.spotify.com/v1/clienttoken", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK && len(body) > 0 {
		var data struct {
			ResponseType string `json:"response_type"`
			GrantedToken struct {
				Token string `json:"token"`
			} `json:"granted_token"`
		}
		if err := json.Unmarshal(body, &data); err == nil && data.GrantedToken.Token != "" {
			c.mu.Lock()
			c.clientToken = data.GrantedToken.Token
			c.mu.Unlock()
			return nil
		}
	}

	// Fallback: try the older endpoint
	req2, _ := http.NewRequestWithContext(ctx, "GET", "https://open.spotify.com/get_access_token?reason=transport&productType=web-player", nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if c.accessToken != "" {
		req2.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return fmt.Errorf("client token request: %w", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)
	if resp2.StatusCode != http.StatusOK {
		return fmt.Errorf("client token request: HTTP %d: %s", resp2.StatusCode, string(body2))
	}

	var data2 struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(body2, &data2); err != nil {
		return fmt.Errorf("parse client token: %w", err)
	}
	if data2.AccessToken == "" {
		return fmt.Errorf("empty client token from fallback")
	}

	c.mu.Lock()
	c.clientToken = data2.AccessToken
	c.mu.Unlock()
	return nil
}

func (c *anonClient) queryGraphQL(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	if err := c.ensureToken(ctx); err != nil {
		return nil, err
	}

	jsonData, _ := json.Marshal(payload)

	do := func() (int, []byte, error) {
		req, _ := http.NewRequestWithContext(ctx, "POST", "https://api-partner.spotify.com/pathfinder/v2/query", bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
		req.Header.Set("Client-Token", c.clientToken)
		req.Header.Set("Spotify-App-Version", c.clientVersion)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return 0, nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, body, nil
	}

	statusCode, body, err := do()
	if err != nil {
		return nil, err
	}

	if (statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden) && strings.Contains(string(body), "ACCESS_TOKEN_INVALID") {
		c.invalidate()
		if err := c.ensureToken(ctx); err != nil {
			return nil, err
		}
		statusCode, body, err = do()
		if err != nil {
			return nil, err
		}
	}

	if statusCode != http.StatusOK {
		errText := string(body)
		if len(errText) > 200 {
			errText = errText[:200]
		}
		return nil, fmt.Errorf("graphql: HTTP %d: %s", statusCode, errText)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse graphql: %w", err)
	}
	return result, nil
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getMap(m map[string]interface{}, key string) map[string]interface{} {
	if val, ok := m[key].(map[string]interface{}); ok {
		return val
	}
	return nil
}

func getSlice(m map[string]interface{}, key string) []interface{} {
	if val, ok := m[key].([]interface{}); ok {
		return val
	}
	return nil
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	if val, ok := m[key].(int); ok {
		return float64(val)
	}
	return 0
}

func extractGraphQLArtists(artistsData map[string]interface{}) []string {
	items := getSlice(artistsData, "items")
	if items == nil {
		return nil
	}
	var names []string
	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		profile := getMap(itemMap, "profile")
		name := getString(profile, "name")
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

func extractGraphQLAlbumArt(coverArt map[string]interface{}) string {
	if coverArt == nil {
		return ""
	}
	sources := getSlice(coverArt, "sources")
	if sources == nil {
		if square := getMap(coverArt, "squareCoverImage"); square != nil {
			if img := getMap(square, "image"); img != nil {
				if data := getMap(img, "data"); data != nil {
					sources = getSlice(data, "sources")
				}
			}
		}
	}
	if sources == nil {
		return ""
	}

	var bestURL string
	var bestWidth float64

	for _, s := range sources {
		sMap, ok := s.(map[string]interface{})
		if !ok {
			continue
		}
		url := getString(sMap, "url")
		if url == "" {
			continue
		}
		width := getFloat(sMap, "width")

		if width == 300 || width == 640 {
			return url
		}
		if width > bestWidth {
			bestWidth = width
			bestURL = url
		}
	}
	if bestURL == "" && len(sources) > 0 {
		if s, ok := sources[0].(map[string]interface{}); ok {
			bestURL = getString(s, "url")
		}
	}
	return bestURL
}

func (s *Service) getAnonClient() *anonClient {
	s.anonMu.Lock()
	defer s.anonMu.Unlock()
	if s.anon == nil {
		s.anon = newAnonClient()
	}
	return s.anon
}

func (s *Service) fetchPlaylistAnon(ctx context.Context, playlistID string) (*domain.Playlist, error) {
	payload := map[string]interface{}{
		"variables": map[string]interface{}{
			"uri":                       "spotify:playlist:" + playlistID,
			"offset":                    0,
			"limit":                     1,
			"enableWatchFeedEntrypoint": false,
		},
		"operationName": "fetchPlaylist",
		"extensions": map[string]interface{}{
			"persistedQuery": map[string]interface{}{
				"version":    1,
				"sha256Hash": fetchPlaylistHash,
			},
		},
	}

	anon := s.getAnonClient()
	result, err := anon.queryGraphQL(ctx, payload)
	if err != nil {
		return nil, err
	}

	data := getMap(result, "data")
	plData := getMap(data, "playlistV2")
	if plData == nil {
		return nil, fmt.Errorf("unexpected graphql response: missing playlistV2")
	}

	ownerV2 := getMap(plData, "ownerV2")
	ownerData := getMap(ownerV2, "data")
	ownerName := getString(ownerData, "name")

	content := getMap(plData, "content")
	totalCount := int(getFloat(content, "totalCount"))

	plURI := getString(plData, "uri")
	plID := playlistID
	if parts := strings.Split(plURI, ":"); len(parts) > 0 {
		plID = parts[len(parts)-1]
	}

	cover := ""
	if images := getMap(plData, "images"); images != nil {
		if items := getSlice(images, "items"); items != nil && len(items) > 0 {
			if first, ok := items[0].(map[string]interface{}); ok {
				if sources := getSlice(first, "sources"); sources != nil && len(sources) > 0 {
					if src, ok := sources[0].(map[string]interface{}); ok {
						cover = getString(src, "url")
					}
				}
			}
		}
	}
	if cover == "" {
		if imagesV2 := getMap(plData, "imagesV2"); imagesV2 != nil {
			if sources := getSlice(imagesV2, "sources"); sources != nil && len(sources) > 0 {
				if src, ok := sources[0].(map[string]interface{}); ok {
					cover = getString(src, "url")
				}
			}
		}
	}

	pl := &domain.Playlist{
		ID:          plID,
		Name:        getString(plData, "name"),
		Description: getString(plData, "description"),
		TrackCount:  totalCount,
		Owner:       ownerName,
		OwnerID:     ownerName,
		ImageURL:    cover,
	}
	return pl, nil
}

func (s *Service) fetchTracksAnon(ctx context.Context, playlistID string) ([]domain.Song, error) {
	anon := s.getAnonClient()
	var allSongs []domain.Song
	limit := 100
	offset := 0
	maxRetries := 3

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		payload := map[string]interface{}{
			"variables": map[string]interface{}{
				"uri":                       "spotify:playlist:" + playlistID,
				"offset":                    offset,
				"limit":                     limit,
				"enableWatchFeedEntrypoint": false,
			},
			"operationName": "fetchPlaylist",
			"extensions": map[string]interface{}{
				"persistedQuery": map[string]interface{}{
					"version":    1,
					"sha256Hash": fetchPlaylistHash,
				},
			},
		}

		var items []interface{}
		ok := false

		for attempt := 0; attempt <= maxRetries; attempt++ {
			result, err := anon.queryGraphQL(ctx, payload)
			if err != nil {
				if attempt < maxRetries {
					anon.invalidate()
					delay := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(delay):
					}
					continue
				}
				return nil, fmt.Errorf("graphql query: %w", err)
			}

			data := getMap(result, "data")
			plData := getMap(data, "playlistV2")
			if plData == nil {
				if attempt < maxRetries {
					anon.invalidate()
					time.Sleep(time.Second)
					continue
				}
				return nil, fmt.Errorf("missing playlistV2 in graphql response")
			}

			content := getMap(plData, "content")
			items = getSlice(content, "items")
			if items == nil {
				break
			}
			ok = true
			break
		}

		if !ok {
			return nil, fmt.Errorf("failed to fetch tracks after retries")
		}

		for _, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			trackData := getMap(getMap(itemMap, "itemV2"), "data")
			if trackData == nil {
				continue
			}

			trackName := getString(trackData, "name")
			if trackName == "" {
				continue
			}

			artists := extractGraphQLArtists(getMap(trackData, "artists"))
			artistStr := strings.Join(artists, ", ")

			albumData := getMap(trackData, "albumOfTrack")
			albumName := getString(albumData, "name")

			albumArt := ""
			if albumData != nil {
				albumArt = extractGraphQLAlbumArt(getMap(albumData, "coverArt"))
			}

			trackDuration := int(getFloat(getMap(trackData, "trackDuration"), "totalMilliseconds") / 1000)
			trackNumber := int(getFloat(trackData, "trackNumber"))

			trackURI := getString(trackData, "uri")
			trackID := getString(trackData, "id")
			if trackID == "" && strings.Contains(trackURI, ":") {
				parts := strings.Split(trackURI, ":")
				trackID = parts[len(parts)-1]
			}
			if trackID == "" {
				trackID = strconv.Itoa(offset + len(allSongs) + 1)
			}

			song := domain.Song{
				ID:         trackID,
				Title:      trackName,
				Artist:     artistStr,
				Album:      albumName,
				AlbumArt:   albumArt,
				Duration:   trackDuration,
				TrackNum:   trackNumber,
				PlaylistID: playlistID,
			}
			allSongs = append(allSongs, song)
		}

		if len(items) < limit {
			break
		}
		offset += limit
	}

	return allSongs, nil
}
