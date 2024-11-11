package database

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/context"
)

func GetTotalBalance(pool *pgxpool.Pool, userID int) (float64, error) {
	query := `
		SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) AS total_balance
		FROM transactions
		WHERE user_id = $1`
	var totalBalance float64
	err := pool.QueryRow(context.Background(), query, userID).Scan(&totalBalance)
	if err != nil {
		return 0, fmt.Errorf("error fetching total balance: %v", err)
	}
	return totalBalance, nil
}

func GetMonthlyExpenses(pool *pgxpool.Pool, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT EXTRACT(MONTH FROM transaction_date) AS month, SUM(amount) AS total
		FROM transactions 
		WHERE user_id = $1 AND type = 'expense'
		GROUP BY month
		ORDER BY month`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching monthly expenses: %v", err)
	}
	defer rows.Close()

	var expenses []map[string]interface{}
	for rows.Next() {
		var month int
		var total float64
		if err := rows.Scan(&month, &total); err != nil {
			return nil, err
		}
		expenses = append(expenses, map[string]interface{}{
			"month": month,
			"total": total,
		})
	}
	return expenses, nil
}

func GetIncomeExpenseSummary(pool *pgxpool.Pool, userID int) (map[string]float64, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END), 0) AS total_expense
		FROM transactions
		WHERE user_id = $1`
	var totalIncome, totalExpense float64
	err := pool.QueryRow(context.Background(), query, userID).Scan(&totalIncome, &totalExpense)
	if err != nil {
		return nil, fmt.Errorf("error fetching income and expense summary: %v", err)
	}
	return map[string]float64{
		"income":  totalIncome,
		"expense": totalExpense,
	}, nil
}

func GetCategoryWiseExpenses(pool *pgxpool.Pool, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT c.name AS category, COALESCE(SUM(t.amount), 0) AS total
		FROM transactions t
		JOIN categories c ON t.category_id = c.id
		WHERE t.user_id = $1 AND t.type = 'expense'
		GROUP BY c.name
		ORDER BY total DESC`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("error fetching category-wise expenses: %v", err)
	}
	defer rows.Close()

	var expenses []map[string]interface{}
	for rows.Next() {
		var category string
		var total float64
		if err := rows.Scan(&category, &total); err != nil {
			return nil, err
		}
		expenses = append(expenses, map[string]interface{}{
			"category": category,
			"total":    total,
		})
	}
	return expenses, nil
}
