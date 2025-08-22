package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"order-service/internal/entity"
	"order-service/internal/repository"
	"os"

	"github.com/rs/zerolog"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
type OrderService struct {
	orderRepo repository.OrderRepository
	productServiceURL string
	pricingServiceURL string

}

func NewOrderService(orderRepo repository.OrderRepository, productServiceURL, pricingServiceURL string)*OrderService{
	return &OrderService{
		orderRepo: orderRepo,
		productServiceURL: pricingServiceURL,
		pricingServiceURL: pricingServiceURL,
	}
}

func (o *OrderService)CreateOrder(order *entity.OrderEntity)(*entity.OrderEntity, error){
	for _, productRequest := range order.ProductRequests{
		// check product availability
		available, err := o.checkProductStock(productRequest.ProductID, productRequest.Quantity)
		 if err != nil {
			logger.Error().Err(err).Msgf("Error checking product stock for product %d", productRequest.ProductID)
			return nil, err
		 }
		// get pricing
		pricing, err := o.GetPricing(productRequest.ProductID)
		if err != nil {
			logger.Error().Err(err).Msgf("Error getting pricing for product  %d", productRequest.ProductID)
			return  nil, err
		}

		if !available {
			logger.Warn().Msgf("product %d out of stock", productRequest.ProductID)
			return  nil, fmt.Errorf("product out of stock")
		}

		productRequest.FinalPrice = float64(productRequest.Quantity)*pricing.FinalPrice
		productRequest.MarkUp = float64(productRequest.Quantity)* pricing.Markup
		productRequest.Discount = float64(productRequest.Quantity)* pricing.Discount
	}

	// calculate order total 
	order.Total = 0
	for _, productRequest := range order.ProductRequests{
		order.Total += productRequest.FinalPrice
	}

	createdOrder, err := o.orderRepo.CreateOrder(order)
	if err != nil {
		logger.Error().Err(err).Msgf("Error creating order")
		return nil, err
	}

	return  createdOrder, nil
}
func (o *OrderService)UpdateOrder(order *entity.OrderEntity)(*entity.OrderEntity, error){
	updateOrder, err := o.orderRepo.UpdateOrder(order)
	if err != nil {
		logger.Error().Err(err).Msgf("Error updating order")
		return nil, err
	}

	return  updateOrder, nil
}
func (o *OrderService)CancelOrder(id int)(*entity.OrderEntity, error){
	order, err := o.orderRepo.GetOrderByID(id)
	if err != nil {
		logger.Error().Err(err).Msgf("Error getting order by ID %d", id)
		return  nil, err
	}
	order.Status = "cancelled"

	updateOrder, err := o.orderRepo.UpdateOrder(order)
	if err != nil {
		logger.Error().Err(err).Msgf("Error updating order")
		return  nil, err
	}

	return  updateOrder, nil
}

func(o *OrderService)checkProductStock(productID int, quantity int)(bool, error){
	resp, err := http.Get(fmt.Sprintf("%s/product/%d/stock", o.pricingServiceURL, productID))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("product not valid")
	}

	var stockData map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&stockData); err != nil {
		return  false, err
	}

	availableStock := stockData["stock"]
	return  availableStock >= quantity,nil

}

func(o *OrderService)GetPricing(productID int)(*entity.Pricing, error){
	resp, err := http.Get(fmt.Sprintf("%s/products/%d/pricing", o.pricingServiceURL, productID))
	if err != nil {
		return  nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK{
		return nil, fmt.Errorf("failed to get pricing")
	}
	var pricing entity.Pricing
	if err := json.NewDecoder(resp.Body).Decode(&pricing);err != nil {
		return nil, err
	}
	return &pricing, nil
}


