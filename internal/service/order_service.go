// Package service
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"order-service/internal/entity"
	"order-service/internal/repository"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

type OrderService struct {
	orderRepo         repository.OrderRepository
	productServiceURL string
	pricingServiceURL string
	kafkaWriter       *kafka.Writer
	rdb               *redis.Client
}

func NewOrderService(orderRepo repository.OrderRepository, productServiceURL, pricingServiceURL string, kafkaWriter *kafka.Writer, rdb *redis.Client) *OrderService {
	return &OrderService{
		orderRepo:         orderRepo,
		productServiceURL: pricingServiceURL,
		pricingServiceURL: pricingServiceURL,
		kafkaWriter:       kafkaWriter,
		rdb:               rdb,
	}
}

func (o *OrderService) CreateOrder(ctx context.Context, order *entity.OrderEntity) (*entity.OrderEntity, error) {

	// get the idempotent key from order
	validate, err := o.validateIdempotentKey(ctx, order.IdempotentKey)
	if err != nil {
		return nil, err
	}
	if !validate {
		return nil, errors.New("idempotent key already exists")
	}

	order.OrderID = randomOrderID()

	availabilityCh := make(chan struct {
		ProductID int
		Available bool
		Err       error
	}, len(order.ProductRequests))

	pricingCh := make(chan struct {
		ProductID  int
		FinalPrice float64
		MarkUp     float64
		Discount   float64
		Err        error
	}, len(order.ProductRequests))

	for _, productRequest := range order.ProductRequests {

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
			available, err := o.checkProductStock(ctx, productRequest.ProductID, productRequest.Quantity)
			availabilityCh <- struct {
				ProductID int
				Available bool
				Err       error
			}{
				ProductID: productRequest.ProductID,
				Available: available,
				Err:       err,
			}
		}(&productRequest)

		// go routine untuk pricing

		go func(productRequest *entity.ProductRequest) {
			pricing, err := o.GetPricing(ctx, productRequest.ProductID)
			pricingCh <- struct {
				ProductID  int
				FinalPrice float64
				MarkUp     float64
				Discount   float64
				Err        error
			}{
				ProductID:  pricing.ProductID,
				FinalPrice: pricing.FinalPrice,
				MarkUp:     pricing.Markup,
				Discount:   pricing.Discount,
				Err:        err,
			}

		}(&productRequest)

	}

	for range order.ProductRequests {
		availabilityResult := <-availabilityCh
		pricingResult := <-pricingCh

		if availabilityResult.Err != nil {
			logger.Error().Err(availabilityResult.Err).Msgf("Error checking product stock for product %d", availabilityResult.ProductID)
			return nil, availabilityResult.Err
		}

		if !availabilityResult.Available {
			logger.Warn().Msgf("product %d out of stock", availabilityResult.ProductID)
			return nil, fmt.Errorf("product out of stock")
		}

		if pricingResult.Err != nil {
			logger.Error().Err(pricingResult.Err).Msgf("Error getting pricing for product  %d", pricingResult.ProductID)
			return nil, pricingResult.Err
		}

		for _, productRequest := range order.ProductRequests {
			if productRequest.ProductID == availabilityResult.ProductID {
				productRequest.FinalPrice = float64(productRequest.Quantity) * pricingResult.FinalPrice
				productRequest.MarkUp = float64(productRequest.Quantity) * pricingResult.MarkUp
				productRequest.Discount = float64(productRequest.Quantity) * pricingResult.Discount
			}

		}
	}

	// calculate order total
	order.Total = 0
	for _, productRequest := range order.ProductRequests {
		order.Total += productRequest.FinalPrice
	}

	createdOrder, err := o.orderRepo.CreateOrder(order)
	if err != nil {
		logger.Error().Err(err).Msgf("Error creating order")
		return nil, err
	}
	// if env is set to test, return
	if os.Getenv("ENV") == "test" {
		return createdOrder, nil
	}

	err = o.publishorderEvent(ctx, createdOrder, "created")
	if err != nil {
		return nil, err
	}
	return createdOrder, nil
}
func (o *OrderService) UpdateOrder(ctx context.Context, order *entity.OrderEntity) (*entity.OrderEntity, error) {
	if order.Status == "paid" {
		// check product availability
		for _, productRequest := range order.ProductRequests {
			available, err := o.checkProductStock(ctx, productRequest.ProductID, productRequest.Quantity)
			if err != nil {
				logger.Error().Err(err).Msgf("error checking product stock for product %d", productRequest.ProductID)
				return nil, err
			}
			if !available {
				logger.Warn().Msgf("Product %d out of stock", productRequest.ProductID)
				return nil, fmt.Errorf("product out of stock")
			}
		}

	}

	updateOrder, err := o.orderRepo.UpdateOrder(order)
	if err != nil {
		logger.Error().Err(err).Msgf("Error updating order")
		return nil, err
	}

	err = o.publishorderEvent(ctx, updateOrder, "updated")
	if err != nil {
		return nil, err
	}

	return updateOrder, nil
}
func (o *OrderService) CancelOrder(ctx context.Context, id int) (*entity.OrderEntity, error) {
	order, err := o.orderRepo.GetOrderByID(id)
	if err != nil {
		logger.Error().Err(err).Msgf("Error getting order by ID %d", id)
		return nil, err
	}
	order.Status = "cancelled"

	updateOrder, err := o.orderRepo.UpdateOrder(order)
	if err != nil {
		logger.Error().Err(err).Msgf("Error updating order")
		return nil, err
	}
	err = o.publishorderEvent(ctx, updateOrder, "cancelled")
	if err != nil {
		return nil, err
	}
	return updateOrder, nil
}

func (o *OrderService) checkProductStock(ctx context.Context, productID int, quantity int) (bool, error) {
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
		return false, err
	}

	availableStock := stockData["stock"]
	return availableStock >= quantity, nil

}

func (o *OrderService) GetPricing(ctx context.Context, productID int) (*entity.Pricing, error) {
	resp, err := http.Get(fmt.Sprintf("%s/products/%d/pricing", o.pricingServiceURL, productID))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get pricing")
	}
	var pricing entity.Pricing
	if err := json.NewDecoder(resp.Body).Decode(&pricing); err != nil {
		return nil, err
	}
	return &pricing, nil
}

func (o *OrderService) publishorderEvent(ctx context.Context, order *entity.OrderEntity, key string) error {
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("order-%s-%d", key, order.ID)),
		Value: orderJSON,
	}
	err = o.kafkaWriter.WriteMessages(ctx, msg)
	if err != nil {
		return err
	}
	return nil
}

func (o *OrderService) validateIdempotentKey(ctx context.Context, key string) (bool, error) {
	// if env is set to test, return true
	if os.Getenv("ENV") == "test" {
		return true, nil
	}
	// check if the key exists in the redis cache
	// if it exists, return false
	redisKey := fmt.Sprintf("idempotent-key:%s", key)
	val, err := o.rdb.Get(ctx, redisKey).Result()
	if err != nil && !errors.Is(err, redis.Nil) {
		return false, err
	}

	if val != "" {
		return false, errors.New("idempotent key already exists")
	}

	// if it doesn't exist, add the key to the cache with a TTL of 24 hours
	// and return true
	err = o.rdb.Set(ctx, redisKey, "exists", 24*time.Hour).Err()

	return true, nil
}

func randomOrderID() int {
	return 1000 + rand.Intn(1000)
}
