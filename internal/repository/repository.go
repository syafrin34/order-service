package repository

import "order-service/internal/entity"

type OrderRepository struct {

}

func NewOrderRepository()*OrderRepository{
	return &OrderRepository{

	}
}

// mock order db
var orders = map[int]*entity.OrderEntity{
	1: {ID: 1, ProductRequests: make([]entity.ProductRequest, 0), Status: "created", Total: 100},
	2: {ID: 2, ProductRequests: make([]entity.ProductRequest, 0), Status: "paid", Total: 200},

}

func (r *OrderRepository)GetOrderByID(id int)(*entity.OrderEntity, error){
	order, ok := orders[id]
	if !ok {
		return nil, nil
	}
	return  order, nil
}

func (r *OrderRepository)CreateOrder(order *entity.OrderEntity)(*entity.OrderEntity, error){
	order.ID = 3
	orders[order.ID] = order
	return order, nil
}

func (r *OrderRepository)UpdateOrder(order *entity.OrderEntity)(*entity.OrderEntity, error){
	orders[order.ID] = order
	return order, nil
}
func (r *OrderRepository)DeleteOrder(id int)error{
	delete(orders, id)
	return  nil
}