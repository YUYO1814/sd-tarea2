package main

import (
"context"
"encoding/json"
"fmt"
"log"
"os"
"time"

"google.golang.org/grpc"
"google.golang.org/grpc/credentials/insecure"

_ "example.com/sd/proto" // asegura que el paquete generado exista en el módulo
)

type Solicitud struct {
Cliente  string `json:"cliente"`
Personas int    `json:"personas"`
Hora     string `json:"hora"`
}

func getenv(k, def string) string {
if v := os.Getenv(k); v != "" {
return v
}
return def
}

func mustDial(addr string) *grpc.ClientConn {
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
conn, err := grpc.DialContext(ctx, addr,
grpc.WithTransportCredentials(insecure.NewCredentials()),
grpc.WithBlock(),
)
if err != nil {
log.Fatalf("no se pudo conectar a %s: %v", addr, err)
}
return conn
}

func main() {
reservasAddr := getenv("RESERVAS_ADDR", "")
monitoreoAddr := getenv("MONITOREO_ADDR", "")

if reservasAddr == "" || monitoreoAddr == "" {
log.Fatalf("Faltan variables de entorno. Exporta RESERVAS_ADDR y MONITOREO_ADDR.")
}

// Carga del lote desde data/reservas.json
dataPath := getenv("RESERVAS_DATA", "data/reservas.json")
f, err := os.Open(dataPath)
if err != nil {
log.Fatalf("No pude abrir %s: %v", dataPath, err)
}
defer f.Close()

var lote []Solicitud
if err := json.NewDecoder(f).Decode(&lote); err != nil {
log.Fatalf("JSON inválido en %s: %v", dataPath, err)
}
if len(lote) == 0 {
log.Fatalf("No hay solicitudes en %s", dataPath)
}
fmt.Printf("Leídas %d solicitudes desde %s\n", len(lote), dataPath)

// Dials gRPC
fmt.Printf("Conectando a reservas gRPC en %s...\n", reservasAddr)
resConn := mustDial(reservasAddr)
defer resConn.Close()

fmt.Printf("Conectando a monitoreo gRPC en %s...\n", monitoreoAddr)
monConn := mustDial(monitoreoAddr)
defer monConn.Close()

// TODO: aquí invocaremos el RPC de reservas y el stream de monitoreo (cuando terminemos de cablear servicios).
fmt.Println("Cliente listo (conexiones establecidas). Próximo paso: cablear RPCs reales.")
}
