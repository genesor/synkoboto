package synctube

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/benbjohnson/clock"
	synkoboto "github.com/genesor/synkoboto/pkg"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewCreator ...
func NewCreator(cfg *synkoboto.Configuration, c *http.Client, l *logrus.Logger) *Creator {
	return &Creator{
		Config: cfg,
		Client: c,
		Logger: l,
		Clock:  clock.New(),
	}
}

// Creator is in charge of creating and managing SyncTube rooms.
type Creator struct {
	Config *synkoboto.Configuration
	Client *http.Client
	Logger *logrus.Logger
	Clock  clock.Clock
}

// Room represents a remote SyncTube room
type Room struct {
	ID     string `json:"id"`
	URL    string
	Cookie *http.Cookie
}

// CreateRoom creates a new SyncTube room.
func (c *Creator) CreateRoom() (*Room, error) {
	req, err := http.NewRequest(http.MethodPost, "https://sync-tube.de/api/create", nil)
	if err != nil {
		return nil, errors.Wrap(err, "error creating HTTP request")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "error performing HTTP request")
	}

	defer resp.Body.Close()
	room := new(Room)
	if err := json.NewDecoder(resp.Body).Decode(room); err != nil {
		return nil, errors.Wrap(err, "error reading HTTP response")
	}

	room.Cookie = resp.Cookies()[0]
	room.URL = fmt.Sprintf("https://sync-tube.de/room/%s", room.ID)

	return room, nil
}

// SetPermissions updates the Room permissions by connection to the Room websocket and
// allowing everybody to skip, seek, play, move, add and remove videos
// This function also renames the Room to Koufar TV
func (c *Creator) SetPermissions(r *Room) error {
	u := fmt.Sprintf("wss://sync-tube.de/ws/%s/ewB9AA==", r.ID)
	c.Logger.Infof("connecting to %s", u)

	h := http.Header(make(map[string][]string))
	h.Set("Cookie", r.Cookie.String()) // Use cookie returned during creation to act as the Room owner
	h.Set("Host", "sync-tube.de")
	h.Set("Origin", "https://sync-tube.de")

	conn, resp, err := websocket.DefaultDialer.Dial(u, h)
	if err != nil {
		body := ""
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()

			b, errRead := io.ReadAll(resp.Body)
			if errRead != nil {
				c.Logger.WithError(errRead).Error("error reading ws dial body")
			} else {
				body = string(b)
			}
		}
		c.Logger.WithError(err).WithField("response", body).Error("error connecting to room WS")

		return errors.Wrap(err, "error connecting to room WS")
	}

	defer conn.Close()

	if err := c.publish(conn, fmt.Sprintf(`[1,"%s",%d]`, c.Config.RoomName, c.Clock.Now().Unix())); err != nil {
		return errors.Wrap(err, "error updating room name")
	}
	if err := c.publish(conn, `[4,{"pid":"skip","group":0,"allow":true}]`); err != nil {
		return errors.Wrap(err, "error updating skip permission")
	}
	if err := c.publish(conn, `[4,{"pid":"seek","group":0,"allow":true}]`); err != nil {
		return errors.Wrap(err, "error updating seek permission")
	}
	if err := c.publish(conn, `[4,{"pid":"play","group":0,"allow":true}]`); err != nil {
		return errors.Wrap(err, "error updating play permission")
	}
	if err := c.publish(conn, `[4,{"pid":"move","group":0,"allow":true}]`); err != nil {
		return errors.Wrap(err, "error updating move permission")
	}
	if err := c.publish(conn, `[4,{"pid":"rem","group":0,"allow":true}]`); err != nil {
		return errors.Wrap(err, "error updating remove permission")
	}
	if err := c.publish(conn, `[4,{"pid":"add","group":0,"allow":true}]`); err != nil {
		return errors.Wrap(err, "error updating add permission")
	}

	return nil
}

func (c *Creator) publish(conn *websocket.Conn, msg string) error {
	return conn.WriteMessage(websocket.TextMessage, []byte(msg))

}
