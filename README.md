# Tarea 2 - Sistemas Distribuidos
### Integración de servicios gRPC con RabbitMQ y MongoDB

**Autor:** Eugenio Pérez  
**Fecha:** 31/10/2025

---

## 1. Descripción general
El sistema consiste en tres microservicios distribuidos sobre distintas máquinas virtuales, comunicados mediante gRPC y mensajería con RabbitMQ.

- **registro** → MV3 (`10.10.31.12:50052`)
- **reservas** → MV2 (`10.10.31.11:50051`)
- **cliente** → MV1 (`10.10.31.10`)

---

## 2. Preparación de entorno
En MV2:
```bash
docker run -d --name mongo -p 27017:27017 mongo:4.4
sudo systemctl start rabbitmq-server
```
En todas las VM:
```bash 
sudo apt update
sudo apt install -y golang-go
```

---

## 3. Compilacion de servicios
En cada VM, dentro de ~/sd-tarea2:
MV3 (registro)
```bash
go build -o bin/registro ./services/registro
nohup ~/sd-tarea2/bin/registro > logs.registro.out 2>&1 &
```
MV2 (reservas)
```bash
go build -o bin/reservas ./services/reservas
export LISTEN_ADDR=":50051" REGISTRO_ADDR="10.10.31.12:50052" \
       RABBITMQ_URL="amqp://guest:guest@127.0.0.1:5672/"
nohup ~/sd-tarea2/bin/reservas > logs.reservas.out 2>&1 &
```
MV1 (cliente)   
```bash 
go build -o bin/cliente ./cmd/cliente
export RESERVAS_ADDR="10.10.31.11:50051"
export MONITOREO_ADDR="10.10.31.12:50052"
~/sd-tarea2/bin/cliente
```     

---

## 4. Smoke test

Prueba directa del RPC:
```bash
grpcurl -plaintext -import-path proto -proto reservas.proto \
-d '{"reservas":[{"name":"eugenio","phone":"N/A","party_size":2,"preferences":"ventana"}]}' \
10.10.31.11:50051 sd.ReservasService.EnviarLote
```
Resultado esperado:
```json
{"message": "Lote recibido: 2 reservas"}

```
En logs.reservas.out:
```
Recibido lote con 2 reservas
```

## 5. Diagrama de despliegue
```
     +----------------+             +----------------+             +----------------+
     |     Cliente    |  gRPC call  |    Reservas    |  gRPC call  |    Registro    |
     | (MV1:50051↔52) |────────────▶| (MV2:50051)    |────────────▶| (MV3:50052)    |
     +----------------+             +----------------+             +----------------+
```
## 6. Notas finales
✅ Registro corriendo en MV3 (:50052)
✅ Reservas corriendo en MV2 (:50051)
✅ Cliente conectado exitosamente (2 reservas enviadas)

Se utilizó Go 1.24.9 y gRPC 1.76.0.

RabbitMQ y MongoDB quedan en ejecución en MV2.

Próximo paso sería integrar el envío real hacia Registro y Monitoreo.

---

## 7. Evidencias de ejecución

### 7.1 RPC exitoso (smoke test con grpcurl)
Comando ejecutado desde MV1:
```bash
grpcurl -plaintext -import-path proto -proto reservas.proto \
-d '{"reservas":[{"name":"eugenio","phone":"N/A","party_size":2,"preferences":"ventana"},{"name":"sofia","phone":"N/A","party_size":4,"preferences":"terraza"}]}' \
10.10.31.11:50051 sd.ReservasService.EnviarLote
``` 
### Respuesta recibida:
```bash
{"message": "Lote recibido: 2 reservas"}
```

### 7.2 Logs del servicio Reservas (MV2)
```bash
2025/10/31 16:53:38 reservas gRPC escuchando en :50051
2025/10/31 17:21:24 Recibido lote con 2 reservas
```

### 7.3 Puertos escuchando en MV2
```bash
LISTEN 0      4096               *:50051            *:*    users:(("reservas",pid=62824,fd=3))
```

MV3 (Registro)
```bash
LISTEN 0      4096               *:50052            *:*    users:(("registro",pid=1205,fd=3))
```

### 7.4 Monitoreo (RabbitMQ → streaming gRPC)

Suscripción (MV1):
```bash
grpcurl -plaintext -import-path proto -proto reservas.proto \
  -d '{"client_id":"cli-1"}' 127.0.0.1:50053 sd.MonitoreoService.Subscribe
```

Publicación (MV2):
```python
python3 - <<'EOF'
import json, pika
url="amqp://sduser:sdpass@127.0.0.1:5672/"
q="monitoreo.reservas"
conn=pika.BlockingConnection(pika.URLParameters(url)); ch=conn.channel()
ch.queue_declare(queue=q, durable=False)
msg={"name":"eugenio","party_size":2,"requested_pref":"ventana","status":"exitoso","mesa_id":"mesa-01","motivo":"","mesa_tipo":"interior"}
ch.basic_publish(exchange="", routing_key=q, body=json.dumps(msg).encode())
print("Publicado OK")
conn.close()
EOF
``` 
Salida esperada (MV1 -grpcurl):
```json
{
  "name": "eugenio",
  "partySize": 2,
  "requestedPref": "ventana",
  "status": "exitoso",
  "mesaId": "mesa-01",
  "mesaTipo": "interior"
}
```
