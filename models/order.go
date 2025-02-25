package models

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type Order struct {
	ProductName string
	Quantity    string
}

type OrderDetails struct {
	ID       int
	Status   string
	Username string
	Items    []Order
}

var (
	currentOrders   = make(map[int64][]Order)
	creatingOrder   = make(map[int64]bool)
	waitingQuantity = make(map[int64]bool)
	db              *sql.DB
)
