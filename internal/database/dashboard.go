package database

import (
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/context"
	"sort"
)

func GetTotalBalance(pool *pgxpool.Pool, userID int) (float64, error) {
	query := `
		SELECT COALESCE(SUM(CASE WHEN type = 'income' THEN amount ELSE -amount END), 0) AS total_balance
		FROM transactions
		WHERE user_id = $1 
		AND DATE_TRUNC('month', transaction_date) = DATE_TRUNC('month', CURRENT_DATE)`
	var totalBalance float64
	err := pool.QueryRow(context.Background(), query, userID).Scan(&totalBalance)
	if err != nil {
		return 0, fmt.Errorf("ошибка при получении общего баланса: %v", err)
	}
	return totalBalance, nil
}

func GetMonthlyExpenses(pool *pgxpool.Pool, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT EXTRACT(MONTH FROM transaction_date) AS month, SUM(amount) AS total
		FROM (
			SELECT transaction_date, amount
			FROM transactions
			WHERE user_id = $1 AND type = 'expense'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
			UNION ALL
			SELECT transaction_date, amount
			FROM transactionhistory
			WHERE user_id = $1 AND type = 'expense'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
		) AS combined
		GROUP BY month
		ORDER BY month`

	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении месячных расходов: %v", err)
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
		FROM (
			SELECT amount, type, transaction_date
			FROM transactions
			WHERE user_id = $1
			AND DATE_TRUNC('month', transaction_date) = DATE_TRUNC('month', CURRENT_DATE)
			UNION ALL
			SELECT amount, type, transaction_date
			FROM transactionhistory
			WHERE user_id = $1
			AND DATE_TRUNC('month', transaction_date) = DATE_TRUNC('month', CURRENT_DATE)
		) AS combined`
	var totalIncome, totalExpense float64
	err := pool.QueryRow(context.Background(), query, userID).Scan(&totalIncome, &totalExpense)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении доходов и расходов: %v", err)
	}
	return map[string]float64{
		"income":  totalIncome,
		"expense": totalExpense,
	}, nil
}

func GetCategoryWiseExpenses(pool *pgxpool.Pool, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT c.name AS category, COALESCE(SUM(t.amount), 0) AS total
		FROM (
			SELECT category_id, amount, transaction_date
			FROM transactions
			WHERE user_id = $1 AND type = 'expense'
			AND DATE_TRUNC('month', transaction_date) = DATE_TRUNC('month', CURRENT_DATE)
			UNION ALL
			SELECT category_id, amount, transaction_date
			FROM transactionhistory
			WHERE user_id = $1 AND type = 'expense'
			AND DATE_TRUNC('month', transaction_date) = DATE_TRUNC('month', CURRENT_DATE)
		) AS t
		JOIN categories c ON t.category_id = c.id
		GROUP BY c.name
		ORDER BY total DESC`
	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении расходов по категориям: %v", err)
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

func GetMonthlyIncome(pool *pgxpool.Pool, userID int) ([]map[string]interface{}, error) {
	query := `
		SELECT EXTRACT(MONTH FROM transaction_date) AS month, SUM(amount) AS total
		FROM (
			SELECT transaction_date, amount
			FROM transactions
			WHERE user_id = $1 AND type = 'income'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
			UNION ALL
			SELECT transaction_date, amount
			FROM transactionhistory
			WHERE user_id = $1 AND type = 'income'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
		) AS combined
		GROUP BY month
		ORDER BY month`

	rows, err := pool.Query(context.Background(), query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении месячных доходов: %v", err)
	}
	defer rows.Close()

	var income []map[string]interface{}
	for rows.Next() {
		var month int
		var total float64
		if err := rows.Scan(&month, &total); err != nil {
			return nil, err
		}
		income = append(income, map[string]interface{}{
			"month": month,
			"total": total,
		})
	}
	return income, nil
}

func GetMonthlyIncomeAndExpenses(pool *pgxpool.Pool, userID int) ([]map[string]interface{}, error) {
	expenseQuery := `
		SELECT EXTRACT(MONTH FROM transaction_date) AS month, SUM(amount) AS total
		FROM (
			SELECT transaction_date, amount
			FROM transactions
			WHERE user_id = $1 AND type = 'expense'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
			UNION ALL
			SELECT transaction_date, amount
			FROM transactionhistory
			WHERE user_id = $1 AND type = 'expense'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
		) AS combined
		GROUP BY month
		ORDER BY month`

	incomeQuery := `
		SELECT EXTRACT(MONTH FROM transaction_date) AS month, SUM(amount) AS total
		FROM (
			SELECT transaction_date, amount
			FROM transactions
			WHERE user_id = $1 AND type = 'income'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
			UNION ALL
			SELECT transaction_date, amount
			FROM transactionhistory
			WHERE user_id = $1 AND type = 'income'
			AND DATE_PART('year', transaction_date) = DATE_PART('year', CURRENT_DATE)
		) AS combined
		GROUP BY month
		ORDER BY month`

	expenseRows, err := pool.Query(context.Background(), expenseQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении месячных расходов: %v", err)
	}
	defer expenseRows.Close()

	incomeRows, err := pool.Query(context.Background(), incomeQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении месячных доходов: %v", err)
	}
	defer incomeRows.Close()

	monthlyData := make(map[int]map[string]float64)

	for expenseRows.Next() {
		var month int
		var total float64
		if err := expenseRows.Scan(&month, &total); err != nil {
			return nil, err
		}
		if _, exists := monthlyData[month]; !exists {
			monthlyData[month] = map[string]float64{"income": 0, "expense": 0}
		}
		monthlyData[month]["expense"] = total
	}

	for incomeRows.Next() {
		var month int
		var total float64
		if err := incomeRows.Scan(&month, &total); err != nil {
			return nil, err
		}
		if _, exists := monthlyData[month]; !exists {
			monthlyData[month] = map[string]float64{"income": 0, "expense": 0}
		}
		monthlyData[month]["income"] = total
	}

	var result []map[string]interface{}
	for month, data := range monthlyData {
		result = append(result, map[string]interface{}{
			"month":   month,
			"income":  data["income"],
			"expense": data["expense"],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i]["month"].(int) < result[j]["month"].(int)
	})

	return result, nil
}
