package entity

type Pricing struct {
	ProductID int `json:"product_id"`
	Markup float64 `json:"markup"`
	Discount float64 `json:"discount"`
	FinalPrice float64 `json:"final_price"`
}