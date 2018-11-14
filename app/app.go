package app

import (
	"context"
	"fmt"
	"gamelink-apns/config"
	push "gamelink-go/proto_nats_msg"
	"github.com/gogo/protobuf/proto"
	"github.com/nats-io/go-nats"
	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
	log "github.com/sirupsen/logrus"
)

type App struct {
	nc    *nats.Conn
	apns  *apns2.Client
	mchan chan push.PushMsgStruct
}

func NewApp() App {
	mchan := make(chan push.PushMsgStruct)
	return App{mchan: mchan}
}

func (a *App) ConnectNats() {
	nc, err := nats.Connect(config.NatsDialAddress)
	if err != nil {
		log.Fatal(err)
	}
	a.nc = nc
}

func (a *App) ConnectApns(ctx context.Context) {
	authKey, err := token.AuthKeyFromFile(config.ServiceKeyPath)
	if err != nil {
		log.Fatal("token error:", err)
	}
	t := &token.Token{
		AuthKey: authKey,
		// KeyID from developer account (Certificates, Identifiers & Profiles -> Keys)
		KeyID: config.KeyID,
		// TeamID from developer account (View Account -> Membership)
		TeamID: config.TeamID,
	}
	a.apns = apns2.NewTokenClient(t)
}

func (a *App) GetAndPush() {
	var msgStruct push.PushMsgStruct
	// Subscribe to updates
	_, err := a.nc.Subscribe(config.NatsApnsChan, func(m *nats.Msg) {
		err := proto.Unmarshal(m.Data, &msgStruct)
		if err != nil {
			return
		}
		fmt.Println("msgStruct", msgStruct)
		a.mchan <- msgStruct
	})
	if err != nil {
		log.Fatal(err)
	}
	for {
		m := <-a.mchan
		a.prepareMsg(m)
	}
}

func (a *App) prepareMsg(msg push.PushMsgStruct) {
	p := fmt.Sprintf("{\"aps\":{\"alert\":%s}}", msg.Message)
	notification := &apns2.Notification{
		DeviceToken: msg.UserInfo.DeviceID,
		Topic:       "Rocket-X",
		Payload:     []byte(p),
	}
	fmt.Println("prepared", notification)
	res, err := a.apns.Push(notification)
	if err != nil {
		log.Warn(err)
	}
	if res.Sent() {
		log.Println("Sent:", res.ApnsID)
	} else {
		fmt.Printf("Not Sent: %v %v %v\n", res.StatusCode, res.ApnsID, res.Reason)
	}
}