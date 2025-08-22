package main

import (
	"order-service/internal/api"
	"order-service/internal/repository"
	"order-service/internal/service"

	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	orderRepo := repository.NewOrderRepository()
	orderService := service.NewOrderService(*orderRepo, "http://localhost:8081", "http://localhost:8083")
	orderHandler := api.NewOrderHandler(*orderService)
	
	
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(echojwt.JWT([]byte("secret")))


	e.POST("/orders", orderHandler.CreateOrder)
	e.PUT("/orders", orderHandler.UpdateOrder)
	e.DELETE("/orders/:id", orderHandler.CancelOrder)

	e.Logger.Fatal(e.Start(":8082"))

}
