package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/MrWormHole/grpc-chat-service/proto"
	"google.golang.org/grpc"
)

var client proto.ChatServiceClient
var wg *sync.WaitGroup

func init() {
	wg = &sync.WaitGroup{}
}

func main() {
	time := time.Now()
	done := make(chan struct{})

	username := flag.String("N", "Anon", "The name of the user")
	flag.Parse()

	id := sha256.Sum256([]byte(time.String() + *username))

	connection, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error occured while trying to connect the server: %v", err)
	}

	client = proto.NewChatServiceClient(connection)
	user := &proto.User{
		Id:       hex.EncodeToString(id[:]),
		Username: *username,
	}
	connect(user)

	wg.Add(1)
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			chatMessage := &proto.ChatMessage{
				Id:        user.Id,
				Message:   scanner.Text(),
				Timestamp: time.String(),
			}

			_, err := client.BroadcastChatMessage(context.Background(), chatMessage)
			if err != nil {
				fmt.Printf("Error occured while sending a message: %v", err)
				break
			}
		}
	}()

	go func() {
		wg.Wait()
		close(done)
	}()

	<-done
}

func connect(user *proto.User) error {
	var streamError error

	stream, err := client.RegisterConnection(context.Background(), &proto.Connect{
		User:   user,
		Active: true,
	})
	if err != nil {
		return fmt.Errorf("Error occured while creating connection: %v", err)
	}

	wg.Add(1)
	go func(stream proto.ChatService_RegisterConnectionClient) {
		defer wg.Done()

		for {
			chatMessage, err := stream.Recv()
			if err != nil {
				streamError = fmt.Errorf("Error occured while reading a message: %v", err)
				break
			}

			fmt.Printf("%v : %s\n", chatMessage.Id, chatMessage.Message)
		}
	}(stream)

	return streamError
}
