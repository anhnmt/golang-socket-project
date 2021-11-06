package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/centrifugal/centrifuge"
	"go.uber.org/zap"

	"github.com/xdorro/golang-socket-project/internal/config"
	"github.com/xdorro/golang-socket-project/internal/logger"
	"github.com/xdorro/golang-socket-project/internal/redis"
)

const (
	port = ":8000"
)

var (
	log *zap.Logger
)

type clientMessage struct {
	Timestamp int64  `json:"timestamp"`
	Input     string `json:"input"`
}

func init() {
	log = logger.NewLogger()
}

func handleLog(e centrifuge.LogEntry) {
	log.Info(fmt.Sprintf("[centrifuge] %s: %v", e.Message, e.Fields))
}

func authMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		ctx = centrifuge.SetCredentials(ctx, &centrifuge.Credentials{
			UserID: "42",
			Info:   []byte(`{"name": "Alexander"}`),
		})
		r = r.WithContext(ctx)
		h.ServeHTTP(w, r)
	})
}

func main() {
	log.Info("Running...")
	// Config
	conf := config.NewConfig(log)

	cfg := centrifuge.DefaultConfig
	cfg.LogLevel = centrifuge.LogLevelDebug
	cfg.LogHandler = handleLog

	node, err := centrifuge.New(cfg)
	if err != nil {
		log.Fatal("centrifuge.New()",
			zap.Error(err),
		)
	}

	// node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
	// 	return centrifuge.ConnectReply{
	// 		Credentials: &centrifuge.Credentials{
	// 			UserID: "123123123",
	// 		},
	// 	}, nil
	// })

	node.OnConnect(func(client *centrifuge.Client) {
		transport := client.Transport()
		log.Info(fmt.Sprintf("user %s connected via %s with protocol: %s", client.UserID(), transport.Name(), transport.Protocol()))

		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			log.Info(fmt.Sprintf("user %s subscribes on %s", client.UserID(), e.Channel))
			cb(centrifuge.SubscribeReply{
				Options: centrifuge.SubscribeOptions{
					Presence:  true,
					JoinLeave: true,
					Recover:   true,
				},
			}, nil)
		})

		client.OnUnsubscribe(func(e centrifuge.UnsubscribeEvent) {
			log.Info(fmt.Sprintf("user %s unsubscribed from %s", client.UserID(), e.Channel))
		})

		client.OnPublish(func(e centrifuge.PublishEvent, cb centrifuge.PublishCallback) {
			log.Info(fmt.Sprintf("user %s publishes into channel %s: %s", client.UserID(), e.Channel, string(e.Data)))
			// cb(centrifuge.PublishReply{
			// 	Options: centrifuge.PublishOptions{
			// 		HistorySize: 100,
			// 		HistoryTTL:  5 * time.Second,
			// 	},
			// }, nil)
			if !client.IsSubscribed(e.Channel) {
				cb(centrifuge.PublishReply{}, centrifuge.ErrorPermissionDenied)
				return
			}

			var msg clientMessage
			if err = json.Unmarshal(e.Data, &msg); err != nil {
				cb(centrifuge.PublishReply{}, centrifuge.ErrorBadRequest)
				return
			}
			msg.Timestamp = time.Now().Unix()
			data, _ := json.Marshal(msg)

			result, err := node.Publish(
				e.Channel, data,
				centrifuge.WithHistory(100, 5*time.Second),
				centrifuge.WithClientInfo(e.ClientInfo),
			)

			cb(centrifuge.PublishReply{Result: &result}, err)
		})

		client.OnPresence(func(e centrifuge.PresenceEvent, cb centrifuge.PresenceCallback) {
			cb(centrifuge.PresenceReply{}, nil)
		})

		client.OnDisconnect(func(e centrifuge.DisconnectEvent) {
			log.Info(fmt.Sprintf("user %s disconnected, disconnect: %s", client.UserID(), e.Disconnect))
		})
	})

	// Handler redis
	redis.HandlerRedisShard(node)

	// Run node
	if err = node.Run(); err != nil {
		log.Fatal("node.Run()",
			zap.Error(err),
		)
	}

	http.Handle("/ws", authMiddleware(centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{})))
	http.Handle("/", http.FileServer(http.Dir("./public")))

	go func() {
		if err = http.ListenAndServe(port, nil); err != nil {
			log.Fatal("http.ListenAndServe()",
				zap.Error(err),
			)
		}
	}()

	waitExitSignal(conf.Ctx, node)
}

func waitExitSignal(ctx context.Context, n *centrifuge.Node) {
	c := make(chan os.Signal, 1)                    // Create channel to signify a signal being sent
	signal.Notify(c, os.Interrupt, syscall.SIGTERM) // When an interrupt or termination signal is sent, notify the channel

	_ = <-c // This blocks the main thread until an interrupt is received
	fmt.Println("Gracefully shutting down...")
	_ = n.Shutdown(ctx)

	fmt.Println("Running cleanup tasks...")

	// Your cleanup tasks go here
	logger.Close(log)
	fmt.Println("Server was successful shutdown.")
}
