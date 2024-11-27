package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/utils"
	"net/http"
	"strconv"
)

func ConvertCurrencyHandler(c *gin.Context) {
	from := c.DefaultQuery("from", "")        // Валюта для конвертации (from)
	to := c.DefaultQuery("to", "")            // Валюта в которую конвертировать (to)
	amountStr := c.DefaultQuery("amount", "") // Сумма для конвертации (amount)

	if from == "" || to == "" || amountStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Отсутствуют параметры 'from', 'to' или 'amount'"})
		return
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверное значение 'amount'"})
		return
	}

	// Получение актуальных курсов валют
	rates, err := utils.GetCachedRates() // Использование общей функции
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения курсов валют"})
		return
	}

	fromRate, fromExists := rates[from]
	toRate, toExists := rates[to]
	if !fromExists || !toExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Валюта не найдена"})
		return
	}

	result := amount * (toRate / fromRate)
	response := map[string]interface{}{
		"from":   from,
		"to":     to,
		"amount": amount,
		"result": result,
	}

	c.JSON(http.StatusOK, response)
}
