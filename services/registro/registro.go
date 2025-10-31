package main

import (
"context"
"log"
"net"
"os"

"google.golang.org/grpc"
pb "example.com/sd/proto"
)

type registroSrv struct {
pb.UnimplementedRegistroServiceServer
}

func getenv(k, def string) string {
if v := os.Getenv(k); v != "" {
return v
}
return def
}

func (s *registroSrv) Persistir(ctx context.Context, r *pb.ResultadoReserva) (*pb.Ack, error) {
log.Printf("Persistir: name=%s party_size=%d requested_pref=%s status=%s mesa_id=%s motivo=%s mesa_tipo=%s",
r.GetName(), r.GetPartySize(), r.GetRequestedPref(), r.GetStatus(),
r.GetMesaId(), r.GetMotivo(), r.GetMesaTipo(),
)
// TODO: guardar en MongoDB
return &pb.Ack{Message: "Persistido (mock)"}, nil
}

func main() {
listenAddr := getenv("LISTEN_ADDR", ":50052")
lis, err := net.Listen("tcp", listenAddr)
if err != nil {
log.Fatalf("no pude escuchar en %s: %v", listenAddr, err)
}
s := grpc.NewServer()
pb.RegisterRegistroServiceServer(s, &registroSrv{})
log.Printf("registro gRPC escuchando en %s", listenAddr)
if err := s.Serve(lis); err != nil {
log.Fatalf("Serve: %v", err)
}
}
