package main
import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcserver "google.golang.org/grpc"
)

var GrpcAddr = ":9092"

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//Создается канал sigCh, который слушает сигналы ОС.
    //signal.Notify — подписывает канал на SIGINT (os.Interrupt) и SIGTERM.
    //Когда сигнал пойман, вызывается cancel(), что запускает завершение программы.
    //Все это — в отдельной горутине, чтобы основной поток мог дальше работать.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()
// если порт занят или не ответил то он закроет
	lis, err := net.Listen("tcp", GrpcAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	svc := NewService()

	// Starting the gRPC server
	//Создается новый gRPC-сервер.
    // NewGrpcHandler — регистрация ваших gRPC-сервисов, реализованных на основе svc.
	grpcServer := grpcserver.NewServer()
	NewGrpcHandler(grpcServer, svc)

	log.Printf("Starting gRPC server Driver service on port %s", lis.Addr().String())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("failed to serve: %v", err)
			cancel()
		}
	}()

	// wait for the shutdown signal
	// Главный поток блокируется, пока не получит сигнал отмены (cancel() вызывается при получении сигнала или ошибке).
    // После этого вызывается GracefulStop(), который завершает работу сервера, не разрывая активные соединения сразу.
	<-ctx.Done()
	log.Println("Shutting down the server...")
	grpcServer.GracefulStop()
}