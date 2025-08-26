// Package repository
package repository

import (
	"database/sql"
	"order-service/internal/entity"
	"order-service/internal/sharding"
)

type OrderRepository struct {
	//db *sql.DB
	dbShards []*sql.DB
	router   *sharding.ShardRouter
}

func NewOrderRepository(dbShards []*sql.DB, router *sharding.ShardRouter) *OrderRepository {
	return &OrderRepository{
		dbShards: dbShards,
		router:   router,
	}
}

func (r *OrderRepository) GetOrderByID(id int) (*entity.OrderEntity, error) {
	orderQuery := `SELECT id, user_id, quantity, total, status, total_mark_up, total_discount, order_id FROM orders WHERE id = ?`
	productrequestQuery := `SELECT product_id, quantity, mark_up, discount, final_price, FROM product_requests WHERE order_id = ?`

	dbindex := r.router.GetShard(id)
	db := r.dbShards[dbindex]

	order := &entity.OrderEntity{}
	err := db.QueryRow(orderQuery, id).Scan(&order.ID, &order.UserID, &order.OrderID, &order.Quantity, &order.Total, &order.Status, &order.TotalMarkUp, &order.TotalDiscount)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(productrequestQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		productRequest := entity.ProductRequest{}
		err := rows.Scan(&productRequest.ProductID, &productRequest.Quantity, &productRequest.MarkUp, &productRequest.Discount, &productRequest.FinalPrice)
		if err != nil {
			return nil, err
		}
		order.ProductRequests = append(order.ProductRequests, productRequest)
	}
	return order, nil
}

func (r *OrderRepository) UpdateOrder(order *entity.OrderEntity) (*entity.OrderEntity, error) {
	dbindex := r.router.GetShard(order.OrderID)
	db := r.dbShards[dbindex]
	// start transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	// update order
	orderQuery := `UPDATE orders SET user_id = ?, quantity = ?, total = ?, status = ?, total_mark_up = ?, total_discount = ? WHERE id = ?`
	_, err = tx.Exec(orderQuery, order.UserID, order.Quantity, order.Total, order.Status, order.TotalMarkUp, order.TotalDiscount, order.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// DELETE existing product request
	deleteQuery := `DELETE FROM product_requests WHERE order_id = ?`
	_, err = tx.Exec(deleteQuery, order.ID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// insert product Request
	productQuery := `INSERT INTO product_requests(order_id, product_id, quantity, mark_up, discount, final_price)VALUES(?, ?, ?, ?, ?, ?)`

	for _, product := range order.ProductRequests {
		_, err := tx.Exec(productQuery, order.ID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// commit transactions
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *OrderRepository) CreateOrder(order *entity.OrderEntity) (*entity.OrderEntity, error) {
	dbindex := r.router.GetShard(order.OrderID)
	db := r.dbShards[dbindex]
	// start transaction
	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}

	//insert order
	orderQuery := `INSERT INTO orders(user_id, order_id, quantity, total, status, total_mark_up, total_discount)VALUES(?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.Exec(orderQuery, order.UserID, order.OrderID, order.Quantity, order.Total, order.Status, order.TotalMarkUp, order.TotalDiscount)

	if err != nil {
		tx.Rollback()
		return nil, err
	}

	orderID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// insert product Request
	// productQuery := `INSERT INTO product_requests(order_id, product_id, quantity, mark_up, discount, final_price)VALUES(?, ?, ?, ?, ?, ?)`

	// for _, product := range order.ProductRequests {
	// 	_, err := tx.Exec(productQuery, orderID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
	// 	if err != nil {
	// 		tx.Rollback()
	// 		return nil, err
	// 	}
	// }

	// insert batch request
	productQuery := `INSERT INTO orders(user_id, order_id, quantity, total, status, total_mark_up, total_discount)VALUES `
	var values []interface{}
	for _, product := range order.ProductRequests {
		productQuery += "(?, ?, ?, ?, ?, ?),"
		values = append(values, orderID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
	}

	// remove the trailing comma
	productQuery = productQuery[:len(productQuery)-1]

	// execute the query batch insert
	_, err = tx.Exec(productQuery, values...)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	// commit transactions
	err = tx.Commit()
	if err != nil {
		return nil, err
	}
	order.ID = int(orderID)
	return order, nil
}
func (r *OrderRepository) DeleteOrder(id int) error {
	dbindex := r.router.GetShard(id)
	db := r.dbShards[dbindex]
	// start transaction
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	// DELETE existing product request
	deleteQuery := `DELETE FROM product_requests WHERE order_id = ?`
	_, err = tx.Exec(deleteQuery, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// DELETE order
	deleteOrderQuery := `DELETE FROM orders WHERE id = ?`
	_, err = tx.Exec(deleteOrderQuery, id)
	if err != nil {
		tx.Rollback()
		return err
	}

	// commit transactions
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *OrderRepository) UpdateOrderStatus(id int, status string) error {
	dbindex := r.router.GetShard(id)
	db := r.dbShards[dbindex]
	// start transaction
	query := `UPDATE orders SET status = ? WHERE id = ?`
	_, err := db.Exec(query, status, id)
	if err != nil {
		return err
	}

	return nil
}
