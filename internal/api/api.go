// Package api
package api

import (
	"order-service/internal/entity"
	"order-service/internal/service"
	"strconv"

	"github.com/labstack/echo/v4"
)

type OrderHandler struct {
	orderService service.OrderService
}

func NewOrderHandler(orderService service.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

func (h *OrderHandler) CreateOrder(c echo.Context) error {
	ctx := c.Request().Context()
	order := entity.OrderEntity{}
	if err := c.Bind(&order); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request payload"})
	}

	createdOrder, err := h.orderService.CreateOrder(ctx, &order)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, createdOrder)

}
func (h *OrderHandler) UpdateOrder(c echo.Context) error {
	ctx := c.Request().Context()
	order := entity.OrderEntity{}
	if err := c.Bind(&order); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request payload"})
	}

	updatedOrder, err := h.orderService.UpdateOrder(ctx, &order)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, updatedOrder)
}

func (h *OrderHandler) CancelOrder(c echo.Context) error {
	ctx := c.Request().Context()
	id := c.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid ID"})
	}
	order, err := h.orderService.CancelOrder(ctx, idInt)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, order)
}
