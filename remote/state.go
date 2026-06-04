package remote

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ParthSareen/zuko/config"
)

var ErrNoServer = errors.New("zuko serve is not running")

type ServeState struct {
	URL        string    `json:"url"`
	LocalToken string    `json:"local_token"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
}

type Client struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type ClientStore struct {
	Clients []Client `json:"clients"`
}

func ServeStatePath() string {
	return filepath.Join(config.ConfigDir(), "serve.json")
}

func ClientsPath() string {
	return filepath.Join(config.ConfigDir(), "clients.json")
}

func SaveServeState(state ServeState) error {
	if err := os.MkdirAll(config.ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ServeStatePath(), data, 0o600)
}

func LoadServeState() (ServeState, error) {
	data, err := os.ReadFile(ServeStatePath())
	if err != nil {
		if os.IsNotExist(err) {
			return ServeState{}, ErrNoServer
		}
		return ServeState{}, err
	}
	var state ServeState
	if err := json.Unmarshal(data, &state); err != nil {
		return ServeState{}, err
	}
	if state.URL == "" || state.LocalToken == "" {
		return ServeState{}, ErrNoServer
	}
	return state, nil
}

func RemoveServeState(localToken string) error {
	state, err := LoadServeState()
	if err != nil {
		if errors.Is(err, ErrNoServer) {
			return nil
		}
		return err
	}
	if localToken != "" && state.LocalToken != localToken {
		return nil
	}
	err = os.Remove(ServeStatePath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func LoadClients() (ClientStore, error) {
	data, err := os.ReadFile(ClientsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return ClientStore{}, nil
		}
		return ClientStore{}, err
	}
	var store ClientStore
	if err := json.Unmarshal(data, &store); err != nil {
		return ClientStore{}, err
	}
	return store, nil
}

func SaveClients(store ClientStore) error {
	if err := os.MkdirAll(config.ConfigDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ClientsPath(), data, 0o600)
}

func PairClient(name string) (Client, string, error) {
	if name == "" {
		name = "iPhone"
	}
	token, err := NewToken()
	if err != nil {
		return Client{}, "", err
	}
	id, err := randomURLToken(8)
	if err != nil {
		return Client{}, "", err
	}
	store, err := LoadClients()
	if err != nil {
		return Client{}, "", err
	}
	client := Client{
		ID:        id,
		Name:      name,
		TokenHash: TokenHash(token),
		CreatedAt: time.Now(),
	}
	store.Clients = append(store.Clients, client)
	if err := SaveClients(store); err != nil {
		return Client{}, "", err
	}
	return client, token, nil
}

func ValidClientToken(token string) bool {
	if token == "" {
		return false
	}
	store, err := LoadClients()
	if err != nil {
		return false
	}
	hash := TokenHash(token)
	for _, client := range store.Clients {
		if subtle.ConstantTimeCompare([]byte(client.TokenHash), []byte(hash)) == 1 {
			return true
		}
	}
	return false
}

func NewToken() (string, error) {
	return randomURLToken(32)
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomURLToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
