package synkoboto

type Configuration struct {
	ServerID  string `config:"SERVER_ID,required"`
	AppID     string `config:"APP_ID,required"`
	BotToken  string `config:"BOT_TOKEN,required"`
	BotSecret string `config:"BOT_SECRET,required"`

	RoomName string `config:"ROOM_NAME"`
}
