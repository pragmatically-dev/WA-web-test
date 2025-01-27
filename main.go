package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Println("Received a message!", v.Message.GetConversation())
	}
}

func sendTestMessage(client *whatsmeow.Client) {
	receiver := os.Getenv("RECV_NUM")

	jid := types.NewJID(receiver, "s.whatsapp.net")
	resp, err := client.SendMessage(context.Background(), jid, &waE2E.Message{Conversation: proto.String("Hola desde golang")})
	if err != nil {
		panic(err)
	}
	fmt.Println("Send message:", resp)
}

func config() (*store.Device, waLog.Logger) {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	//Setting UserAgent
	store.SetOSInfo("Linux", store.GetWAVersion())
	store.DeviceProps.PlatformType = waCompanionReg.DeviceProps_CHROME.Enum()
	clientLog := waLog.Stdout("Client", "INFO", true)

	return deviceStore, clientLog
}

func main() {
	deviceStore, clientLog := config()
	client := whatsmeow.NewClient(deviceStore, clientLog)
	defer client.Disconnect()

	client.AddEventHandler(eventHandler)

	// Already logged in, just connect
	if client.Store.ID != nil && client.Connect() != nil {
		panic(fmt.Errorf("Unable to connect"))
	}

	// No ID stored, new login
	qrChan, _ := client.GetQRChannel(context.Background())

	if client.Connect() != nil {
		panic(fmt.Errorf("Unable to connect"))
	}

	for evt := range qrChan {

		if evt.Event == "code" {
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		}

	}

	sendTestMessage(client)

	// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

}
