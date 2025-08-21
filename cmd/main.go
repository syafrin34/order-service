package main

import (
	"order-service/internal/api"
	"order-service/internal/repository"
	"order-service/internal/service"

	"github.com/labstack/echo/v4"
)

func main() {
	orderRepo := repository.NewOrderRepository()
	orderService := service.NewOrderService(*orderRepo, "", "")
	orderHandler := api.NewOrderHandler(*orderService)
	
	
	e := echo.New()

	e.POST("/orders", orderHandler.CreateOrder)
	e.PUT("/orders", orderHandler.UpdateOrder)
	e.DELETE("/orders/:id", orderHandler.CancelOrder)

	e.Logger.Fatal(e.Start(":8082"))

}
