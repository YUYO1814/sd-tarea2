import os
import threading
import queue
import signal
import sys

import grpc
from concurrent import futures

import pika


# imports de los stubs gRPC generados en services/monitoreo/
sys.path.append(os.path.dirname(__file__))
import reservas_pb2 as pb2
import reservas_pb2_grpc as pb2_grpc

RABBITMQ_URL = os.getenv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")
QUEUE_NAME   = os.getenv("MONITOREO_QUEUE", "monitoreo.reservas")
LISTEN_ADDR  = os.getenv("MONITOREO_ADDR", "0.0.0.0:50053")

class MonitoreoService(pb2_grpc.MonitoreoServiceServicer):
    def __init__(self, q: queue.Queue):
        self.q = q

    def Subscribe(self, request, context):
        client_id = getattr(request, "client_id", "")
        print(f"[monitoreo] nueva suscripción client_id={client_id}")
        # Stream infinito hasta que el cliente cierre
        while True:
            try:
                item = self.q.get(timeout=1.0)
            except queue.Empty:
                if context.is_active():
                    continue
                else:
                    print("[monitoreo] stream cancelado por cliente")
                    break
            # item es un dict serializado desde RabbitMQ o un texto simple
            try:
                msg = pb2.ResultadoReserva(
                    name=item.get("name",""),
                    party_size=int(item.get("party_size",0)),
                    requested_pref=item.get("requested_pref",""),
                    status=item.get("status",""),
                    mesa_id=item.get("mesa_id",""),
                    motivo=item.get("motivo",""),
                    mesa_tipo=item.get("mesa_tipo",""),
                )
            except Exception as e:
                print(f"[monitoreo] error construyendo ResultadoReserva: {e}")
                continue
            yield msg

def _rabbit_consumer(url: str, queue_name: str, out_q: queue.Queue):
    params = pika.URLParameters(url)
    connection = pika.BlockingConnection(params)
    channel = connection.channel()
    channel.queue_declare(queue=queue_name, durable=False)

    def _on_msg(ch, method, properties, body):
        # Esperamos JSON simple; si no, lo envolvemos como texto
        import json
        try:
            payload = json.loads(body.decode("utf-8"))
            if isinstance(payload, dict):
                out_q.put(payload)
            else:
                out_q.put({"status":"info","motivo":str(payload)})
        except Exception:
            out_q.put({"status":"info","motivo":body.decode("utf-8", errors="ignore")})

    channel.basic_consume(queue=queue_name, on_message_callback=_on_msg, auto_ack=True)
    print(f"[monitoreo] Consumidor RabbitMQ conectado a {url}, cola={queue_name}")
    try:
        channel.start_consuming()
    except KeyboardInterrupt:
        pass
    finally:
        try:
            channel.stop_consuming()
        except Exception:
            pass
        connection.close()

def serve():
    out_q = queue.Queue(maxsize=1024)

    # Hilo consumidor de RabbitMQ
    t = threading.Thread(target=_rabbit_consumer, args=(RABBITMQ_URL, QUEUE_NAME, out_q), daemon=True)
    t.start()

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=4))
    pb2_grpc.add_MonitoreoServiceServicer_to_server(MonitoreoService(out_q), server)
    server.add_insecure_port(LISTEN_ADDR)
    server.start()
    print(f"[monitoreo] gRPC escuchando en {LISTEN_ADDR}")

    # Señales para apagar limpio
    def _stop(sig, frame):
        print("[monitoreo] apagando...")
        server.stop(0)
        sys.exit(0)

    signal.signal(signal.SIGINT, _stop)
    signal.signal(signal.SIGTERM, _stop)
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
