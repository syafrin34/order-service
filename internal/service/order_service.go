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

func (o *OrderService)CreateOrder( order *entity.OrderEntity)(*entity.OrderEntity, error){
	
	availabilityCh := make(chan struct{
		ProductID int
		Available bool
		Err error
	}, len(order.ProductRequests))

	pricingCh := make(chan struct{
		ProductID int
		FinalPrice float64
		MarkUp float64
		Discount float64
		Err error
	}, len(order.ProductRequests))

	
	for _, productRequest := range order.ProductRequests{

		// 1. proses syncronus

		// check product availability
		// available, err := o.checkProductStock(productRequest.ProductID, productRequest.Quantity)
		//  if err != nil {
		// 	logger.Error().Err(err).Msgf("Error checking product stock for product %d", productRequest.ProductID)
		// 	return nil, err
		//  }
		// get pricing
		// pricing, err := o.GetPricing(productRequest.ProductID)
		// if err != nil {
		// 	logger.Error().Err(err).Msgf("Error getting pricing for product  %d", productRequest.ProductID)
		// 	return  nil, err
		// }

		// if !available {
		// 	logger.Warn().Msgf("product %d out of stock", productRequest.ProductID)
		// 	return  nil, fmt.Errorf("product out of stock")
		// }

		// productRequest.FinalPrice = float64(productRequest.Quantity)*pricing.FinalPrice
		// productRequest.MarkUp = float64(productRequest.Quantity)* pricing.Markup
		// productRequest.Discount = float64(productRequest.Quantity)* pricing.Discount

		
		// 2. proses asyncronus
		// go routine proses cek available stock product
		go func(productRequest *entity.ProductRequest) {
			available, err := o.checkProductStock(productRequest.ProductID, productRequest.Quantity)
			availabilityCh <- struct{ProductID int; Available bool; Err error}{
				ProductID: productRequest.ProductID,
				Available: available,
				Err: err,
			}
		}(&productRequest)
		
		// go routine untuk pricing

		go func(productRequest *entity.ProductRequest){
			pricing, err := o.GetPricing(productRequest.ProductID)
			pricingCh <-  struct{ProductID int; FinalPrice float64; MarkUp float64; Discount float64; Err error}{
				ProductID: pricing.ProductID,
				FinalPrice: pricing.FinalPrice,
				MarkUp: pricing.Markup,
				Discount: pricing.Discount,
				Err: err,

			}

		}(&productRequest)


	}

	for range order.ProductRequests{
		availabilityResult := <- availabilityCh
		pricingResult  := <- pricingCh

		if availabilityResult.Err != nil {
			logger.Error().Err(availabilityResult.Err).Msgf("Error checking product stock for product %d", availabilityResult.ProductID)
			return  nil, availabilityResult.Err
		}

		if !availabilityResult.Available {
			logger.Warn().Msgf("product %d out of stock", availabilityResult.ProductID)
			return  nil, fmt.Errorf("product out of stock")
		}

		if pricingResult.Err != nil {
			logger.Error().Err(pricingResult.Err).Msgf("Error getting pricing for product  %d", pricingResult.ProductID)
			return  nil, pricingResult.Err
		}

		for _, productRequest := range order.ProductRequests{
			if productRequest.ProductID == availabilityResult.ProductID{
				productRequest.FinalPrice = float64(productRequest.Quantity)*pricingResult.FinalPrice
				productRequest.MarkUp = float64(productRequest.Quantity)* pricingResult.MarkUp
				productRequest.Discount = float64(productRequest.Quantity)* pricingResult.Discount 
			}


		}
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


