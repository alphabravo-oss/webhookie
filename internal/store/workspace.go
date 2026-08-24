package store

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID               string    `json:"id"`
	Provider         string    `json:"provider"`
	Name             string    `json:"name"`
	InteractivityURL string    `json:"interactivityUrl"`
	SigningSecret    string    `json:"signingSecret"`
	CreatedAt        time.Time `json:"createdAt"`
	Channels         []Channel `json:"channels,omitempty"`
}

type Channel struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	SinkID      string    `json:"sinkId"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Kind        string    `json:"kind"`
	Path        string    `json:"path,omitempty"`
	URL         string    `json:"url,omitempty"`
	Token       string    `json:"token,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Interaction struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"createdAt"`
	WorkspaceID string    `json:"workspaceId"`
	ChannelID   string    `json:"channelId"`
	EventID     string    `json:"eventId"`
	Kind        string    `json:"kind"`
	ActionID    string    `json:"actionId"`
	Payload     string    `json:"payload"`
	Target      string    `json:"target"`
	Status      *int      `json:"status"`
	Error       string    `json:"error,omitempty"`
	LatencyMS   int       `json:"latencyMs"`
}

func (s *Store) UpsertWorkspace(ctx context.Context, ws Workspace) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, provider, name, interactivity_url, signing_secret, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			interactivity_url=excluded.interactivity_url,
			signing_secret=excluded.signing_secret
	`, ws.ID, ws.Provider, ws.Name, ws.InteractivityURL, ws.SigningSecret, ws.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetWorkspace(ctx context.Context, id string) (Workspace, error) {
	ws, err := scanWorkspace(s.db.QueryRowContext(ctx, `SELECT id, provider, name, interactivity_url, signing_secret, created_at FROM workspaces WHERE id=?`, id))
	if err != nil {
		return Workspace{}, err
	}
	ch, err := s.ListChannels(ctx, ws.ID)
	if err != nil {
		return Workspace{}, err
	}
	ws.Channels = ch
	return ws, nil
}

func (s *Store) GetWorkspaceByProvider(ctx context.Context, provider string) (Workspace, error) {
	ws, err := scanWorkspace(s.db.QueryRowContext(ctx, `SELECT id, provider, name, interactivity_url, signing_secret, created_at FROM workspaces WHERE provider=?`, provider))
	if err != nil {
		return Workspace{}, err
	}
	ch, err := s.ListChannels(ctx, ws.ID)
	if err != nil {
		return Workspace{}, err
	}
	ws.Channels = ch
	return ws, nil
}

func (s *Store) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, provider, name, interactivity_url, signing_secret, created_at FROM workspaces ORDER BY provider`)
	if err != nil {
		return nil, err
	}
	var out []Workspace
	for rows.Next() {
		ws, err := scanWorkspace(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, ws)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range out {
		ch, err := s.ListChannels(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Channels = ch
	}
	return out, nil
}

func (s *Store) InsertChannel(ctx context.Context, ch Channel) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO channels (id, workspace_id, sink_id, name, slug, kind, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, ch.ID, ch.WorkspaceID, ch.SinkID, ch.Name, ch.Slug, ch.Kind, ch.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) GetChannel(ctx context.Context, id string) (Channel, error) {
	return scanChannel(s.db.QueryRowContext(ctx, `
		SELECT c.id, c.workspace_id, c.sink_id, c.name, c.slug, c.kind, c.created_at, s.path, s.token
		FROM channels c JOIN sinks s ON s.id = c.sink_id WHERE c.id=?
	`, id))
}

func (s *Store) GetChannelBySink(ctx context.Context, sinkID string) (Channel, error) {
	return scanChannel(s.db.QueryRowContext(ctx, `
		SELECT c.id, c.workspace_id, c.sink_id, c.name, c.slug, c.kind, c.created_at, s.path, s.token
		FROM channels c JOIN sinks s ON s.id = c.sink_id WHERE c.sink_id=?
	`, sinkID))
}

func (s *Store) ListChannels(ctx context.Context, workspaceID string) ([]Channel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.workspace_id, c.sink_id, c.name, c.slug, c.kind, c.created_at, s.path, s.token
		FROM channels c JOIN sinks s ON s.id = c.sink_id
		WHERE c.workspace_id=? ORDER BY c.created_at
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		ch, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ch)
	}
	return out, rows.Err()
}

func (s *Store) InsertInteraction(ctx context.Context, in Interaction) error {
	var status any
	if in.Status != nil {
		status = *in.Status
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO interactions (id, created_at, workspace_id, channel_id, event_id, kind, action_id, payload, target, status, error, latency_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, in.ID, in.CreatedAt.UTC().Format(time.RFC3339Nano), in.WorkspaceID, in.ChannelID, in.EventID, in.Kind, in.ActionID, []byte(in.Payload), in.Target, status, in.Error, in.LatencyMS)
	return err
}

func (s *Store) GetInteraction(ctx context.Context, id string) (Interaction, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, created_at, workspace_id, channel_id, event_id, kind, action_id, payload, target, status, error, latency_ms FROM interactions WHERE id=?`, id)
	return scanInteraction(row)
}

func scanInteraction(row rowScanner) (Interaction, error) {
	var in Interaction
	var created string
	var status sql.NullInt64
	var errText sql.NullString
	var payload []byte
	if err := row.Scan(&in.ID, &created, &in.WorkspaceID, &in.ChannelID, &in.EventID, &in.Kind, &in.ActionID, &payload, &in.Target, &status, &errText, &in.LatencyMS); err != nil {
		return Interaction{}, err
	}
	in.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	in.Payload = string(payload)
	if status.Valid {
		v := int(status.Int64)
		in.Status = &v
	}
	in.Error = errText.String
	return in, nil
}

func (s *Store) ListInteractions(ctx context.Context, channelID string) ([]Interaction, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, created_at, workspace_id, channel_id, event_id, kind, action_id, payload, target, status, error, latency_ms FROM interactions WHERE channel_id=? ORDER BY created_at DESC LIMIT 100`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Interaction
	for rows.Next() {
		in, err := scanInteraction(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func NewChannelID() string { return uuid.NewString() }

func Slugify(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s := strings.Trim(re.ReplaceAllString(name, "-"), "-")
	if s == "" {
		s = "channel"
	}
	return s
}

func scanWorkspace(row rowScanner) (Workspace, error) {
	var ws Workspace
	var created string
	if err := row.Scan(&ws.ID, &ws.Provider, &ws.Name, &ws.InteractivityURL, &ws.SigningSecret, &created); err != nil {
		return Workspace{}, err
	}
	ws.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return ws, nil
}

func scanChannel(row rowScanner) (Channel, error) {
	var ch Channel
	var created string
	if err := row.Scan(&ch.ID, &ch.WorkspaceID, &ch.SinkID, &ch.Name, &ch.Slug, &ch.Kind, &created, &ch.Path, &ch.Token); err != nil {
		return Channel{}, err
	}
	ch.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return ch, nil
}
