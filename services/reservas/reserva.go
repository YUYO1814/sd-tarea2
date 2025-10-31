package main

import (
"context"
"fmt"
"log"
"net"
"os"

"google.golang.org/grpc"
pb "example.com/sd/proto"
)

type reservasSrv struct {
pb.UnimplementedReservasServiceServer
}

func getenv(k, def string) string {
if v := os.Getenv(k); v != "" {
return v
}
return def
}

func (s *reservasSrv) EnviarLote(ctx context.Context, lote *pb.LoteReservas) (*pb.Ack, error) {
log.Printf("Recibido lote con %d reservas", len(lote.GetReservas()))
// TODO: decidir mesa, enviar a registro (gRPC) y a monitoreo (RabbitMQ)
return &pb.Ack{Message: fmt.Sprintf("Lote recibido: %d reservas", len(lote.GetReservas()))}, nil
}

func main() {
listenAddr := getenv("LISTEN_ADDR", ":50051")
lis, err := net.Listen("tcp", listenAddr)
if err != nil {
log.Fatalf("no pude escuchar en %s: %v", listenAddr, err)
}
s := grpc.NewServer()
pb.RegisterReservasServiceServer(s, &reservasSrv{})
log.Printf("reservas gRPC escuchando en %s", listenAddr)
if err := s.Serve(lis); err != nil {
log.Fatalf("Serve: %v", err)
}
}
