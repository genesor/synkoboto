package synctube

import (
	"net/http"
	"testing"

	synkoboto "github.com/genesor/synkoboto/pkg"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestCreator_CreateRoom(t *testing.T) {
	c := NewCreator(&synkoboto.Configuration{RoomName: "test room"}, http.DefaultClient, logrus.New())

	res, err := c.CreateRoom()
	require.NoError(t, err)
	require.NotEmpty(t, res.ID)
	require.NotEmpty(t, res.URL)
	require.NotEmpty(t, res.Cookie)

	t.Logf("Room URL: %s", res.URL)

	err = c.SetPermissions(res)
	require.NoError(t, err)
}
