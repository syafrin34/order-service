package repository

import (
	"database/sql"
	"order-service/internal/entity"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB)*OrderRepository{
	return &OrderRepository{
		db: db,
	}
}

func (r *OrderRepository)GetOrderByID(id int)(*entity.OrderEntity, error){
	orderQuery := `SELECT id, user_id, quantity, total, status, total_mark_up, total_discount FROM orders WHERE id = ?`
	productrequestQuery := `SELECT product_id, quantity, mark_up, discount, final_price, FROM product_requests WHERE order_id = ?`

	order := &entity.OrderEntity{}
	err := r.db.QueryRow(orderQuery, id).Scan(&order.ID, &order.UserID, &order.Quantity, &order.Total, &order.Status, &order.TotalMarkUp, &order.TotalDiscount)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(productrequestQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next(){
		productRequest := entity.ProductRequest{}
		err := rows.Scan(&productRequest.ProductID, &productRequest.Quantity, &productRequest.MarkUp, &productRequest.Discount, &productRequest.FinalPrice)
		if err != nil {
			return nil, err
		}
		order.ProductRequests = append(order.ProductRequests, productRequest)
	}
	return  order, nil
}

func (r *OrderRepository)UpdateOrder(order *entity.OrderEntity)(*entity.OrderEntity, error){
   // start transaction
   tx, err := r.db.Begin()  
   if err != nil {
	return  nil, err
   }

   // update order
   orderQuery := `UPDATE orders SET user_id = ?, quantity = ?, total = ?, status = ?, total_mark_up = ?, total_discount = ? WHERE id = ?`
   _, err = tx.Exec(orderQuery, order.UserID, order.Quantity, order.Total, order.Status, order.TotalMarkUp, order.TotalDiscount, order.ID)
   if err != nil {
	tx.Rollback()
	return  nil, err
   } 

   // DELETE existing product request
   deleteQuery := `DELETE FROM product_requests WHERE order_id = ?`
   _, err = tx.Exec(deleteQuery, order.ID)
   if err != nil {
	tx.Rollback()
	return  nil, err
   } 

   // insert product Request
	productQuery := `INSERT INTO product_requests(order_id, product_id, quantity, mark_up, discount, final_price)VALUES(?, ?, ?, ?, ?, ?)`

	for _, product := range order.ProductRequests{
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

func (r *OrderRepository)CreateOrder(order *entity.OrderEntity)(*entity.OrderEntity, error){
	// start transaction
   tx, err := r.db.Begin()  
   if err != nil {
	return  nil, err
   }

   // insert order
   orderQuery := `INSERT INTO orders(user_id, quantity, total, status, total_mark_up, total_discount)VALUES(?, ?, ?, ?, ?, ?)`
   res, err := tx.Exec(orderQuery, order.UserID, order.Quantity, order.Total, order.Status, order.TotalMarkUp, order.TotalDiscount)
   if err != nil {
	tx.Rollback()
	return  nil, err
   } 

   orderID, err := res.LastInsertId()
   if err != nil {
	tx.Rollback()
	return  nil, err
   } 

   // insert product Request
	productQuery := `INSERT INTO product_requests(order_id, product_id, quantity, mark_up, discount, final_price)VALUES(?, ?, ?, ?, ?, ?)`

	for _, product := range order.ProductRequests{
		_, err := tx.Exec(productQuery, orderID, product.ProductID, product.Quantity, product.MarkUp, product.Discount, product.FinalPrice)
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
	order.ID = int(orderID)
	return order, nil
}
func (r *OrderRepository)DeleteOrder(id int)error{
	 // start transaction
	tx, err := r.db.Begin()  
	if err != nil {
		return   err
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
		return  err
	}


	return  nil
}

func (r *OrderRepository)UpdateOrderStatus(id int, status string)error{
   // start transaction
	query := 	`UPDATE orders SET status = ? WHERE id = ?`
	_, err := r.db.Exec(query, status, id)
	if err != nil {
		return err
	}

	return nil
}