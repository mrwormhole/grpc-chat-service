package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"

	"github.com/MrWormHole/grpc-chat-service/proto"
	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
)

var grpcLog glog.LoggerV2

func init() {
	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

func main() {
	var connections []*Connection

	server := &Server{connections}
	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error occured while creating the server %v", err)
	}

	grpcLog.Info("Starting server at port :8080")
	proto.RegisterChatServiceServer(grpcServer, server)
	reflection.Register(grpcServer)

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatalf("Error occured while creating the grpc server %v", err)
	}
}

type Connection struct {
	stream proto.ChatService_RegisterConnectionServer
	id     string
	active bool
	error  chan error
}

type Server struct {
	Connection []*Connection
}

func (s *Server) RegisterConnection(connect *proto.Connect, stream proto.ChatService_RegisterConnectionServer) error {
	connection := &Connection{
		stream: stream,
		id:     connect.User.Id,
		active: true,
		error:  make(chan error),
	}

	s.Connection = append(s.Connection, connection)
	grpcLog.Infof("A user connected to stream(%v) with id(%v)", connection.stream, connection.id)

	return <-connection.error
}

func (s *Server) BroadcastChatMessage(context context.Context, chatMessage *proto.ChatMessage) (*proto.Close, error) {
	wg := sync.WaitGroup{}
	done := make(chan struct{})

	for _, connection := range s.Connection {
		wg.Add(1)
		go func(chatMessage *proto.ChatMessage, connection *Connection) {
			defer wg.Done()

			if connection.active {
				err := connection.stream.Send(chatMessage)
				grpcLog.Infof("Sending message to stream(%v)", connection.stream)

				if err != nil {
					grpcLog.Errorf("Error Occured: %v on stream(%v)", err, connection.stream)
					connection.active = false
					connection.error <- err
				}
			}
		}(chatMessage, connection)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	<-done
	return &proto.Close{}, nil
}
